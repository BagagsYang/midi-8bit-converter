#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
STAGE_DIR="${1:-$ROOT/dist/pro/stage}"

case "$STAGE_DIR" in
	""|"/"|"$ROOT"|"$ROOT/")
		echo "Refusing unsafe stage directory: $STAGE_DIR" >&2
		exit 1
		;;
esac

copy_path() {
	local src="$1"
	local dest="$2"
	if [ -e "$src" ]; then
		mkdir -p "$(dirname "$dest")"
		cp -R "$src" "$dest"
	fi
}

copy_dir_contents() {
	local src="$1"
	local dest="$2"
	if [ -d "$src" ]; then
		mkdir -p "$dest"
		rsync -a --exclude='.DS_Store' --exclude='overlays/' "$src"/ "$dest"/
	fi
}

rm -rf "$STAGE_DIR"
mkdir -p "$STAGE_DIR"

copy_path "$ROOT/.dockerignore" "$STAGE_DIR/.dockerignore"
copy_path "$ROOT/.gitattributes" "$STAGE_DIR/.gitattributes"
copy_path "$ROOT/.gitignore" "$STAGE_DIR/.gitignore"
copy_path "$ROOT/AGENTS.md" "$STAGE_DIR/AGENTS.md"
copy_path "$ROOT/CLA.md" "$STAGE_DIR/CLA.md"
copy_path "$ROOT/CLAUDE.md" "$STAGE_DIR/CLAUDE.md"
copy_path "$ROOT/CONTRIBUTING.md" "$STAGE_DIR/CONTRIBUTING.md"
copy_path "$ROOT/CONTRIBUTING.zh-Hans.md" "$STAGE_DIR/CONTRIBUTING.zh-Hans.md"
copy_path "$ROOT/LICENSE.md" "$STAGE_DIR/LICENSE.md"
copy_path "$ROOT/README.md" "$STAGE_DIR/README.md"
copy_path "$ROOT/README.zh-Hans.md" "$STAGE_DIR/README.zh-Hans.md"
copy_path "$ROOT/global.json" "$STAGE_DIR/global.json"

copy_dir_contents "$ROOT/CLA-signatures" "$STAGE_DIR/CLA-signatures"
copy_dir_contents "$ROOT/assets" "$STAGE_DIR/assets"
copy_dir_contents "$ROOT/backend" "$STAGE_DIR/backend"
copy_dir_contents "$ROOT/docs" "$STAGE_DIR/docs"
copy_dir_contents "$ROOT/frontend" "$STAGE_DIR/frontend"
copy_dir_contents "$ROOT/legacy" "$STAGE_DIR/legacy"

copy_dir_contents "$ROOT/deploy/production" "$STAGE_DIR/deploy/production"
copy_dir_contents "$ROOT/deploy/web-flask" "$STAGE_DIR/deploy/web-flask"

mkdir -p "$STAGE_DIR/.github/workflows" "$STAGE_DIR/scripts"
copy_path "$ROOT/.github/dependabot.yml" "$STAGE_DIR/.github/dependabot.yml"
copy_path "$ROOT/.github/workflows/check-cla.yml" "$STAGE_DIR/.github/workflows/check-cla.yml"
copy_path "$ROOT/.github/workflows/ci.yml" "$STAGE_DIR/.github/workflows/ci.yml"
copy_path "$ROOT/.github/workflows/close-unsolicited-prs.yml" "$STAGE_DIR/.github/workflows/close-unsolicited-prs.yml"
copy_path "$ROOT/.github/workflows/dependabot-auto-merge.yml" "$STAGE_DIR/.github/workflows/dependabot-auto-merge.yml"
copy_path "$ROOT/.github/workflows/security.yml" "$STAGE_DIR/.github/workflows/security.yml"
copy_path "$ROOT/scripts/generate_python_parity_fixtures.py" "$STAGE_DIR/scripts/generate_python_parity_fixtures.py"

copy_dir_contents "$ROOT/overlays/backend" "$STAGE_DIR/backend"
copy_dir_contents "$ROOT/overlays/frontend/src" "$STAGE_DIR/frontend/src"

echo "$STAGE_DIR"
