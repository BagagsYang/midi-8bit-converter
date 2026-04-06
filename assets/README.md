# Shared Assets

`assets/previews/` is the canonical source of waveform preview WAV files.

Usage:

- `apps/web-flask/` serves these files through a dedicated Flask route.
- `apps/macos/` copies these files into the app bundle at build time.
- `apps/windows/` links these files into the WinUI project at build and publish time.
