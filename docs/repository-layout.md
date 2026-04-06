# Repository Layout

This repository stays as one monorepo, but the layout is now explicit:

- `apps/web-flask/`: legacy Flask/browser UI
- `apps/macos/`: Xcode project, SwiftUI app, macOS build helper
- `apps/windows/`: WinUI 3 solution, native C# renderer, installer, parity tests
- `apps/desktop/`: reserved placeholder
- `core/python-renderer/`: canonical Python renderer
- `assets/previews/`: canonical preview WAV assets
- `docs/reviews/`: review artifacts and reports

Conventions:

- Platform-specific UI, packaging, and release logic stays under `apps/`.
- Shared renderer logic stays under `core/`.
- Shared binary or media assets stay under `assets/`.
- Root-level files are reserved for repo-wide configuration and documentation.
