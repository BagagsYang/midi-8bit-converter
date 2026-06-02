#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-/home/deploy/octabit-pro}"
BRANCH="${BRANCH:-main}"
GO_SERVICE="${GO_SERVICE:-octabit-web}"
OUTPUT_BIN="${OUTPUT_BIN:-$APP_ROOT/backend/octabit-server}"
OUTPUT_DIST="${OUTPUT_DIST:-$APP_ROOT/frontend/dist}"
CADDY_CONFIG="${CADDY_CONFIG:-/etc/caddy/Caddyfile}"
RELOAD_CADDY="${RELOAD_CADDY:-1}"

echo "=== update private monorepo ($APP_ROOT) ==="
git -C "$APP_ROOT" fetch --prune origin "$BRANCH"
git -C "$APP_ROOT" checkout "$BRANCH"
git -C "$APP_ROOT" reset --hard "origin/$BRANCH"

echo "=== build pro artifacts ==="
"$APP_ROOT/scripts/pro/build.sh"

echo "=== install backend binary ==="
install -m 0755 "$APP_ROOT/dist/pro/octabit-server" "$OUTPUT_BIN"

echo "=== install frontend dist ==="
rm -rf "$OUTPUT_DIST"
mkdir -p "$(dirname "$OUTPUT_DIST")"
cp -R "$APP_ROOT/dist/pro/frontend-dist" "$OUTPUT_DIST"

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
