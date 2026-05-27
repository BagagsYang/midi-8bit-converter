# Web UI i18n

Language/语言: English | [简体中文](./README.zh-CN.md)

The legacy Flask-rendered frontend uses the JSON catalogs in this directory for
Flask-rendered HTML and the inline browser UI in `templates/index.html`. The
production Vue frontend keeps its catalogs in `frontend/src/i18n/` for English,
Spanish, French, and Simplified Chinese. Keep the legacy Flask catalog keys
aligned across `en.json`, `fr.json`, and `zh-CN.json`; English remains the
fallback locale.

## French slice coverage

The first French slice covers the visible browser UI already routed through the
shared catalog: the language selector, settings dialog, theme controls, MIDI queue controls,
layer and curve controls, waveform labels, processing status and alerts, and
the Flask missing-file validation errors used by `/synthesise`.

## Deferred strings

- Web Flask documentation and launcher text remain in their existing English
  and Simplified Chinese documents for this slice.
- Browser console warnings, IndexedDB/localStorage keys, and JavaScript
  function names remain internal developer strings.
- Technical values such as `Hz`, `dB`, waveform payload values, generated file
  names, and preview asset names remain locale-neutral.
- Native macOS and Windows app strings are deliberately out of scope for this
  web-only slice.
