# Repository layout

Language/语言: English | [简体中文](./repository-layout.zh-CN.md)

This repository is a monorepo for OctaBit, a simple web tool for converting
MIDI files into 8-bit style music. The current production web frontend is the
Vue app in `frontend/`, served from its Vite `dist` build for `octabit.cc`. The
primary backend is the Go service in `backend/`, which owns the stable `/api/*`
contract, workspace/synthesis service, compatibility routes, previews, and
runtime renderer. Flask and the Python renderer are retained under `legacy/`
for fixture regeneration and fallback reference. The native macOS and Windows
apps are deprecated/paused, not actively developed, and retained for reference
or possible future revival.

## Top-level map

| Path | Purpose |
| --- | --- |
| `AGENTS.md` | Repository instructions for coding agents and local workflows. |
| `README.md`, `README.zh-CN.md` | Root project overview, setup notes, app entry points, and repository licence summary. |
| `LICENSE.md` | Repository AGPL licence text. |
| `frontend/` | Production Vue/Vite frontend. |
| `backend/` | Primary Go backend module and frozen Python baseline fixtures. |
| `legacy/web-flask/` | Legacy Flask backend/API and Flask-rendered frontend fallback retained for parity reference. |
| `legacy/python-renderer/` | Canonical Python MIDI-to-WAV parity reference. |
| `legacy/native/` | Deprecated/paused macOS and Windows native apps. |
| `assets/previews/` | Canonical waveform preview WAV files shared by the apps. |
| `docs/` | API contract, repository layout notes, licensing audit, and review reports. |
| `deploy/production/` | Non-Docker production deployment notes, helper script, and Caddy examples for Vue production. |
| `deploy/web-flask/` | Docker deployment documentation and Dockerfile for the legacy Flask fallback path. |
| `scripts/` | Opt-in local maintenance and fixture-regeneration scripts. |
| `compose.web.yml` | Docker Compose entry point for the legacy Flask fallback path. |
| `global.json` | .NET SDK selection for the retained Windows solution. |
| `.dockerignore`, `.gitignore`, `.gitattributes` | Repository packaging, ignore, and line-ending rules. |
| `output/`, `tmp/` | Tracked historical generated review artefacts; both paths are ignored for future generated output. |

Local-only folders such as `.venv/`, build outputs, `.codex/`, `.sisyphus/`,
`.DS_Store`, `__pycache__/`, `.xcodebuild/`, and app `build/` folders are not
part of the maintained source layout.

### `backend/`

Primary Go backend runtime and migration proof module. It satisfies the
OpenAPI contract with frozen Python baseline fixtures while removing Python
from the normal runtime path.

- `go.mod`: Go module for the migration backend.
- `cmd/server/`: process startup, environment config, SQLite workspace store
  opening, logging, graceful shutdown, and HTTP server wiring.
- `internal/config/`: current Flask-compatible environment variable parsing and
  defaults.
- `internal/httpapi/`: initial OpenAPI-shaped HTTP routes for health, the
  workspace state/upload/queue/config API, workspace-backed
  `/api/synthesis-jobs` JSON and multipart render/poll/download/delete flows,
  legacy `/synthesise` plus `/synthesise/jobs` render/poll/download/delete
  routes, and previews plus route tests, OpenAPI route-registration
  conformance, and Python-baseline replay for the routes currently covered by
  frozen fixtures.
- `internal/jobs/`: bounded render execution, legacy job file storage, and
  in-memory render job lifecycle tests for queue semantics shared with the
  workspace-backed job flow.
- `internal/midi/`: Standard MIDI File note extraction using
  `gitlab.com/gomidi/midi/v2/smf`, selected through fixture parity against the
  PrettyMIDI-derived baseline.
- `internal/renderer/`: renderer limits, layer validation, frequency curve
  interpolation, curve hashes, output naming, and note-event PCM/WAV synthesis
  aligned with the Python baseline.
- `internal/storage/`: SQLite schema, connection pragmas, token hashing, path
  helpers, and workspace cleanup/cascade behavior aligned with the Flask
  storage model.
- `internal/workspace/`: workspace token lifecycle, state payloads, upload
  queue operations, limits, config persistence, SQLite-backed synthesis jobs,
  WAV output cleanup, and renderer form payload conversion aligned with the
  Python baseline.
- `testdata/python-baseline/`: normalised API transcripts, representative MIDI
  inputs, parsed note-event fixtures, expected WAV outputs, renderer naming/hash expectations, and
  workspace config normalisation cases. Regeneration is explicit through
  `scripts/generate_python_parity_fixtures.py`; routine post-migration tests
  read these files without requiring Python.

