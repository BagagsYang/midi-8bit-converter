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

## Tagging Convention

| Prefix | Product | Synced to `bagags/octabit`? |
|--------|---------|---------------------------|
| `v*` | OctaBit OSS | Yes — `sync-oss.sh` pushes tags to public mirror |
| `pro-v*` | OctaBit Pro | No — stays in private monorepo |

Tags are created on the monorepo and follow [Semantic Versioning](https://semver.org/).
When a `v*` tag is pushed (or the commit that carries it reaches `main`), the sync workflow
creates the matching tag in `bagags/octabit`. GitHub Releases on the public repo are created
separately via `gh release create` or the GitHub UI.

To tag a release:
```bash
# OSS release — tag will be synced to public mirror
git tag -a v2.4.0 -m "v2.4.0"
git push origin v2.4.0

# Pro release — tag stays private
git tag -a pro-v1.0.0 -m "pro-v1.0.0"
git push origin pro-v1.0.0
```

## Public Mirror Action

`Sync OSS Mirror` runs after `CI` succeeds on `main`. It requires an `OSS_SYNC_TOKEN` repository secret in the private upstream repository with access to push to `bagags/octabit`.

The workflow validates that the secret is present and can read the mirror before running the Pro build. If the mirror commit changes files under `.github/workflows/`, the token must also be allowed to update workflow files.
