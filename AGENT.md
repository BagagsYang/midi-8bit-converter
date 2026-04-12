# AGENTS.md

## Repository overview
This repository is a monorepo.

- apps/web-flask: legacy Flask web UI
- apps/macos: native SwiftUI macOS app
- apps/windows: native WinUI 3 Windows app
- core/python-renderer: canonical Python renderer
- assets/previews: shared waveform preview assets

## Localisation rules
- Do not duplicate app source trees for Chinese.
- Keep one codebase per platform.
- Extract user-facing strings into platform-appropriate localisation resources.
- Documentation may have parallel Chinese files named `*.zh-CN.md`.
- Prefer English as the fallback locale.
- Preserve existing behaviour unless the task explicitly asks for UI changes.

## Web app rules
- Avoid keeping large inline JS in templates when adding localisation.
- Prefer `i18n/*.json` plus separate JS helpers.
- Do not change synthesis logic in `core/python-renderer` for localisation-only tasks.

## macOS rules
- Use Apple-native localisation files.
- Do not hardcode Chinese strings directly in Swift views.

## Windows rules
- Use WinUI resource files (`.resw`) for localisation.
- Do not hardcode Chinese strings directly in XAML if a resource key can be used.

## Validation
After making changes, run only the checks relevant to the touched app and report what was and was not run.
