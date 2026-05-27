# Localisation procedure

Language/语言: English | [简体中文](./localisation.zh-CN.md)

This document is the standard process for adding or changing user-facing
localisation in OctaBit.

## Scope

- Production UI localisation lives in `frontend/src/i18n/`.
- Repository documentation is maintained in English and Simplified Chinese.
- Legacy Flask UI catalogues live in `legacy/web-flask/i18n/` and should only
  change when a task explicitly targets the legacy fallback.
- Paused native macOS and Windows app localisation is out of scope unless those
  apps are explicitly revived or targeted.

## Add a production frontend locale

1. Choose a stable locale code. Prefer a generic language code such as `es` or
   `fr` unless the requested locale needs a region, such as `zh-CN`.
2. Add `frontend/src/i18n/<locale>.json` with the same keys as `en.json`.
3. Add `toolbar.language_option.<locale>` to every production frontend catalog.
   Use the language's native display name, for example `Español`.
4. Update `frontend/src/composables/useLocale.ts` so the locale is imported,
   included in the `Locale` type, added to `translationsByLocale`, and listed
   in `supportedLocales`.
5. Extend `frontend/src/composables/__tests__/useLocale.test.ts` for URL/cookie
   selection when behaviour changes, and keep the catalog parity test covering
   every production frontend catalog.

Do not hardcode new user-facing web copy in Vue components, composables, or
templates. Add a catalog key and call `t(...)`.

## Documentation updates

When the set of production UI languages changes, update existing English and
Simplified Chinese docs that list frontend catalog coverage. At minimum check:

- `README.md`
- `README.zh-CN.md`
- `docs/repository-layout.md`
- `docs/repository-layout.zh-CN.md`
- `AGENTS.md`
- `CLAUDE.md`
- `legacy/web-flask/i18n/README.md`
- `legacy/web-flask/i18n/README.zh-CN.md`

Keep documentation language coverage separate from UI language coverage. The
repository docs remain English and Simplified Chinese even when the production
UI supports more languages.

## Verification

Run the frontend checks after production UI localisation changes:

```bash
cd frontend
npm run test
npm run build
```

Before deployment, confirm the built bundle contains all locale option markers:

```bash
for catalog in frontend/src/i18n/*.json; do
  locale="$(basename "$catalog" .json)"
  grep -R -q "\"toolbar.language_option.$locale\"" frontend/dist/assets
done
```

The production helper `deploy/production/deploy-vue-production.sh` performs the
same marker check against the local build and, when `PUBLIC_URL` is set, against
the public JavaScript bundle after Caddy reload.

