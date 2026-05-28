# Localisation procedure

This agent-facing document is the standard process for adding or changing
user-facing localisation in OctaBit. Its role is similar to `AGENTS.md` and
`CLAUDE.md`: keep it concise, operational, and directly executable by another
agent. It is intentionally maintained in English only.

## Scope

- Production UI localisation lives in `frontend/src/i18n/`.
- Repository product documentation is maintained in English and Simplified
  Chinese, but agent instruction docs such as this file are English-only.
- Legacy Flask UI catalogues live in `legacy/web-flask/i18n/` and should only
  change when a task explicitly targets the legacy fallback.
- Paused native macOS and Windows app localisation is out of scope unless those
  apps are explicitly revived or targeted.

## Agent checklist for a production frontend locale

Follow this checklist for every new production UI language. Do not skip steps
because the deploy script derives its locale checks from the catalog files.

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

Keep deploy logic locale-agnostic. A normal locale addition should not change
`deploy/production/deploy-vue-production.sh`; the script already discovers
locales from `frontend/src/i18n/*.json` and checks every
`toolbar.language_option.<locale>` marker.

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

If Caddy serves a directory other than `frontend/dist`, set `WEB_ROOT` when
running the helper so the built files are published before the public check:

```bash
WEB_ROOT=/var/www/octabit deploy/production/deploy-vue-production.sh
```

`WEB_ROOT` must point at a dedicated static web root. The helper syncs
`frontend/dist/` there with `rsync --delete`, then verifies locale markers in
the published assets. Leave `WEB_ROOT` unset when Caddy serves
`$APP_DIR/frontend/dist` directly.
