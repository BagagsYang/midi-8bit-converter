# OctaBit Pro Agent Notes

Use `AGENTS.md` and `CLAUDE.md` for public OctaBit architecture. Use this file only for private Pro workflow details.

## Private Boundaries

- Pro-only source belongs under `overlays/`.
- Pro deploy assets belong under `deploy/pro/`.
- Pro tooling belongs under `scripts/pro/`.
- Keep root `backend/` and `frontend/` public-safe so OSS tests and the public mirror remain clean.
- Before changing an OSS file with a Pro replacement, check the matching overlay and update both when behavior must stay aligned.

## Required Checks

Run touched-area OSS checks plus the Pro assembled build:

```bash
cd backend && go test ./...
cd frontend && npm test && npm run build
scripts/pro/build.sh
```

For public mirror changes, run a dry sync and scan for Pro markers before pushing.
