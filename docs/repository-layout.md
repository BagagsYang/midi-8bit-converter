# Repository layout

Language/语言: English | [简体中文](./repository-layout.zh-CN.md)

This repository is a monorepo for OctaBit, a simple web tool for converting
MIDI files into 8-bit style music. The current production web frontend is the
Vue app in `apps/web-vue/`, served from its Vite `dist` build for
`octabit.cc`. Flask/Gunicorn in `apps/web-flask/` remains the private backend
API and workspace/synthesis service, and its server-rendered frontend is kept
as a legacy fallback. The native macOS and Windows apps are deprecated/paused,
not actively developed, and retained for reference or possible future revival.
The repository also contains the canonical Python renderer, shared preview
assets, API contract documentation, deployment files, and reference
documentation.

## Top-level map

| Path | Purpose |
| --- | --- |
| `AGENTS.md` | Repository instructions for coding agents and local workflows. |
| `README.md`, `README.zh-CN.md` | Root project overview, setup notes, app entry points, and repository licence summary. |
| `LICENSE.md` | Repository AGPL licence text. |
| `apps/` | Production Vue frontend, Flask backend/fallback, retained native app code, and the reserved desktop placeholder. |
| `core/python-renderer/` | Canonical Python MIDI-to-WAV renderer and parity reference. |
| `assets/previews/` | Canonical waveform preview WAV files shared by the apps. |
| `docs/` | API contract, repository layout notes, licensing audit, and review reports. |
| `deploy/production/` | Non-Docker production deployment notes, helper script, and Caddy examples for Vue production. |
| `deploy/web-flask/` | Docker deployment documentation and Dockerfile for the Flask backend or legacy fallback path. |
| `compose.web.yml` | Docker Compose entry point for the Flask backend or legacy fallback path. |
| `global.json` | .NET SDK selection for the retained Windows solution. |
| `.dockerignore`, `.gitignore`, `.gitattributes` | Repository packaging, ignore, and line-ending rules. |
| `output/`, `tmp/` | Tracked historical generated review artefacts; both paths are ignored for future generated output. |

Local-only folders such as `.venv/`, build outputs, `.codex/`, `.sisyphus/`,
`.DS_Store`, `__pycache__/`, `.xcodebuild/`, and app `build/` folders are not
part of the maintained source layout.

## Application targets

### `apps/web-vue/`

Production Vue/Vite frontend for the public browser experience.

- `index.html`: Vite application shell.
- `src/App.vue`: top-level Vue workflow and state orchestration.
- `src/api/`: typed client for the Flask `/api/*` routes.
- `src/components/`: upload queue, layer editor, output controls, header
  controls, converted files, and curve editor components.
- `src/i18n/`: English, French, and Simplified Chinese frontend catalogs.
- `src/styles/app.css`: current OctaBit visual system reused from the Flask UI.
- `vite.config.ts`: development proxy for `/api` and `/static/previews` to
  `http://127.0.0.1:8000`.
- `package.json` and `package-lock.json`: Vue/Vite dependency metadata.

Production Caddy serves `apps/web-vue/dist` and proxies API plus preview asset
requests to Flask/Gunicorn.

### `apps/web-flask/`

Flask backend API, workspace/synthesis service, preview route provider, and
legacy Flask-rendered frontend fallback.

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

The Flask backend delegates synthesis to `core/python-renderer/midi_to_wave.py`
and serves preview audio from `assets/previews/`.

### `apps/macos/`

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

### `apps/windows/`

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

### `apps/desktop/`

Reserved placeholder for a future desktop packaging layer. It contains README
files only and no app implementation.

## Shared core and assets

### `core/python-renderer/`

Canonical Python MIDI-to-WAV renderer.

- `midi_to_wave.py`: renderer module and CLI entry point.
- `requirements.txt`: renderer/runtime dependencies only.
- `tests/`: renderer tests.
- `README.md`: renderer interface, layer schema, and dependency boundary.

