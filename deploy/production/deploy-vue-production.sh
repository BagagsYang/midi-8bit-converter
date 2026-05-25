#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-${REPO_DIR:-/home/deploy/octabit}}"
BRANCH="${BRANCH:-main}"
GO_SERVICE="${GO_SERVICE:-octabit-web}"
CADDY_CONFIG="${CADDY_CONFIG:-/etc/caddy/Caddyfile}"
RELOAD_CADDY="${RELOAD_CADDY:-1}"

cd "$APP_DIR"

git fetch --prune origin "$BRANCH"
git checkout "$BRANCH"
git pull --ff-only origin "$BRANCH"

cd backend
go build -o octabit-server ./cmd/server
cd "$APP_DIR"

cd frontend
npm ci
npm run build
cd "$APP_DIR"

sudo systemctl restart "$GO_SERVICE"
sudo systemctl status "$GO_SERVICE" --no-pager --lines=20

curl -fsS http://127.0.0.1:8000/api/health

if [ "$RELOAD_CADDY" = "1" ]; then
	sudo caddy validate --config "$CADDY_CONFIG"
	sudo systemctl reload caddy
fi
