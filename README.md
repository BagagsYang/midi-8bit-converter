<p align="center">
    <img src="https://github.com/user-attachments/files/27846479/octabit_icon.tiff" alt="OctaBit icon">
</p>

# OctaBit

Language/语言: English | [简体中文](./README.zh-CN.md)

OctaBit is a browser-based tool for converting MIDI files into 8-bit style WAV audio. The public service is <https://octabit.cc>.

The production web frontend is the Vue 3 app in `frontend/`. The primary backend is the Go service in `backend/`, which implements the stable `/api/*` contract, legacy `/synthesise*` compatibility routes, workspace storage, synthesis jobs, and the Go MIDI-to-WAV renderer. The older Flask backend and Python renderer are retained under `legacy/` for fixture regeneration and fallback reference. The native macOS and Windows apps are deprecated/paused and retained only for reference or possible future revival.

## What Is Active

| Path | Role |
| --- | --- |
| `frontend/` | Production Vue 3 frontend served from the Vite `dist` build |
| `backend/` | Primary Go backend API, workspace/synthesis service, compatibility routes, Go renderer, and frozen Python parity fixtures |
| `legacy/web-flask/` | Legacy Flask backend/API and Flask-rendered frontend fallback retained for parity reference |
| `legacy/python-renderer/` | Canonical Python MIDI-to-WAV parity reference |
| `assets/previews/` | Shared waveform preview WAV files served through the backend |
| `deploy/production/` | Non-Docker production deployment notes, helper script, and Caddy examples for Vue production |
| `deploy/web-flask/` | Legacy Docker image definition and notes for the Flask backend fallback path |
| `compose.web.yml` | Minimal Docker Compose entrypoint for the legacy Flask fallback path |
| `docs/api-contract.md`, `docs/openapi.yaml` | Web API request and response contract |
| `scripts/generate_python_parity_fixtures.py` | Opt-in regeneration script for Python baseline fixtures |

Retained native app folders:

| Path | Status |
| --- | --- |
| `legacy/native/macos/` | Deprecated/paused native SwiftUI macOS app |
| `legacy/native/windows/` | Deprecated/paused native WinUI 3 Windows app |

## Run The Web App

From the repository root:

```bash
cd backend
PORT=8000 go run ./cmd/server
```

In another terminal, run the Vue frontend:

```bash
cd frontend
npm ci
npm run dev
```

Open `http://127.0.0.1:5173/`. Vite proxies `/api/*` and `/static/previews/*` to the Go backend on `127.0.0.1:8000`.

The legacy Flask-rendered frontend can still be opened directly for fallback or fixture-regeneration testing:

```bash
python3 -m venv .venv
./.venv/bin/python3 -m pip install -r legacy/web-flask/requirements.txt
./.venv/bin/python3 legacy/web-flask/app.py
```

Run the routine post-migration checks:

```bash
cd backend && go test ./...
cd frontend && npm run build
```

Run legacy Python checks only when changing fallback or parity-reference code:

```bash
./.venv/bin/python3 -m unittest discover -s legacy/web-flask/tests
./.venv/bin/python3 -m unittest discover -s legacy/python-renderer/tests
```

## User Limits

These are the current default limits for the web app and renderer. Deployment operators can change web-service limits through environment variables, and renderer safety limits are enforced in the Go renderer with the same parity targets as `legacy/python-renderer/midi_to_wave.py`.

| Limit | Default | Source |
| --- | ---: | --- |
| Request upload size | 20 MiB | `WEB_MAX_UPLOAD_BYTES` |
| Workspace lifetime after last activity | 86400 seconds | `WEB_WORKSPACE_TTL_SECONDS` |
| Queued MIDI files per workspace | 20 files | `WEB_WORKSPACE_MAX_QUEUED_FILES` |
| Total queued upload storage per workspace | 100 MiB | `WEB_WORKSPACE_MAX_UPLOAD_BYTES` |
| Converted WAV files per workspace | 20 files | `WEB_WORKSPACE_MAX_CONVERTED_FILES` |
| Compatibility job download lifetime | 1800 seconds | `WEB_DOWNLOAD_TTL_SECONDS` |
| Active render workers per container | 2 workers | `WEB_RENDER_WORKERS` |
| Waiting render queue per container | 8 jobs | `WEB_RENDER_QUEUE_SIZE` |
| MIDI duration | 1800 seconds | renderer limit |
| Rendered samples | 172800000 samples | renderer limit |
| WAV sample data size | 345600000 bytes, about 329.6 MiB | renderer limit |
| MIDI notes | 20000 notes | renderer limit |
| Sound layers | 4 layers | renderer limit and web config |
| Frequency curve points per layer | 8 points | renderer limit |
| Sample rates | 44100, 48000, or 96000 Hz | web validation |
| Pulse duty cycle | 0.01 to 0.99 | renderer validation |
| Web layer volume | 0.0 to 2.0 | workspace config validation |
| Frequency curve gain | -36 dB to 12 dB | renderer validation |
| Frequency curve range | MIDI note 0 to 127 frequencies | renderer validation |