## Application targets

### `frontend/`

Production Vue/Vite frontend for the public browser experience.

- `index.html`: Vite application shell.
- `src/App.vue`: top-level Vue workflow and state orchestration.
- `src/api/`: typed client for the backend `/api/*` routes.
- `src/components/`: upload queue, layer editor, output controls, header
  controls, converted files, and curve editor components.
- `src/i18n/`: English, French, and Simplified Chinese frontend catalogs.
- `src/styles/app.css`: current OctaBit visual system reused from the Flask UI.
- `vite.config.ts`: development proxy for `/api` and `/static/previews` to
  `http://127.0.0.1:8000`.
- `package.json` and `package-lock.json`: Vue/Vite dependency metadata.

Production Caddy serves `frontend/dist` and proxies API, preview asset, and
compatibility synthesis requests to the Go backend.

### `legacy/web-flask/`

Legacy Flask backend API, workspace/synthesis service, preview route provider,
and legacy Flask-rendered frontend fallback retained for parity fixture
regeneration and fallback reference.

- `app.py`: Flask entry point, upload handling, synthesis/API endpoints,
  preview routes, and server-side render job endpoints.
- `synthesis_jobs.py`: filesystem-backed synthesis job lifecycle, cleanup, and
  render-thread orchestration.
- `templates/index.html`: browser UI shell.
- `static/css/` and `static/js/`: web-specific styling and browser behaviour.
- `i18n/`: JSON catalogues for English, French, and Simplified Chinese UI text.
- `tests/`: Flask and render-path tests.
- `requirements.txt`: web runtime dependencies; it includes the shared renderer
  requirements.
- `Launch_Synthesiser.command` and `Launch_Synthesiser.bat`: local launchers.
- `README.md`, `README.zh-CN.md`, `User_Guide.txt`: web app documentation.

The Flask backend delegates synthesis to `legacy/python-renderer/midi_to_wave.py`
and serves preview audio from `assets/previews/`; normal production runtime
uses the Go backend instead.

### `legacy/native/macos/`

Deprecated/paused native SwiftUI macOS app and Xcode project. This code is not
the main development target; it is retained for reference or possible future
revival while the project focuses on the web service.

- `MIDI8BitSynthesiser.xcodeproj/`: Xcode project and shared scheme.
- `MIDI8BitSynthesiser/`: SwiftUI app source.
- `MIDI8BitSynthesiserTests/`: XCTest target for model and filename logic.
- `macos/build_desktop_resources.sh`: Xcode build-phase script that freezes the
  Python renderer into a helper binary and copies preview WAV assets into the
  app bundle.
- `requirements-build.txt`: Python build dependencies for the helper.
- `macos/README.md`, `macos/README.zh-CN.md`: macOS build and usage notes.

The macOS app does not run the Flask server. It launches the bundled Python
helper for each queued MIDI file.

### `legacy/native/windows/`

Deprecated/paused native WinUI 3 Windows app, C# renderer, tests, installer,
and review tooling. This code is not the main development target; it is
retained for reference or possible future revival while the project focuses on
the web service.

- `Midi8BitSynthesiser.sln`: Windows solution.
- `Directory.Packages.props`: central NuGet package versions.
- `src/Midi8BitSynthesiser.Core/`: C# rendering engine, waveform models, and
  output naming.
- `src/Midi8BitSynthesiser.App/`: WinUI 3 shell, compatibility checks, file
  dialog services, preview playback, localisation resources, and app manifest.
- `tests/Midi8BitSynthesiser.Tests/`: unit, workflow, compatibility, and Python
  parity tests.
- `installer/Midi8BitSynthesiser.iss`: Inno Setup installer script.
- `installer/RuntimeNotice.txt`: installer pre-install runtime notice.
- `scripts/create_review_bundle.sh`: script for preparing a Windows review
  bundle.
- `README.md`, `README.zh-CN.md`, `REVIEWING.md`: Windows build and review
  documentation.

The retained Windows app has its own C# renderer and validates it against the
Python reference renderer in parity tests. The app project links preview WAV
files from the canonical `assets/previews/` folder for build and publish
output. A byte-identical tracked copy also exists under
`src/Midi8BitSynthesiser.App/Assets/Previews/`, but the project file uses the
shared asset folder as the build source.

## Shared core and assets

### `legacy/python-renderer/`

Canonical Python MIDI-to-WAV renderer.

- `midi_to_wave.py`: renderer module and CLI entry point.
- `requirements.txt`: renderer/runtime dependencies only.
- `tests/`: renderer tests.
- `README.md`: renderer interface, layer schema, and dependency boundary.

