# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OctaBit is a browser-based MIDI-to-8-bit-WAV converter (live at octabit.cc). AGPL-3.0-or-later.

The production stack is a Vue 3 frontend (`frontend/`) talking to a Go backend (`backend/`) over a stable `/api/*` contract. The legacy Flask backend and Python renderer under `legacy/` are retained only for parity fixture regeneration and fallback reference — new feature work targets the Go backend.

## Essential Commands

```bash
# Go backend — run, test all, test a single package
cd backend && PORT=8000 go run ./cmd/server
cd backend && go test ./...
cd backend && go test ./internal/workspace/

# Vue frontend — install, dev server, type-check + build, tests
cd frontend && npm ci
cd frontend && npm run dev          # starts on :5173, proxies /api to :8000
cd frontend && npm run build        # vue-tsc --noEmit + vite build
cd frontend && npm test             # vitest run
cd frontend && npm run test:watch   # vitest watch mode

# Regenerate Python parity fixtures (only when renderer behavior changes)
python3 scripts/generate_python_parity_fixtures.py

# Legacy Flask fallback (rarely needed)
python3 -m venv .venv
./.venv/bin/python3 -m pip install -r legacy/web-flask/requirements.txt
./.venv/bin/python3 -m pip install -r legacy/python-renderer/requirements.txt
./.venv/bin/python3 legacy/web-flask/app.py
./.venv/bin/python3 -m unittest discover -s legacy/web-flask/tests
./.venv/bin/python3 -m unittest discover -s legacy/python-renderer/tests
```

## Architecture

### Go Backend (`backend/`)

The backend uses only the standard library + `gitlab.com/gomidi/midi/v2` for MIDI parsing and `modernc.org/sqlite` for storage (pure Go SQLite, no CGO).

**Package dependency graph (top-down):**

```
cmd/server (main: wires config, store, service, router, graceful shutdown)
  └── internal/httpapi (net/http mux, all route handlers, cookie auth, multipart parsing)
       ├── workspace.Service (token lifecycle, upload queue, config, job orchestration)
       │    ├── storage.Store (SQLite: workspace/upload/job tables, token hashing, cascade cleanup)
       │    ├── jobs.Manager + jobs.Executor (bounded semaphore, render lifecycle, TTL expiry)
       │    ├── midi.Parser (SMF note extraction via gomidi/smf)
       │    └── renderer (limits, layer validation, freq curve interpolation, PCM/WAV synthesis)
       └── jobs.LegacyService (file-based job store for /synthesise* compat routes)
```

- `internal/config`: all configuration from env vars with defaults (no config files). `findUp()` searches parent dirs for `assets/previews/`.
- `internal/renderer`: the Go renderer is the production render path. Python parity fixtures in `backend/testdata/python-baseline/` are frozen snapshots — Go tests assert output matches these fixtures.
- `internal/httpapi`: uses Go 1.22+ path-value routing (`{job_id}`). Two error shapes: `{"error":{"code":"...","message":"..."}}` for `/api/*`, `{"error":"..."}` for `/synthesise*`.
- `internal/workspace`: anonymous cookie-backed workspaces with hex-token identity. Jobs render asynchronously through a bounded executor (workers + queue slots). `RunInline` option bypasses the semaphore for tests.

### Vue Frontend (`frontend/`)

- **Composables** (`src/composables/`): `useWorkspace.ts` is the central state machine — fetches workspace, manages upload queue, triggers synthesis, polls jobs. `useLocale.ts` and `useTheme.ts` handle i18n and dark mode.
- **API client** (`src/api/client.ts`): thin fetch wrapper for all `/api/*` calls, handles cookie passthrough and error normalization.
- **Components**: `UploadQueue.vue`, `LayerEditor.vue`, `FrequencyCurveEditor.vue`, `OutputControls.vue`, `ConvertedFilesList.vue`, `HeaderControls.vue`. Orchestrated by `App.vue`.
- **i18n**: JSON catalogs in `src/i18n/` (en, fr, zh-CN). English is the fallback; keep keys aligned across all catalogs you touch.
- **Testing**: Vitest + jsdom + `@vue/test-utils`. Test setup in `vitest.setup.ts`. Component tests live in `src/composables/__tests__/`.

### Key Conventions

- Resource IDs are 32-char hex strings (16 random bytes). Validated with `isValidResourceID()` in the router.
- Upload filenames must end in `.mid` or `.midi` (case-insensitive).
- Workspace cookies are named `octabit_workspace`, HttpOnly, SameSite=Lax.
- Render output naming: `<original>_<wave>.wav` for single layer, `<original>_mix.wav` for multi-layer, `<original>_<base>_<hash>.wav` when frequency curves are non-empty. The hash is derived from sanitised layer payload.
- The `ci.yml` workflow runs `go test ./...` in `backend/` and `npm ci && npm test && npm run build` in `frontend/` on pushes/PRs to `main`.
- The `dev` branch is the active development branch; `main` is the stable deployable branch. Feature branches branch from `dev`.

### Repository Layout vs Naming

Trust the filesystem and `README.md` over the contributor guide when paths disagree.