The renderer accepts platform-neutral file paths and waveform layer settings,
then writes a WAV file to disk. The Flask backend calls it directly. The
retained macOS app also calls it directly, and the retained Windows app uses it
as the parity reference for the native C# renderer.

### `assets/previews/`

Canonical preview WAV assets used by the web frontend/backend path and retained
native app paths. `assets/README.md` records their intended usage and
provenance.

## Documentation and generated artefacts

- `docs/api-contract.md` and `docs/api-contract.zh-CN.md`: current web API
  contract, compatibility route notes, job payloads, and public-demo safeguards.
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
./.venv/bin/python3 -m pip install -r apps/web-flask/requirements.txt
./.venv/bin/python3 -m pip install -r apps/macos/requirements-build.txt
./.venv/bin/python3 -m pip install -r core/python-renderer/requirements.txt
```

Common checks:

```bash
./.venv/bin/python3 -m unittest discover -s apps/web-flask/tests
./.venv/bin/python3 -m unittest discover -s core/python-renderer/tests
```

The paused Windows app can still be inspected with .NET 8 and Python renderer
dependencies:

```powershell
dotnet restore apps/windows/Midi8BitSynthesiser.sln
dotnet build apps/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64
dotnet test apps/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build
```

No maintained Windows release workflow or CI publish pipeline remains in the
repository. If native Windows packaging work is revived later, re-establish the
publish/release steps from the current project files rather than relying on a
removed workflow.

The paused macOS app builds through Xcode with the `MIDI8BitSynthesiser`
scheme. The Xcode build phase runs
`apps/macos/macos/build_desktop_resources.sh`.

For Vue development, run the Flask backend on port 8000 and then the Vite dev
server:

```bash
PORT=8000 WEB_FLASK_OPEN_BROWSER=0 ./.venv/bin/python3 apps/web-flask/app.py
cd apps/web-vue
npm ci
npm run dev
```

For Vue production builds:

```bash
cd apps/web-vue
npm ci
npm run build
```

The current non-Docker production path runs Flask/Gunicorn from a Python
virtual environment, with Gunicorn bound privately to `127.0.0.1:8000`, systemd
managing the service, and Caddy serving `apps/web-vue/dist` while reverse
proxying `/api/*`, `/static/previews/*`, and `/synthesise*` to that private
Gunicorn listener. Keep the upload directory, job TTL, maximum upload size, and
Gunicorn timeout aligned with the current synthesis job behaviour. See
`deploy/production/README.md` for the Caddy production and rollback examples.

The Docker deployment remains available as an alternate Flask-backend or legacy
fallback path:

```bash
docker compose -f compose.web.yml up -d --build
```

The compose file binds the service to `127.0.0.1:8000` for tunnel-first testing
and builds only the Flask backend/fallback, shared renderer, shared preview
assets, and project licence into the image.

## Dependency and packaging boundaries

- Python renderer dependencies live in `core/python-renderer/requirements.txt`.
- Web-only Python dependencies live in `apps/web-flask/requirements.txt`.
- Production frontend JavaScript dependencies live in `apps/web-vue/package.json`
  and `apps/web-vue/package-lock.json`.
- macOS helper build dependencies live in `apps/macos/requirements-build.txt`.
- Windows NuGet versions live in `apps/windows/Directory.Packages.props`.
- Docker deployment files are scoped to the Flask backend/fallback path.
- Retained native app packaging stays under the relevant app folder.

## Ownership boundaries

- Shared renderer behaviour belongs in `core/python-renderer/`.
- Production web UI belongs under `apps/web-vue/`.
- Flask backend API and legacy Flask-rendered fallback logic belongs under
  `apps/web-flask/`.
- Retained native UI, launch, and packaging logic stays under the relevant
  `apps/` folder.
- Shared binary/media assets belong under `assets/`.
- Repository-wide documentation, audits, and review notes belong under `docs/`.
- Deployment-specific files belong under `deploy/` and root deployment entry
  points such as `compose.web.yml`.
