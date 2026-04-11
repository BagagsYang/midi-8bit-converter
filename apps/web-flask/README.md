# Web Flask App

This folder contains the legacy browser-distributed version of the MIDI-8bit Synthesiser.

## Responsibilities

- Flask entrypoint and request handling
- HTML templates and web-specific static assets
- Legacy launcher script
- Browser UI only; synthesis is delegated to the Python renderer in `../../core/python-renderer/`
- Phase 1 proving UI for per-layer frequency-gain curves

## Run

From the repository root:

```bash
python3 -m venv .venv
./.venv/bin/python3 -m pip install -r apps/web-flask/requirements.txt
./.venv/bin/python3 apps/web-flask/app.py
```

Or launch the helper script:

```bash
apps/web-flask/Launch_Synthesiser.command
```

## Shared Dependencies

- Renderer: `../../core/python-renderer/midi_to_wave.py`
- Canonical preview assets: `../../assets/previews/`

This app serves preview WAVs from the shared asset folder and should not duplicate renderer logic.

## Current Upload Contract

`POST /synthesise` uses `multipart/form-data` with:

- `midi_file`: uploaded `.mid` or `.midi`
- `rate`: sample rate integer
- `layers_json`: JSON array of layer objects

Each layer object contains:

- `type`
- `duty`
- `volume`
- `frequency_curve`: optional array of `{frequency_hz, gain_db}` points

The browser UI stores layer state in JavaScript and serialises it into `layers_json`.

## Output Naming

- Single audible layer without a curve: `<original>_<wave>.wav`
- Multiple audible layers without a curve: `<original>_mix.wav`
- Any audible layer with a non-empty frequency curve: `<original>_<base>_<hash>.wav`

The hash is derived from the sanitised layer payload so different curve settings do not reuse the same export name.
