# Contributing to OctaBit

Language/语言: English | [简体中文](./CONTRIBUTING.zh-CN.md)

Thank you for helping improve OctaBit. This guide explains where contributions
fit in the repository, when to open an issue first, and what to include in a
pull request.

## Start with the right area

OctaBit is a monorepo. The current active contribution targets are:

- `frontend/`: the production Vue browser frontend.
- `backend/`: the primary Go backend API, workspace/synthesis service, and
  Go renderer.
- `legacy/web-flask/`: the legacy Flask backend/API and Flask-rendered frontend
  fallback retained for parity reference.
- `legacy/python-renderer/`: the canonical Python MIDI-to-WAV renderer retained
  for parity reference.
- `docs/`, `deploy/production/`, `deploy/web-flask/`, and `assets/previews/`: supporting
  documentation, deployment, and shared asset areas.

The native macOS and Windows apps under `legacy/native/macos/` and
`legacy/native/windows/` are paused/reference areas. Please open an issue
before starting substantial work there so maintainers can confirm the scope.

For a fuller map of the repository, see [docs/repository-layout.md](./docs/repository-layout.md).

## Before you start

Small documentation fixes and narrow bug fixes may go straight to a pull
request.

Please open an issue first for substantial changes, including:

- public web UI or workflow changes;
- renderer behavior, renderer schema, or output naming changes;
- deployment, Docker, or server runtime changes;
- architecture, repository layout, or dependency changes;
- licensing-sensitive work, vendored assets, generated media, or new third-party
  material;
- macOS or Windows app changes.

In the issue, describe the problem, the intended behavior, and the area of the
repository you expect to touch.

## Branch workflow

OctaBit uses a simple long-term branch workflow:

- `main` is the stable, deployable branch used by the live server. Keep it in a
  state that can be deployed.
- `dev` is the active development branch for ongoing work and larger changes.
- Ordinary development should happen on `dev` or on short-lived feature
  branches based on `dev`.
- Merge larger changes into `main` only after they have been reviewed and
  tested on `dev` or a feature branch.

Avoid long-lived release branches or heavyweight process unless maintainers
explicitly agree that a specific change needs it.

## Development setup

Run commands from the repository root unless a document says otherwise.

Create the local Python environment:

```bash
python3 -m venv .venv
```

Install only the dependencies needed for the area you are touching:

```bash
./.venv/bin/python3 -m pip install -r legacy/web-flask/requirements.txt
./.venv/bin/python3 -m pip install -r legacy/python-renderer/requirements.txt
```

For area-specific notes, start with:

- [frontend/README.md](./frontend/README.md)
- [backend/README.md](./backend/README.md)
- [legacy/web-flask/README.md](./legacy/web-flask/README.md)
- [legacy/python-renderer/README.md](./legacy/python-renderer/README.md)
- [deploy/production/README.md](./deploy/production/README.md)

## Making changes

- Keep the Vue app as the production public frontend.
- Keep the Go backend as the primary API and synthesis service.
- Keep shared synthesis behavior in `backend/internal/renderer/` and the parity
  reference in `legacy/python-renderer/`.
- Do not duplicate app source trees for localisation. Use the existing
  localisation resources for the touched platform.
- For `frontend/`, prefer `src/i18n/*.json` for user-facing UI strings.
- For the legacy Flask-rendered UI in `legacy/web-flask/`, prefer `i18n/*.json`
  plus separate static JS/CSS over adding large inline scripts or hardcoded
  user-facing strings in templates.
- Keep English and Simplified Chinese documentation pairs aligned when changing
  paired docs.
- Avoid unrelated refactors in feature or bug-fix pull requests.

## Commit convention

Follow [Conventional Commits](https://www.conventionalcommits.org/): `type(scope?): description`.
Common types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`.

Example: `feat(backend): add export-to-MP3 endpoint`, `fix(ui): correct layer chip alignment`.

## Pull request checklist

Before opening a pull request, make sure it includes:

- a clear summary of the problem and the change;
- the main repository areas touched;
- any user-facing behavior, UI, deployment, or compatibility impact;
- screenshots or short notes for visible web UI changes;
- the checks you ran, plus any relevant checks you could not run;
- provenance and license notes for new dependencies, vendored assets, generated
  media, or other third-party material.

## Validation

Run the checks relevant to the area you touched and report the result in the
pull request.

For the Go backend:

```bash
cd backend && go test ./...
```

For the Vue frontend:

```bash
cd frontend && npm run build
```

For the legacy Python code:

```bash
./.venv/bin/python3 -m unittest discover -s legacy/web-flask/tests
./.venv/bin/python3 -m unittest discover -s legacy/python-renderer/tests
```

For documentation-only changes, proofread the affected files and keep English
and Chinese versions aligned when both exist.

For deployment changes, include static review of the changed deployment files
and any available Docker or Compose validation.

For native app changes, run the checks documented in the relevant app README
when your machine has the required tools. If you cannot run a check because the
environment is missing Xcode, .NET, Docker, or another dependency, say that in
the pull request.

## Licensing and provenance

OctaBit is licensed under the GNU Affero General Public License v3.0 or later
(`AGPL-3.0-or-later`). See [LICENSE.md](./LICENSE.md).

By contributing, you confirm that you have the right to submit your code,
documentation, assets, and other materials, and that they are compatible with
the repository license. For new dependencies, vendored assets, generated media,
or licensing-sensitive files, include the source and license information in the
pull request.
