# OctaBit Pro

Private monorepo for developing OctaBit Pro and mirroring the OSS subset to `bagags/octabit`.

## Layout

OSS code stays at the repository root:

- `backend/`
- `frontend/`
- `assets/`
- `docs/`
- `legacy/`
- public deployment and contributor files

Pro-only code lives outside OSS package roots:

- `overlays/backend/`: files copied over staged `backend/`
- `overlays/frontend/src/`: files copied over staged `frontend/src/`
- `deploy/pro/`: private production deploy assets
- `scripts/pro/`: private assemble, build, dev, and sync scripts

Do not place Pro-only Go files under `backend/` or Pro-only frontend files under `frontend/`; root OSS checks must remain public-safe.

## Commands

```bash
# OSS checks from the private monorepo root
cd backend && go test ./...
cd frontend && npm ci && npm test && npm run build

# Pro staging and build
scripts/pro/assemble.sh /tmp/octabit-pro-stage
scripts/pro/build.sh

# Pro dev, assembled once
scripts/pro/dev.sh

# Public mirror dry run using a local public clone
OSS_REMOTE_URL=/Users/yangyi/Programming/octabit SYNC_DRY_RUN=1 SYNC_KEEP_WORKDIR=1 scripts/pro/sync-oss.sh
```

## Sync Rule

The public mirror is generated from an allowlist. New public files must be added to `scripts/pro/sync-oss.sh`; new private files must stay under private paths or be explicitly excluded.

## Public Mirror Action

`Sync OSS Mirror` runs after `CI` succeeds on `main`. It requires an `OSS_SYNC_TOKEN` repository secret in `bagags/octabit-pro` with access to push to `bagags/octabit`.

The workflow validates that the secret is present and can read the mirror before running the Pro build. If the mirror commit changes files under `.github/workflows/`, the token must also be allowed to update workflow files.
