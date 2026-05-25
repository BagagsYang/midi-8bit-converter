# OctaBit Go Backend

This directory is the primary Go backend runtime. It implements the stable
frontend-facing API, legacy synthesis compatibility routes, workspace storage,
job lifecycle, preview serving, and MIDI-to-WAV rendering without requiring
Python at runtime.

Current scope:

- `cmd/server`: process startup, environment config, SQLite workspace store
  opening, logging, graceful shutdown, and HTTP server wiring.
- `internal/config`: defaults and environment-variable parsing aligned with the
  existing Flask backend.
- `internal/httpapi`: initial HTTP routes for `/api/health`, the workspace
  state/upload/queue/config API, workspace-backed `/api/synthesis-jobs` JSON
  and multipart render/poll/download/delete flows, legacy
  `/synthesise` plus `/synthesise/jobs` render/poll/download/delete routes,
  and `/static/previews/*`, with route tests, OpenAPI route-registration
  conformance, and Python-transcript replay for the routes currently covered by
  frozen fixtures.
- `internal/jobs`: bounded render execution, legacy job file storage, and
  in-memory render job lifecycle tests for queue semantics shared with the
  workspace-backed job flow.
- `internal/midi`: Standard MIDI File note extraction through
  `gitlab.com/gomidi/midi/v2/smf`, selected by fixture parity against
  PrettyMIDI-derived notes.
- `internal/renderer`: renderer limits, layer validation, frequency curve
  interpolation, curve hashes, output naming, and note-event PCM/WAV synthesis
  aligned with the Python baseline.
- `internal/storage`: SQLite schema, connection pragmas, token hashing, path
  helpers, and workspace cleanup/cascade behavior aligned with the Flask
  storage model.
- `internal/workspace`: workspace token lifecycle, state payloads, upload queue
  operations, limits, config persistence, SQLite-backed synthesis jobs, WAV
  output cleanup, and renderer form payload conversion aligned with the Python
  baseline.
- `testdata/python-baseline`: frozen Python parity fixtures generated from the
  current Flask backend and Python renderer.

Run the current Go checks from this directory:

```bash
go test ./...
```

Start the server:

```bash
cd backend
PORT=8000 go run ./cmd/server
```

Python is used only for the explicit legacy fixture-regeneration workflow in
`scripts/generate_python_parity_fixtures.py`; routine Go tests read the frozen
fixtures directly.
