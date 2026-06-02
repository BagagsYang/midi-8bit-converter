#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_DIR="${PRO_DIST_DIR:-$ROOT/dist/pro}"
STAGE_DIR="${PRO_STAGE_DIR:-$DIST_DIR/stage}"
GOCACHE_DIR="${GOCACHE:-$DIST_DIR/gocache}"

mkdir -p "$DIST_DIR" "$GOCACHE_DIR"
"$ROOT/scripts/pro/assemble.sh" "$STAGE_DIR" >/dev/null

echo "=== pro backend tests ==="
(
	cd "$STAGE_DIR/backend"
	GOCACHE="$GOCACHE_DIR" go test ./...
)

echo "=== pro backend build ==="
(
	cd "$STAGE_DIR/backend"
	GOCACHE="$GOCACHE_DIR" go build -o "$DIST_DIR/octabit-server" ./cmd/pro-server
)

echo "=== pro frontend install/test/build ==="
(
	cd "$STAGE_DIR/frontend"
	npm ci
	npm test
	npm run build
)

rm -rf "$DIST_DIR/frontend-dist"
cp -R "$STAGE_DIR/frontend/dist" "$DIST_DIR/frontend-dist"

echo "Pro build artifacts:"
echo "  $DIST_DIR/octabit-server"
echo "  $DIST_DIR/frontend-dist"
