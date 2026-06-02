# Legacy Core

- `legacy/web-flask/` is a legacy Flask backend/API plus Flask-rendered frontend fallback retained for parity fixtures and fallback reference.
- `legacy/python-renderer/` is the canonical Python MIDI-to-WAV parity reference and renderer test area.
- `legacy/native/macos/` and `legacy/native/windows/` are deprecated/paused native app references.
- For legacy Flask UI strings use `legacy/web-flask/i18n/*.json`; keep `en.json`, `fr.json`, and `zh-CN.json` aligned when touching that catalog set.
- Run legacy Python tests only when touching `legacy/web-flask/`, `legacy/python-renderer/`, or fixture regeneration behavior.