The renderer accepts platform-neutral file paths and waveform layer settings,
then writes a WAV file to disk. The legacy Flask backend calls it directly. The
retained macOS app also calls it directly, and the retained Windows app uses it
as the parity reference for the native C# renderer.

### `assets/previews/`

Canonical preview WAV assets used by the web frontend/backend path and retained
native app paths. `assets/README.md` records their intended usage and
provenance.

## Documentation and generated artefacts

- `docs/api-contract.md`, `docs/api-contract.zh-CN.md`, and
  `docs/openapi.yaml`: current web API contract, compatibility route notes, job
  payloads, and public-demo safeguards.
- `docs/repository-layout.md` and `docs/repository-layout.zh-CN.md`: current
  repository layout in English and Simplified Chinese.
- `docs/licensing-audit.md`: licensing and attribution audit for repository and
  release planning.
- `docs/reviews/windows-app-review.md`: Windows review notes.
- `output/pdf/repo-structure-evaluation.pdf`,
  `tmp/pdfs/repo-structure-evaluation.html`, and
  `tmp/pdfs/rendered/repo-structure-evaluation.png`: tracked historical
  generated review artefacts. They are not the source of truth for the current
  layout.

## Build and development flow

Run commands from the repository root unless a document says otherwise.

Create the local Python environment:

```bash
python3 -m venv .venv
```

Install only the dependencies for the area being worked on:

```bash
./.venv/bin/python3 -m pip install -r legacy/web-flask/requirements.txt
./.venv/bin/python3 -m pip install -r legacy/native/macos/requirements-build.txt
./.venv/bin/python3 -m pip install -r legacy/python-renderer/requirements.txt
```

Routine checks:

```bash
cd backend && go test ./...
cd frontend && npm run build
```

Legacy Python checks are for fallback or parity-reference changes:

```bash
./.venv/bin/python3 -m unittest discover -s legacy/web-flask/tests
./.venv/bin/python3 -m unittest discover -s legacy/python-renderer/tests
```

The paused Windows app can still be inspected with .NET 8 and Python renderer
dependencies:

```powershell
dotnet restore legacy/native/windows/Midi8BitSynthesiser.sln
dotnet build legacy/native/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64
dotnet test legacy/native/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build
```

No maintained Windows release workflow or CI publish pipeline remains in the
repository. If native Windows packaging work is revived later, re-establish the
publish/release steps from the current project files rather than relying on a
removed workflow.

The paused macOS app builds through Xcode with the `MIDI8BitSynthesiser`
scheme. The Xcode build phase runs
`legacy/native/macos/macos/build_desktop_resources.sh`.

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
`frontend/dist` while reverse proxying `/api/*`, `/static/previews/*`, and
`/synthesise*` to that private listener. Keep the workspace/job directory,
job TTL, maximum upload size, and render worker settings aligned with the
current synthesis job behaviour. See `deploy/production/README.md` for the
Caddy production and rollback examples.

The Docker deployment remains available as an alternate legacy Flask fallback
path:

```bash
docker compose -f compose.web.yml up -d --build
```

The compose file binds the service to `127.0.0.1:8000` for tunnel-first testing
and builds only the legacy Flask backend/fallback, shared renderer, shared preview
assets, and project licence into the image.

## Dependency and packaging boundaries

- Python renderer dependencies live in `legacy/python-renderer/requirements.txt`.
- Go backend dependencies live in `backend/go.mod` and `backend/go.sum`.
- Web-only legacy Python dependencies live in `legacy/web-flask/requirements.txt`.
- Production frontend JavaScript dependencies live in `frontend/package.json`
  and `frontend/package-lock.json`.
- macOS helper build dependencies live in `legacy/native/macos/requirements-build.txt`.
- Windows NuGet versions live in `legacy/native/windows/Directory.Packages.props`.
- Docker deployment files are scoped to the legacy Flask fallback path.
- Retained native app packaging stays under the relevant app folder.

## Ownership boundaries

- Runtime renderer behaviour belongs in `backend/internal/renderer/`.
- Python parity reference behaviour belongs in `legacy/python-renderer/`.
- Production web UI belongs under `frontend/`.
- Legacy Flask backend API and Flask-rendered fallback logic belongs under
  `legacy/web-flask/`.
- Retained native UI, launch, and packaging logic stays under the relevant
  `legacy/native/` folder.
- Shared binary/media assets belong under `assets/`.
- Repository-wide documentation, audits, and review notes belong under `docs/`.
- Deployment-specific files belong under `deploy/` and root deployment entry
  points such as `compose.web.yml`.
