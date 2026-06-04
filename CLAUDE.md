<!-- MONOREPO-START -->
## Monorepo Structure

This is a **private monorepo** for developing both OctaBit OSS and OctaBit Pro.

| Directory | Content | Public? |
|-----------|---------|---------|
| `backend/`, `frontend/` | OSS codebase | Synced to `bagags/octabit` |
| `overlays/backend/`, `overlays/frontend/src/` | Pro replacement files | Private |
| `scripts/pro/` | Pro build, dev, sync tooling | Private |
| `deploy/pro/` | Pro deployment assets | Private |

**Critical rule**: When editing files in `backend/` or `frontend/`, check whether a Pro overlay
exists at the same relative path under `overlays/`. If it does, both versions may need updating.

**Pro development**: See `AGENTS.pro.md` and `README.pro.md`. Run `scripts/pro/build.sh` to
assemble and test the Pro build. The assemble step copies OSS code + overlays into a staging
directory, then builds from there.

**Public mirror**: `scripts/pro/sync-oss.sh` extracts the OSS subset and pushes to the public
repository. The script uses an allowlist approach — only explicitly listed paths are synced;
everything under `overlays/`, `scripts/pro/`, and `deploy/pro/` is excluded.
<!-- MONOREPO-END -->
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OctaBit is a browser-based MIDI-to-8-bit-WAV converter (live at octabit.cc). AGPL-3.0-or-later.

The production stack is a Vue 3 frontend (`frontend/`) talking to a Go backend (`backend/`) over a stable `/api/*` contract.

`bagags/octabit` is the public OSS mirror generated from a private upstream monorepo. The public mirror is for reading, auditing, running, and self-hosting the AGPL-licensed OSS code; it is not an open contribution target and does not accept unsolicited pull requests.

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
       │    ├── jobs.Executor (bounded semaphore, render lifecycle, TTL expiry)
       │    ├── midi.Parser (SMF note extraction via gomidi/smf)
       │    └── renderer (limits, layer validation, freq curve interpolation, PCM/WAV synthesis)
```

- `internal/config`: all configuration from env vars with defaults (no config files). `findUp()` searches parent dirs for `assets/previews/`.
- `internal/renderer`: the Go renderer is the production render path. Fixtures in `backend/testdata/python-baseline/` are frozen snapshots — Go tests assert output matches these fixtures.
- `internal/httpapi`: uses Go 1.22+ path-value routing (`{job_id}`). Error shape: `{"error":{"code":"...","message":"..."}}` for all `/api/*` routes.
- `internal/workspace`: anonymous cookie-backed workspaces with hex-token identity. Jobs render asynchronously through a bounded executor (workers + queue slots). `RunInline` option bypasses the semaphore for tests.

### Vue Frontend (`frontend/`)

- **Composables** (`src/composables/`): `useWorkspace.ts` is the central state machine — fetches workspace, manages upload queue, triggers synthesis, polls jobs. `useLocale.ts` and `useTheme.ts` handle i18n and dark mode.
- **API client** (`src/api/client.ts`): thin fetch wrapper for all `/api/*` calls, handles cookie passthrough and error normalization.
- **Components**: `UploadQueue.vue`, `LayerEditor.vue`, `FrequencyCurveEditor.vue`, `OutputControls.vue`, `ConvertedFilesList.vue`, `HeaderControls.vue`. Orchestrated by `App.vue`.
- **Types & Utilities**: `src/types/` (API/UI type definitions), `src/lib.ts` (shared helpers), `src/icons.ts` (icon mappings). Styles in `src/styles/app.css`.
- **i18n**: JSON catalogs in `src/i18n/` (`en`, `es`, `fr`, `ja`, `zh-Hans`, `zh-Hant`). English is the fallback; keep keys aligned across all catalogs you touch.
- **Testing**: Vitest + jsdom + `@vue/test-utils`. Test setup in `vitest.setup.ts`. Composable tests live in `src/composables/__tests__/`.

### Key Conventions

- Resource IDs are 32-char hex strings (16 random bytes). Validated with `isValidResourceID()` in the router.
- Upload filenames must end in `.mid` or `.midi` (case-insensitive).
- Workspace cookies are named `octabit_workspace`, HttpOnly, SameSite=Lax.
- Render output naming: `<original>_<wave>.wav` for single layer, `<original>_mix.wav` for multi-layer, `<original>_<base>_<hash>.wav` when frequency curves are non-empty. The hash is derived from sanitised layer payload.
- The `ci.yml` workflow runs `go test ./...` in `backend/` and `npm ci && npm test && npm run build` in `frontend/` on pushes/PRs to `main`.
- The `dev` branch is the active development branch; `main` is the stable deployable branch. Feature branches branch from `dev`.

### Repository Layout vs Naming

Trust the filesystem and `README.md` over the contributor guide when paths disagree.

## Commit Convention

Follow [Conventional Commits](https://www.conventionalcommits.org/): `type(scope?): description`. Types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`.

## Companion Files

- `AGENTS.md` — PR guidelines, agent behavioral instructions, and extended directory listing (`deploy/`, `docs/`). Some agents may read that file first or exclusively; it cross-references back here for the authoritative command reference and architecture.
