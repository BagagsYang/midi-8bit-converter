# Core

- Monorepo centered on production web app.
- Production frontend: `frontend/` Vue/Vite app, served from Vite `dist`, talks to stable `/api/*` contract.
- Primary backend: `backend/` Go API/workspace/synthesis service with compatibility routes and Go renderer.
- Legacy/parity areas: `legacy/web-flask/`, `legacy/python-renderer/`, `legacy/native/macos/`, `legacy/native/windows/`. Treat as fallback/reference unless task explicitly targets them.
- Shared assets: `assets/previews/` waveform preview WAVs.
- Deployment docs/configs: production Go+Vue under `deploy/production/`; legacy Flask fallback under `deploy/web-flask/`, `compose.web.yml`, and `docs/`.
- Read frontend details in `mem:frontend/core`; backend details in `mem:backend/core`; legacy/parity guidance in `mem:legacy/core`.
- Project-wide tech stack and version pins: `mem:tech_stack`.
- Commands: `mem:suggested_commands`; completion checks: `mem:task_completion`; conventions: `mem:conventions`.
