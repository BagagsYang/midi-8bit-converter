# Python Renderer

This folder contains the canonical Python MIDI-to-WAV renderer used directly by the Flask app and macOS helper build, and indirectly by the Windows parity tests.

## Public Interface

- Module: `midi_to_wave.py`
- Primary function: `midi_to_audio(midi_path, output_path, sample_rate=48000, layers=None)`
- CLI entrypoint:
  - positional args: input MIDI path, output WAV path
  - options: `--type`, `--duty`, `--rate`, `--layers-json`

## Contract

- Inputs are platform-neutral file paths plus waveform configuration
- Output is a rendered WAV file written to disk
- Invalid configuration should fail with an explicit error instead of falling back silently, except for the documented default single pulse layer when no audible layers are supplied

UI code, packaging code, and platform-specific launch behavior should stay outside this folder.

## Dependency Scope

- `requirements.txt` contains only the renderer/runtime dependencies.
- Web-specific packages live in `apps/web-flask/requirements.txt`.
- macOS helper build packages live in `apps/macos/requirements-build.txt`.
