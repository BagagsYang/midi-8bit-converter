# Frontend Core

- `frontend/` is the production frontend.
- Vue 3 + Vite + TypeScript app; entrypoint `frontend/src/main.ts` mounts `App.vue`.
- Runtime API surface should stay on stable `/api/*` contract backed by the Go service.
- Production output is Vite `dist`.
- UI/localization strings live in `frontend/src/i18n/*.json`; align `en.json`, `fr.json`, and `zh-CN.json` keys when touching a frontend catalog set.
- Locale composable imports `en`, `fr`, and `zh-CN` from `frontend/src/composables/useLocale.ts`.
