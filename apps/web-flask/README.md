# Web Flask App

This folder contains the legacy browser-distributed version of the MIDI-8bit Synthesiser.

## Responsibilities

- Flask entrypoint and request handling
- HTML templates and web-specific static assets
- Legacy launcher script
- Browser UI only; synthesis is delegated to the Python renderer in `../../core/python-renderer/`

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

On Windows, use:

```bat
apps\web-flask\Launch_Synthesiser.bat
```

## Shared Dependencies

- Renderer: `../../core/python-renderer/midi_to_wave.py`
- Canonical preview assets: `../../assets/previews/`

This app serves preview WAVs from the shared asset folder and should not duplicate renderer logic.
