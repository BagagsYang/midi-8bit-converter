# Repository Guidelines

See [CLAUDE.md](./CLAUDE.md) for comprehensive project context, architecture, and conventions.

## Project Structure & Module Organization

OctaBit is a monorepo focused on the web app. Work from the repository root unless a subproject README says otherwise.

- `frontend/`: production Vue/Vite frontend. It talks to the stable `/api/*` contract and is served in production from the Vite `dist` build.
- `backend/`: primary Go backend API, workspace/synthesis service, compatibility routes, Go renderer, and Go tests.
- `legacy/web-flask/`: legacy Flask backend/API and Flask-rendered frontend fallback retained for parity fixtures and fallback reference.
- `legacy/python-renderer/`: canonical Python MIDI-to-WAV parity reference and renderer tests.
- `assets/previews/`: shared waveform preview WAV files.
- `deploy/production/`: non-Docker production deployment notes, Caddy examples, and Vue production helper script.
- `deploy/web-flask/`, `compose.web.yml`, and `docs/`: legacy Flask fallback deployment and API documentation.
- `legacy/native/macos/` and `legacy/native/windows/`: deprecated/paused native apps retained for reference.

## Build, Test, and Development Commands

[CLAUDE.md](./CLAUDE.md) is the authoritative command reference — use it for the full set (test invocation, parity fixtures, Docker, legacy fallback). The essentials:

```bash
python3 -m venv .venv
./.venv/bin/python3 -m pip install -r legacy/web-flask/requirements.txt
./.venv/bin/python3 -m pip install -r legacy/python-renderer/requirements.txt

cd backend && PORT=8000 go run ./cmd/server
cd backend && go test ./...

cd frontend && npm ci
cd frontend && npm run dev
cd frontend && npm run build
```

## Coding Style & Naming Conventions

Prefer small, localized changes. Keep runtime synthesis behavior in `backend/internal/renderer/` and parity reference behavior in `legacy/python-renderer/`. For production Vue UI strings, use `frontend/src/i18n/*.json`; for legacy Flask-rendered UI strings, use `legacy/web-flask/i18n/*.json`. Keep English as the fallback and align keys across all catalogs you touch (locale list is maintained in [CLAUDE.md](./CLAUDE.md)). Use descriptive Go/Python names, TypeScript component names in PascalCase, and existing file naming patterns.

## Testing Guidelines

Run checks for the touched area and report skipped checks.

```bash
cd backend && go test ./...
cd frontend && npm run build
```

Run legacy Python tests only when touching `legacy/web-flask/`,
`legacy/python-renderer/`, or fixture-regeneration behavior.

Name Python tests `test_*.py`. For web API or localization changes, add render-level or endpoint assertions where practical.

## Commit & Pull Request Guidelines

Recent history uses short imperative messages and lightweight prefixes such as `feat:`, `fix:`, and `docs:`. Keep commits focused, for example `fix: prevent duplicate waveform layers`.

The public `bagags/octabit` repository is an OSS mirror of a private upstream monorepo, not an open contribution target. Pull requests are only for prior-arranged work and should include a clear summary, touched areas, user-facing or deployment impact, screenshots for visible UI changes, linked issues when relevant, and the exact checks run. Note source and license details for new dependencies, vendored assets, or generated media.

## Agent-Specific Instructions

Treat `frontend/` as the production frontend and `backend/` as the primary backend. Treat `legacy/web-flask/`, `legacy/python-renderer/`, and `legacy/native/*` as legacy/parity reference areas unless the task explicitly targets them. Preserve existing behavior unless the request asks for a UI, API, or renderer change.
