#!/usr/bin/env bash
set -euo pipefail

# --- paths (override via env) ---
PUBLIC_REPO="${PUBLIC_REPO:-/home/deploy/octabit}"
PRO_REPO="${PRO_REPO:-/home/deploy/octabit-pro}"
BRANCH="${BRANCH:-feat/octabit-pro}"
GO_SERVICE="${GO_SERVICE:-octabit-web}"
OUTPUT_BIN="${OUTPUT_BIN:-$PUBLIC_REPO/backend/octabit-server}"
OUTPUT_DIST="${OUTPUT_DIST:-$PUBLIC_REPO/frontend/dist}"
CADDY_CONFIG="${CADDY_CONFIG:-/etc/caddy/Caddyfile}"
RELOAD_CADDY="${RELOAD_CADDY:-1}"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT
STAGED_APP="$WORK_DIR/octabit"

echo "=== update public repo ($PUBLIC_REPO) ==="
git -C "$PUBLIC_REPO" fetch --prune origin "$BRANCH"
git -C "$PUBLIC_REPO" checkout "$BRANCH"
git -C "$PUBLIC_REPO" reset --hard "origin/$BRANCH"

echo "=== update pro repo ($PRO_REPO) ==="
git -C "$PRO_REPO" fetch --prune origin "$BRANCH"
git -C "$PRO_REPO" checkout "$BRANCH"
git -C "$PRO_REPO" reset --hard "origin/$BRANCH"

echo "=== assemble staged build ==="
mkdir -p "$STAGED_APP"
git -C "$PUBLIC_REPO" archive "$BRANCH" | tar -x -C "$STAGED_APP"
mkdir -p "$STAGED_APP/backend/cmd/pro-server"
cp "$PRO_REPO/backend/cmd/pro-server/main.go" "$STAGED_APP/backend/cmd/pro-server/main.go"
if [ -d "$PRO_REPO/backend/overlays" ]; then
	cp -R "$PRO_REPO/backend/overlays/"* "$STAGED_APP/backend/"
fi
cp -R "$PRO_REPO/frontend/overlays/src/"* "$STAGED_APP/frontend/src/"

echo "=== build go backend ==="
cd "$STAGED_APP/backend"
go build -o "$WORK_DIR/octabit-server" ./cmd/pro-server
install -m 0755 "$WORK_DIR/octabit-server" "$OUTPUT_BIN"

echo "=== build frontend ==="
cd "$STAGED_APP/frontend"
npm ci
npm run build
rm -rf "$OUTPUT_DIST"
mkdir -p "$(dirname "$OUTPUT_DIST")"
cp -R dist "$OUTPUT_DIST"

echo "=== restart backend ==="
sudo systemctl restart "$GO_SERVICE"
sudo systemctl status "$GO_SERVICE" --no-pager --lines=20

echo "=== health check ==="
curl -fsS http://127.0.0.1:8000/api/health

if [ "$RELOAD_CADDY" = "1" ]; then
	echo "=== reload caddy ==="
	sudo caddy validate --config "$CADDY_CONFIG"
	sudo systemctl reload caddy
fi

echo "=== deploy complete ==="