Queued uploads and converted WAV files are temporary. When users clear queued or converted files in the browser, the web app asks the server to delete the corresponding temporary files immediately.

## Web API

The Vue frontend uses anonymous, cookie-backed temporary workspaces through the Go API. `GET /api/workspace` creates or restores the workspace, and resource routes require the active workspace cookie. The readable API contract is in `docs/api-contract.md`, with the machine-readable OpenAPI contract in `docs/openapi.yaml`.

Primary routes:

- `GET /api/health`
- `GET /api/workspace`
- `POST /api/workspace/uploads`
- `DELETE /api/workspace/uploads/<file_id>`
- `PATCH /api/workspace/queue`
- `PUT /api/workspace/config`
- `POST /api/synthesis-jobs`
- `GET /api/synthesis-jobs/<job_id>`
- `GET /api/synthesis-jobs/<job_id>/download`
- `DELETE /api/synthesis-jobs/<job_id>`

Compatibility routes remain for older clients:

- `POST /synthesise`
- `POST /synthesise/jobs`
- `GET /synthesise/jobs/<job_id>`
- `GET /synthesise/jobs/<job_id>/download`
- `DELETE /synthesise/jobs/<job_id>`

API errors use `{"error":{"code":"...","message":"..."}}`. Compatibility routes keep the older `{"error":"..."}` shape.

## Sound Configuration

The web app stores sample rate and layer settings in the temporary workspace. Synthesis supports pulse, sine, sawtooth, and triangle layers. Frequency-gain curves are validated by the shared renderer and are applied per layer during synthesis.

Output naming:

- Single audible layer without a curve: `<original>_<wave>.wav`
- Multiple audible layers without a curve: `<original>_mix.wav`
- Any audible layer with a non-empty frequency curve: `<original>_<base>_<hash>.wav`

The hash is derived from the sanitised layer payload, so different curve settings do not reuse the same export name.

## Localisation

The production Vue UI keeps JSON catalog files under `frontend/src/i18n/` for English, Spanish, French, and Simplified Chinese. The legacy Flask-rendered UI keeps its catalogs under `legacy/web-flask/i18n/`. Keep `en.json`, `es.json`, `fr.json`, and `zh-CN.json` key sets aligned in the production frontend catalogs. English is the fallback locale. Repository documentation remains English and Simplified Chinese. Follow the standard process in [docs/localisation.md](./docs/localisation.md).

User-facing web strings should go through the catalog rather than being hardcoded in templates or JavaScript. Native macOS and Windows localisation work is out of scope while those apps remain paused.

## Deployment

The intended production model runs without Docker:

```bash
cd backend && go build -o octabit-server ./cmd/server
PORT=8000 WEB_SYNTHESISE_JOB_ROOT=/var/lib/octabit ./octabit-server
cd frontend && npm ci && npm run build
```

For public deployment, keep the Go backend private on `127.0.0.1:8000`. Caddy serves `frontend/dist` as the public frontend and reverse proxies `/api/*`, `/static/previews/*`, and `/synthesise*` to Go. Production deployment notes, Caddy examples, smoke checks, and rollback steps are in `deploy/production/README.md`.

The Docker image in `deploy/web-flask/` remains available for a Flask-backend or legacy Flask-rendered fallback path. It pins its Python base image by digest and installs from hash-locked requirement files.

## License

This project is licensed under the GNU Affero General Public License v3.0 or later (`AGPL-3.0-or-later`). See [LICENSE.md](./LICENSE.md) for details.
