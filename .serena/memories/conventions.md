# Conventions

- Prefer small, localized changes; preserve existing behavior unless request asks for UI, API, or renderer change.
- Treat `frontend/` as production frontend and `backend/` as primary backend.
- Treat `legacy/*` areas as legacy/parity references unless explicitly targeted.
- Runtime synthesis behavior belongs in `backend/internal/renderer/`; Python parity reference behavior belongs in `legacy/python-renderer/`.
- Production Vue UI strings: `frontend/src/i18n/*.json`.
- Legacy Flask-rendered UI strings: `legacy/web-flask/i18n/*.json`.
- Keep English as fallback; align `en.json`, `fr.json`, and `zh-CN.json` keys in any catalog set touched.
- Descriptive Go/Python names; TypeScript component names in PascalCase; follow existing file naming patterns.
- Python tests use `test_*.py` naming.
