# Web UI i18n

The web app uses the JSON catalogs in this directory for both Flask-rendered
HTML and the inline browser UI in `templates/index.html`. Keep the key sets in
`en.json`, `fr.json`, and `zh-CN.json` aligned; English remains the fallback
locale.

## French slice coverage

The first French slice covers the visible browser UI already routed through the
shared catalog: the language selector, theme tooltip, MIDI queue controls,
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
