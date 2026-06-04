# Repository layout

Language/语言: English | [简体中文](./repository-layout.zh-Hans.md)

This repository is a monorepo for OctaBit, a simple web tool for converting
MIDI files into 8-bit style music. The current production web frontend is the
Vue app in `frontend/`, served from its Vite `dist` build for `octabit.cc`. The
primary backend is the Go service in `backend/`, which owns the stable `/api/*`
contract, workspace/synthesis service, previews, and runtime renderer.

## Top-level map

| Path | Purpose |
| --- | --- |
| `AGENTS.md` | Repository instructions for coding agents and local workflows. |
| `README.md`, `README.zh-Hans.md` | Root project overview, setup notes, app entry points, and repository licence summary. |
| `LICENSE.md` | Repository AGPL licence text. |
| `frontend/` | Production Vue/Vite frontend. |
| `backend/` | Primary Go backend module and frozen Python baseline fixtures. |
| `assets/previews/` | Canonical waveform preview WAV files shared by the apps. |
| `docs/` | API contract, repository layout notes, localisation procedure, licensing audit, and review reports. |
| `deploy/production/` | Non-Docker production deployment notes, helper script, and Caddy examples for Vue production. |
| `scripts/` | Opt-in local maintenance scripts. |
| `.gitignore`, `.gitattributes` | Repository ignore and line-ending rules. |
| `output/`, `tmp/` | Tracked historical generated review artefacts; both paths are ignored for future generated output. |

Build outputs, `.codex/`, `.sisyphus/`, `.DS_Store`, and app `build/` folders
are not part of the maintained source layout.

### `backend/`

Primary Go backend runtime. It satisfies the OpenAPI contract with frozen
Python baseline fixtures while removing Python from the normal runtime path.

- `go.mod`: Go module for the backend.
- `cmd/server/`: process startup, environment config, SQLite workspace store
  opening, logging, graceful shutdown, and HTTP server wiring.
- `internal/config/`: environment variable parsing and defaults.
- `internal/httpapi/`: OpenAPI-shaped HTTP routes for health, the workspace
  state/upload/queue/config API, workspace-backed `/api/synthesis-jobs` JSON
  and multipart render/poll/download/delete flows, and previews plus route
  tests and OpenAPI route-registration conformance.
- `internal/jobs/`: bounded render execution and in-memory render job lifecycle
  tests for queue semantics shared with the workspace-backed job flow.
- `internal/midi/`: Standard MIDI File note extraction using
  `gitlab.com/gomidi/midi/v2/smf`.
- `internal/renderer/`: renderer limits, layer validation, frequency curve
  interpolation, curve hashes, output naming, and note-event PCM/WAV synthesis.
- `internal/storage/`: SQLite schema, connection pragmas, token hashing, path
  helpers, and workspace cleanup/cascade behavior.
- `internal/workspace/`: workspace token lifecycle, state payloads, upload
  queue operations, limits, config persistence, SQLite-backed synthesis jobs,
  WAV output cleanup, and renderer form payload conversion.
- `testdata/python-baseline/`: normalised API transcripts, representative MIDI
  inputs, parsed note-event fixtures, expected WAV outputs, renderer naming/hash expectations, and
  workspace config normalisation cases.

## Application targets

### `frontend/`

Production Vue/Vite frontend for the public browser experience.

- `index.html`: Vite application shell.
- `src/App.vue`: top-level Vue workflow and state orchestration.
- `src/api/`: typed client for the backend `/api/*` routes.
- `src/components/`: upload queue, layer editor, output controls, header
  controls, converted files, and curve editor components.
- `src/i18n/`: English, Spanish, French, Japanese, Simplified Chinese, and Traditional Chinese frontend catalogs.
- `src/styles/app.css`: current OctaBit visual system.
- `vite.config.ts`: development proxy for `/api` and `/static/previews` to
  `http://127.0.0.1:8000`.
- `package.json` and `package-lock.json`: Vue/Vite dependency metadata.

Production Caddy serves `frontend/dist` and proxies API and preview asset
requests to the Go backend.


## Shared core and assets

### `assets/previews/`

Canonical preview WAV assets used by the web frontend/backend path.
`assets/README.md` records their intended usage and provenance.

## Documentation and generated artefacts

- `docs/api-contract.md`, `docs/api-contract.zh-Hans.md`, and
  `docs/openapi.yaml`: current web API contract, job payloads, and public-demo
  safeguards.
- `docs/repository-layout.md` and `docs/repository-layout.zh-Hans.md`: current
  repository layout in English and Simplified Chinese.
- `docs/localisation.md`: English-only agent procedure for production UI
  localisation and related documentation updates.
- `docs/licensing-audit.md`: licensing and attribution audit for repository and
  release planning.
- `output/pdf/repo-structure-evaluation.pdf`,
  `tmp/pdfs/repo-structure-evaluation.html`, and
  `tmp/pdfs/rendered/repo-structure-evaluation.png`: tracked historical
  generated review artefacts. They are not the source of truth for the current
  layout.

## Build and development flow

Run commands from the repository root unless a document says otherwise.

Routine checks:

```bash
cd backend && go test ./...
cd frontend && npm run build
```

For Vue development, run the Go backend on port 8000 and then the Vite dev
server:

```bash
cd backend
PORT=8000 go run ./cmd/server
```

```bash
cd frontend
npm ci
npm run dev
```

For Vue production builds:

```bash
cd frontend
npm ci
npm run build
```

The current non-Docker production path runs the Go backend privately on
`127.0.0.1:8000`, with systemd managing the service and Caddy serving
`frontend/dist` while reverse proxying `/api/*` and `/static/previews/*` to
that private listener. Keep the workspace/job directory, job TTL, maximum
upload size, and render worker settings aligned with the current synthesis job
behaviour. See `deploy/production/README.md` for the Caddy production and
rollback examples.

## Dependency and packaging boundaries

- Go backend dependencies live in `backend/go.mod` and `backend/go.sum`.
- Production frontend JavaScript dependencies live in `frontend/package.json`
  and `frontend/package-lock.json`.

## Ownership boundaries

- Runtime renderer behaviour belongs in `backend/internal/renderer/`.
- Production web UI belongs under `frontend/`.
- Shared binary/media assets belong under `assets/`.
- Repository-wide documentation, audits, and review notes belong under `docs/`.
- Deployment-specific files belong under `deploy/`.
