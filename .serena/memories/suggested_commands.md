# Suggested Commands

- Create repo Python env: `python3 -m venv .venv`.
- Install legacy Flask deps only when needed: `./.venv/bin/python3 -m pip install -r legacy/web-flask/requirements.txt`.
- Install legacy Python renderer deps only when needed: `./.venv/bin/python3 -m pip install -r legacy/python-renderer/requirements.txt`.
- Run Go backend: from `backend/`, `PORT=8000 go run ./cmd/server`.
- Test Go backend: from `backend/`, `go test ./...`.
- Install frontend deps: from `frontend/`, `npm ci`.
- Run frontend dev server: from `frontend/`, `npm run dev`.
- Build frontend: from `frontend/`, `npm run build`.
- Test frontend: from `frontend/`, `npm test` or `npm run test`.
- Prefer `rg` / `rg --files` for repository search.
