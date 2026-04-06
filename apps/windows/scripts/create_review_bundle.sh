#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WINDOWS_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${WINDOWS_ROOT}/../.." && pwd)"
BUNDLE_ROOT="${REPO_ROOT}/windows-review-bundle"
ARCHIVE_PATH="${REPO_ROOT}/windows-review-bundle.zip"

rm -rf "${BUNDLE_ROOT}" "${ARCHIVE_PATH}"
mkdir -p \
  "${BUNDLE_ROOT}/apps" \
  "${BUNDLE_ROOT}/core/python-renderer" \
  "${BUNDLE_ROOT}/assets/previews" \
  "${BUNDLE_ROOT}/.github/workflows"

rsync -a \
  --exclude '.DS_Store' \
  --exclude '__pycache__' \
  "${WINDOWS_ROOT}/" \
  "${BUNDLE_ROOT}/apps/windows/"

rsync -a \
  --exclude '.DS_Store' \
  --exclude '__pycache__' \
  "${REPO_ROOT}/core/python-renderer/" \
  "${BUNDLE_ROOT}/core/python-renderer/"

rsync -a \
  --exclude '.DS_Store' \
  "${REPO_ROOT}/assets/previews/" \
  "${BUNDLE_ROOT}/assets/previews/"

cp \
  "${REPO_ROOT}/.github/workflows/windows-release.yml" \
  "${BUNDLE_ROOT}/.github/workflows/"

cp \
  "${REPO_ROOT}/global.json" \
  "${BUNDLE_ROOT}/"

(cd "${REPO_ROOT}" && zip -r "$(basename "${ARCHIVE_PATH}")" "$(basename "${BUNDLE_ROOT}")" -x "*.DS_Store" "*/__pycache__/*")

echo "Created ${ARCHIVE_PATH}"
