#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
VUE_DIR="${PROJECT_ROOT}/frontend"
FLASK_BACKEND_URL="http://127.0.0.1:8000"
VUE_FRONTEND_URL="http://127.0.0.1:5173"

if [ ! -d "${PROJECT_ROOT}/.venv" ]; then
    echo "ERROR: Virtual environment (.venv) not found in the project folder."
    echo "Please read the 'User_Guide.txt' for setup instructions."
    echo "Press any key to exit..."
    read -n 1
    exit 1
fi

if [ ! -x "${PROJECT_ROOT}/.venv/bin/python3" ]; then
    echo "ERROR: ${PROJECT_ROOT}/.venv/bin/python3 was not found."
    echo "Please recreate the virtual environment and reinstall the dependencies."
    echo "Press any key to exit..."
    read -n 1
    exit 1
fi

if ! command -v npm >/dev/null 2>&1; then
    echo "ERROR: npm was not found. Install Node.js and npm before launching the Vue frontend."
    echo "Press any key to exit..."
    read -n 1
    exit 1
fi

if [ ! -d "${VUE_DIR}/node_modules" ]; then
    echo "ERROR: Vue dependencies are not installed."
    echo "Run this once from the repository root:"
    echo "  cd frontend && npm ci"
    echo "Press any key to exit..."
    read -n 1
    exit 1
fi

echo "Starting OctaBit..."
echo "------------------------------------------------"
echo "Flask API backend: ${FLASK_BACKEND_URL}"
echo "Vue frontend:      ${VUE_FRONTEND_URL}"
echo "------------------------------------------------"
echo "TO STOP: Press Ctrl+C in this window."
echo "------------------------------------------------"

cleanup() {
    if [ -n "${FLASK_PID:-}" ] && kill -0 "${FLASK_PID}" >/dev/null 2>&1; then
        kill "${FLASK_PID}" >/dev/null 2>&1 || true
    fi
}
trap cleanup EXIT INT TERM

cd "${PROJECT_ROOT}"
PORT=8000 WEB_FLASK_OPEN_BROWSER=0 "${PROJECT_ROOT}/.venv/bin/python3" legacy/web-flask/app.py &
FLASK_PID=$!

(sleep 3 && open "${VUE_FRONTEND_URL}") &

cd "${VUE_DIR}"
npm run dev
