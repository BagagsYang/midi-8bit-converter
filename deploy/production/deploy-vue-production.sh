#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-${REPO_DIR:-/home/deploy/octabit}}"
BRANCH="${BRANCH:-main}"
GO_SERVICE="${GO_SERVICE:-octabit-web}"
CADDY_CONFIG="${CADDY_CONFIG:-/etc/caddy/Caddyfile}"
RELOAD_CADDY="${RELOAD_CADDY:-1}"
PUBLIC_URL="${PUBLIC_URL:-https://octabit.cc}"

verify_frontend_locales() {
	local target="$1"
	local label="${2:-$target}"
	local missing=0
	local catalog locale marker

	for catalog in frontend/src/i18n/*.json; do
		locale="$(basename "$catalog" .json)"
		marker="\"toolbar.language_option.$locale\""
		if [ -d "$target" ]; then
			if ! grep -R -q "$marker" "$target"; then
				echo "Missing locale marker $marker in $label" >&2
				missing=1
			fi
		else
			if ! grep -q "$marker" "$target"; then
				echo "Missing locale marker $marker in $label" >&2
				missing=1
			fi
		fi
	done

	if [ "$missing" -ne 0 ]; then
		exit 1
	fi
}

cd "$APP_DIR"

git fetch --prune origin "$BRANCH"
git checkout "$BRANCH"
git reset --hard "origin/$BRANCH"
echo "Deploying $(git rev-parse --short HEAD) from $APP_DIR"

cd backend
go build -o octabit-server ./cmd/server
cd "$APP_DIR"

cd frontend
npm ci
npm run build
cd "$APP_DIR"
verify_frontend_locales frontend/dist/assets
local_asset_path="$(sed -n 's/.*src="\([^"]*\/assets\/index-[^"]*\.js\)".*/\1/p' frontend/dist/index.html | head -n 1)"
echo "Local Vite asset: ${local_asset_path:-not found}"

sudo systemctl restart "$GO_SERVICE"
sudo systemctl status "$GO_SERVICE" --no-pager --lines=20

curl -fsS http://127.0.0.1:8000/api/health

if [ "$RELOAD_CADDY" = "1" ]; then
	sudo caddy validate --config "$CADDY_CONFIG"
	sudo systemctl reload caddy
fi

if [ -n "$PUBLIC_URL" ]; then
	public_index="$(mktemp)"
	public_js="$(mktemp)"
	cleanup_public_check() {
		rm -f "$public_index" "$public_js"
	}
	trap cleanup_public_check EXIT

	curl -fsSL "$PUBLIC_URL/" -o "$public_index"
	asset_path="$(sed -n 's/.*src="\([^"]*\/assets\/index-[^"]*\.js\)".*/\1/p' "$public_index" | head -n 1)"
	if [ -z "$asset_path" ]; then
		echo "Could not find the Vite JavaScript asset in $PUBLIC_URL/" >&2
		exit 1
	fi
	public_asset_url="${PUBLIC_URL%/}${asset_path}"
	echo "Public Vite asset: $public_asset_url"
	if [ -n "$local_asset_path" ] && [ "$asset_path" != "$local_asset_path" ]; then
		echo "Public index references $asset_path, but the local build references $local_asset_path." >&2
		echo "Caddy may be serving a different root than $APP_DIR/frontend/dist, or an upstream cache may still be serving stale files." >&2
	fi
	curl -fsSL "$public_asset_url" -o "$public_js"
	verify_frontend_locales "$public_js" "$public_asset_url"
fi
