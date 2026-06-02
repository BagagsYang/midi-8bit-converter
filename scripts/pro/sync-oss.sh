#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OSS_REMOTE_URL="${OSS_REMOTE_URL:-https://github.com/bagags/octabit.git}"
OSS_BRANCH="${OSS_BRANCH:-main}"
SYNC_DRY_RUN="${SYNC_DRY_RUN:-0}"
SYNC_KEEP_WORKDIR="${SYNC_KEEP_WORKDIR:-0}"
SYNC_COMMIT_MESSAGE="${SYNC_COMMIT_MESSAGE:-chore: sync public mirror}"
WORK_DIR="${SYNC_WORK_DIR:-$(mktemp -d)}"
PUBLIC_CLONE="$WORK_DIR/public"
MIRROR_DIR="$WORK_DIR/mirror"

cleanup() {
	if [ "$SYNC_KEEP_WORKDIR" != "1" ] && [ -z "${SYNC_WORK_DIR:-}" ]; then
		rm -rf "$WORK_DIR"
	fi
}
trap cleanup EXIT

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

rm -rf "$PUBLIC_CLONE" "$MIRROR_DIR"
git clone --branch "$OSS_BRANCH" --single-branch "$OSS_REMOTE_URL" "$PUBLIC_CLONE"
mkdir -p "$MIRROR_DIR"

copy_path "$ROOT/.dockerignore" "$MIRROR_DIR/.dockerignore"
copy_path "$ROOT/.gitattributes" "$MIRROR_DIR/.gitattributes"
copy_path "$ROOT/.gitignore" "$MIRROR_DIR/.gitignore"
copy_path "$ROOT/AGENTS.md" "$MIRROR_DIR/AGENTS.md"
copy_path "$ROOT/CLA.md" "$MIRROR_DIR/CLA.md"
copy_path "$ROOT/CLAUDE.md" "$MIRROR_DIR/CLAUDE.md"
copy_path "$ROOT/CONTRIBUTING.md" "$MIRROR_DIR/CONTRIBUTING.md"
copy_path "$ROOT/CONTRIBUTING.zh-Hans.md" "$MIRROR_DIR/CONTRIBUTING.zh-Hans.md"
copy_path "$ROOT/LICENSE.md" "$MIRROR_DIR/LICENSE.md"
copy_path "$ROOT/README.md" "$MIRROR_DIR/README.md"
copy_path "$ROOT/README.zh-Hans.md" "$MIRROR_DIR/README.zh-Hans.md"
copy_path "$ROOT/compose.web.yml" "$MIRROR_DIR/compose.web.yml"
copy_path "$ROOT/global.json" "$MIRROR_DIR/global.json"

copy_dir_contents "$ROOT/CLA-signatures" "$MIRROR_DIR/CLA-signatures"
copy_dir_contents "$ROOT/assets" "$MIRROR_DIR/assets"
copy_dir_contents "$ROOT/backend" "$MIRROR_DIR/backend"
copy_dir_contents "$ROOT/docs" "$MIRROR_DIR/docs"
copy_dir_contents "$ROOT/frontend" "$MIRROR_DIR/frontend"
copy_dir_contents "$ROOT/legacy" "$MIRROR_DIR/legacy"

copy_dir_contents "$ROOT/deploy/production" "$MIRROR_DIR/deploy/production"
copy_dir_contents "$ROOT/deploy/web-flask" "$MIRROR_DIR/deploy/web-flask"

mkdir -p "$MIRROR_DIR/.github/workflows" "$MIRROR_DIR/scripts"
copy_path "$ROOT/.github/dependabot.yml" "$MIRROR_DIR/.github/dependabot.yml"
copy_path "$ROOT/.github/workflows/check-cla.yml" "$MIRROR_DIR/.github/workflows/check-cla.yml"
copy_path "$ROOT/.github/workflows/ci.yml" "$MIRROR_DIR/.github/workflows/ci.yml"
copy_path "$ROOT/.github/workflows/close-unsolicited-prs.yml" "$MIRROR_DIR/.github/workflows/close-unsolicited-prs.yml"
copy_path "$ROOT/.github/workflows/dependabot-auto-merge.yml" "$MIRROR_DIR/.github/workflows/dependabot-auto-merge.yml"
copy_path "$ROOT/.github/workflows/security.yml" "$MIRROR_DIR/.github/workflows/security.yml"
copy_path "$ROOT/scripts/generate_python_parity_fixtures.py" "$MIRROR_DIR/scripts/generate_python_parity_fixtures.py"

rsync -a --delete --exclude='.git/' "$MIRROR_DIR"/ "$PUBLIC_CLONE"/

if git -C "$PUBLIC_CLONE" diff --quiet && git -C "$PUBLIC_CLONE" diff --cached --quiet && [ -z "$(git -C "$PUBLIC_CLONE" status --short)" ]; then
	echo "Public mirror is already up to date."
	if [ "$SYNC_KEEP_WORKDIR" = "1" ]; then
		echo "Kept workdir: $WORK_DIR"
	fi
	exit 0
fi

git -C "$PUBLIC_CLONE" status --short

if [ "$SYNC_DRY_RUN" = "1" ]; then
	echo "Dry run only; not committing or pushing."
	if [ "$SYNC_KEEP_WORKDIR" = "1" ]; then
		echo "Kept workdir: $WORK_DIR"
	fi
	exit 0
fi

git -C "$PUBLIC_CLONE" config user.name "${OSS_SYNC_GIT_USER_NAME:-octabit-pro sync}"
git -C "$PUBLIC_CLONE" config user.email "${OSS_SYNC_GIT_USER_EMAIL:-octabit-pro-sync@users.noreply.github.com}"
git -C "$PUBLIC_CLONE" add -A
git -C "$PUBLIC_CLONE" commit -m "$SYNC_COMMIT_MESSAGE"
git -C "$PUBLIC_CLONE" push origin "HEAD:$OSS_BRANCH"
