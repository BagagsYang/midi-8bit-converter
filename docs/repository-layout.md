# Repository layout

Language/语言: English | [简体中文](./repository-layout.zh-CN.md)

This repository stays as one monorepo, and the layout is explicit.

## Layout

- `apps/web-flask/`: legacy Flask/browser UI
- `apps/macos/`: Xcode project, SwiftUI app, macOS build helper
- `apps/windows/`: WinUI 3 solution, native C# renderer, installer, parity tests
- `apps/desktop/`: reserved placeholder
- `core/python-renderer/`: canonical Python renderer
- `assets/previews/`: canonical preview WAV assets
- `docs/reviews/`: review artifacts and reports

## Notes

- Platform-specific UI, packaging, and release logic stays under `apps/`.
- Shared renderer logic stays under `core/`.
- Shared binary or media assets stay under `assets/`.
- Root-level files are reserved for repo-wide configuration and documentation.
