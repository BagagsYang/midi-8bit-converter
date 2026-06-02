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

## Tagging a Release

Tags live in this monorepo. Prefix decides whether the tag reaches the public mirror:

| Prefix | Destination | Example |
|--------|-------------|---------|
| `v*` | `bagags/octabit` (public) | `v2.4.0` |
| `pro-v*` | private only | `pro-v1.0.0` |

### OSS release (`v*`)

1. Ensure CI is green on `main` and the sync workflow has completed.
2. Create an annotated tag and push it:
   ```bash
   git tag -a v2.4.0 -m "v2.4.0"
   git push origin v2.4.0
   ```
3. The `sync-oss.sh` script (run by CI) detects `v*` tags at HEAD and pushes them
   to `bagags/octabit`. If CI already ran before tagging, trigger a re-run or run
   the sync script locally with a valid `OSS_SYNC_TOKEN`.
4. Create the GitHub Release on `bagags/octabit`:
   ```bash
   gh release create v2.4.0 --repo bagags/octabit --title "v2.4.0" --notes "..."
   ```
   Use the `/tag-release` skill when available — it automates steps 2–4.

### Pro release (`pro-v*`)

1. Ensure `scripts/pro/build.sh` passes.
2. Tag and push — the tag stays in this repo:
   ```bash
   git tag -a pro-v1.0.0 -m "pro-v1.0.0"
   git push origin pro-v1.0.0
   ```
3. No GitHub Release needed. Document the change in your internal changelog.

### After tagging

- Bump version strings in source if any (check `frontend/package.json`, etc.).
- The next OSS sync automatically filters out `pro-*` tags — no risk of leaking
  Pro version info to the public mirror.
