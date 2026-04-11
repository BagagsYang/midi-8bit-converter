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

## Layer Schema

Each layer contains:

- `type`: one of `pulse`, `sine`, `sawtooth`, `triangle`
- `duty`: pulse width, validated between `0.01` and `0.99`
- `volume`: linear base gain, validated as `>= 0`
- `frequency_curve`: optional array of `{frequency_hz, gain_db}` points

Frequency curves are evaluated against each rendered note's fundamental frequency.
The evaluated curve gain multiplies the layer's base `volume` for that note.

Curve rules:

- Missing, `null`, or empty `frequency_curve` means `0 dB` everywhere
- Supported note-frequency range: `8.1757989156 Hz` to `12543.8539514 Hz`
- Supported gain range: `-36 dB` to `+12 dB`
- Up to `8` curve points per layer
- Points are sorted by ascending `frequency_hz`
- Interpolation is linear in `gain_db` over a logarithmic frequency axis
- Values below the first point clamp to the first point's gain
- Values above the last point clamp to the last point's gain

UI code, packaging code, and platform-specific launch behavior should stay outside this folder.

## Dependency Scope

- `requirements.txt` contains only the renderer/runtime dependencies.
- Web-specific packages live in `apps/web-flask/requirements.txt`.
- macOS helper build packages live in `apps/macos/requirements-build.txt`.
