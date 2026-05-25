# Tech Stack

- OS context: Darwin/macOS, shell usually `zsh`.
- Backend: Go module `octabit/backend`, `go 1.25.0`, stdlib HTTP server plus packages including `gitlab.com/gomidi/midi/v2` and `modernc.org/sqlite`.
- Frontend: Vue `^3.5.13`, Vite `^7.1.0`, TypeScript `^5.9.3`, `vue-tsc`, Vitest, jsdom, npm lockfile.
- Legacy Flask/Python dependencies are installed from `legacy/web-flask/requirements.txt` and `legacy/python-renderer/requirements.txt` into repo-local `.venv` when needed.
- No project-wide install command; install only dependencies for the touched area.
