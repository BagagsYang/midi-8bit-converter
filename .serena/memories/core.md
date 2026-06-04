# Core

- Monorepo centered on production web app.
- Production frontend: `frontend/` Vue/Vite app, served from Vite `dist`, talks to stable `/api/*` contract.
- Primary backend: `backend/` Go API/workspace/synthesis service with Go renderer.
- Shared assets: `assets/previews/` waveform preview WAVs.
- Deployment docs/configs: production Go+Vue under `deploy/production/`.
- Read frontend details in `mem:frontend/core`; backend details in `mem:backend/core`.
- Project-wide tech stack and version pins: `mem:tech_stack`.
- Commands: `mem:suggested_commands`; completion checks: `mem:task_completion`; conventions: `mem:conventions`.
