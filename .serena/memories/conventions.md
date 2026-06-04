# Conventions

- Prefer small, localized changes; preserve existing behavior unless request asks for UI, API, or renderer change.
- Treat `frontend/` as production frontend and `backend/` as primary backend.
- Runtime synthesis behavior belongs in `backend/internal/renderer/`.
- Production Vue UI strings: `frontend/src/i18n/*.json`.
- Keep English as fallback; align keys across all frontend catalogs.
- TypeScript component names in PascalCase; follow existing file naming patterns.
