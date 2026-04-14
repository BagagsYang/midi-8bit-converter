# macOS app

Language/语言: English | [简体中文](./README.zh-CN.md)

This folder contains the native macOS app packaging and build notes.

## Build

1. Install full Xcode.
2. Recreate or refresh the repo-local virtual environment: `python3 -m venv .venv`
3. Install the macOS build dependencies: `./.venv/bin/python3 -m pip install -r apps/macos/requirements-build.txt`
4. Open `apps/macos/MIDI8BitSynthesiser.xcodeproj` and run the `MIDI8BitSynthesiser` scheme.

## How the app works

- SwiftUI provides the native macOS interface.
- The Xcode build phase runs `apps/macos/macos/build_desktop_resources.sh`.
- That script freezes `core/python-renderer/midi_to_wave.py` into a bundled helper binary with PyInstaller.
- The same script copies the canonical preview WAV assets from `assets/previews/` into the app bundle.
- The app launches the bundled helper directly for each queued MIDI file, so no Flask server or browser is involved.
