# Task Completion

- Run checks for the touched area and report skipped checks.
- Backend changes: from `backend/`, run `go test ./...`.
- Frontend changes: from `frontend/`, run `npm run build`; run `npm test`/`npm run test` when tests are relevant.
- Web API or localization changes: add render-level or endpoint assertions where practical.
- Visible UI changes should be browser-verified and include screenshots or a concise visual QA summary when relevant.
- For dependency, vendored asset, or generated media additions, note source/license details.
