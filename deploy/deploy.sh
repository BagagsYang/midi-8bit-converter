#!/usr/bin/env bash
set -euo pipefail

# --- paths (override via env) ---
PUBLIC_REPO="${PUBLIC_REPO:-/home/deploy/octabit}"
PRO_REPO="${PRO_REPO:-/home/deploy/octabit-pro}"
BRANCH="${BRANCH:-main}"
GO_SERVICE="${GO_SERVICE:-octabit-web}"
CADDY_CONFIG="${CADDY_CONFIG:-/etc/caddy/Caddyfile}"
RELOAD_CADDY="${RELOAD_CADDY:-1}"

echo "=== pull public repo ($PUBLIC_REPO) ==="
cd "$PUBLIC_REPO"
git fetch --prune origin "$BRANCH"
git checkout "$BRANCH"
git reset --hard "origin/$BRANCH"

echo "=== pull pro repo ($PRO_REPO) ==="
cd "$PRO_REPO"
git fetch --prune origin "$BRANCH"
git checkout "$BRANCH"
git reset --hard "origin/$BRANCH"

echo "=== inject go pro entry point ==="
cp "$PRO_REPO/backend/cmd/pro-server/main.go" "$PUBLIC_REPO/backend/cmd/pro-server/main.go"

echo "=== build go backend ==="
cd "$PUBLIC_REPO/backend"
go build -o octabit-server ./cmd/pro-server

echo "=== clean up injected go files ==="
rm -rf "$PUBLIC_REPO/backend/cmd/pro-server"

echo "=== inject frontend overlays ==="
cp -r "$PRO_REPO/frontend/overlays/src/"* "$PUBLIC_REPO/frontend/src/"

echo "=== build frontend ==="
cd "$PUBLIC_REPO/frontend"
npm ci
npm run build

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
