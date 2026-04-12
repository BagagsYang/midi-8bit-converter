# macOS app

Language: English | [简体中文](./README.zh-CN.md)

This folder contains the native macOS app packaging and build notes.

## Build

1. Install full Xcode.
2. Recreate or refresh the repo-local virtual environment: `python3 -m venv .venv`
3. Install the macOS build dependencies: `./.venv/bin/python3 -m pip install -r apps/macos/requirements-build.txt`
4. Open `apps/macos/MIDI8BitSynthesiser.xcodeproj` and run the `MIDI8BitSynthesiser` scheme.

## Launching

- Launch the built `.app` bundle through Xcode, Finder, or `open -na <path-to-app>`.
- Do not execute `MIDI8BitSynthesiser.app/Contents/MacOS/MIDI8BitSynthesiser` directly during manual testing. On recent macOS releases that path can abort inside AppKit/HIServices before the app UI is initialized, which produces a misleading crash report even though the normal app-bundle launch path works.

## How the app works

- SwiftUI provides the native macOS interface.
- The Xcode build phase runs `apps/macos/macos/build_desktop_resources.sh`.
- That script freezes `core/python-renderer/midi_to_wave.py` into a bundled helper binary with PyInstaller.
- The same script copies the canonical preview WAV assets from `assets/previews/` into the app bundle.
- The app launches the bundled helper directly for each queued MIDI file, so no Flask server or browser is involved.

## Current layer controls

- Each layer still has waveform, pulse width, and base volume controls.
- Layers can now optionally enable a frequency-gain curve that is evaluated against each note's fundamental frequency during export.
- The inline curve editor uses a log-frequency x-axis and a dB y-axis, with up to 8 draggable points per layer.
- Export naming matches the current web/core behavior: single-layer exports keep the waveform suffix, multi-layer exports use `_mix`, and curve-bearing exports append a stable hash suffix.
- The existing preview button is still a raw waveform preview; it does not yet render the frequency curve into the preview sound.

## Tests

- The shared `MIDI8BitSynthesiser` scheme now includes a lightweight `MIDI8BitSynthesiserTests` XCTest target for pure model, payload, and filename logic.
