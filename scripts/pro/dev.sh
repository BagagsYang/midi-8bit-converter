#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
STAGE_DIR="${PRO_DEV_STAGE:-$ROOT/dist/pro/dev-stage}"
PORT="${PORT:-8000}"

"$ROOT/scripts/pro/assemble.sh" "$STAGE_DIR" >/dev/null

cleanup() {
	if [ -n "${BACKEND_PID:-}" ]; then
		kill "$BACKEND_PID" 2>/dev/null || true
	fi
}
trap cleanup EXIT INT TERM

(
	cd "$STAGE_DIR/backend"
	PORT="$PORT" go run ./cmd/pro-server
) &
BACKEND_PID="$!"

(
	cd "$STAGE_DIR/frontend"
	if [ ! -d node_modules ]; then
		npm ci
	fi
	npm run dev
)
