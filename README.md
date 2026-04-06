# MIDI-8bit Synthesiser

This repository is a reorganised monorepo for the MIDI-8bit Synthesiser product family. Platform-specific apps live under `apps/`, the Python reference renderer lives under `core/`, and shared preview assets live under `assets/`.

## Layout

| Folder | Responsibility |
| --- | --- |
| `apps/web-flask/` | Legacy Flask/browser UI |
| `apps/macos/` | Native macOS SwiftUI app and Xcode project |
| `apps/windows/` | Native Windows WinUI 3 solution, C# renderer, installer |
| `apps/desktop/` | Reserved placeholder for future desktop packaging work |
| `core/python-renderer/` | Canonical Python MIDI-to-WAV renderer and parity reference |
| `assets/previews/` | Canonical waveform preview WAV files used by all apps |
| `docs/` | Reviews and repository structure notes |

## Shared Contract

- Canonical renderer entrypoint: `core/python-renderer/midi_to_wave.py`
- Stable inputs: MIDI path, output WAV path, sample rate, waveform layers
- Stable output: rendered WAV file or explicit error
- Windows intentionally keeps a native C# implementation and validates it against the Python renderer in parity tests

## Build Notes

Create the repo-local environment at the repository root:

```bash
python3 -m venv .venv
```

Install only the dependencies needed for the app you are working on:

- Web UI:
  `./.venv/bin/python3 -m pip install -r apps/web-flask/requirements.txt`
- macOS helper build:
  `./.venv/bin/python3 -m pip install -r apps/macos/requirements-build.txt`
- Windows parity tests:
  `./.venv/bin/python3 -m pip install -r core/python-renderer/requirements.txt`

App-specific instructions live in:

- `apps/web-flask/README.md`
- `apps/macos/macos/README.md`
- `apps/windows/README.md`

Repository layout notes live in `docs/repository-layout.md`.
