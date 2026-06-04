# OctaBit Go Backend

This directory is the primary Go backend runtime. It implements the stable
frontend-facing API, workspace storage, job lifecycle, preview serving, and
MIDI-to-WAV rendering without requiring Python at runtime.

Current scope:

- `cmd/server`: process startup, environment config, SQLite workspace store
  opening, logging, graceful shutdown, and HTTP server wiring.
- `internal/config`: defaults and environment-variable parsing.
- `internal/httpapi`: HTTP routes for `/api/health`, the workspace
  state/upload/queue/config API, workspace-backed `/api/synthesis-jobs` JSON
  and multipart render/poll/download/delete flows, and `/static/previews/*`,
  with route tests and OpenAPI route-registration conformance.
- `internal/jobs`: bounded render execution and in-memory render job lifecycle
  tests for queue semantics shared with the workspace-backed job flow.
- `internal/midi`: Standard MIDI File note extraction through
  `gitlab.com/gomidi/midi/v2/smf`.
- `internal/renderer`: renderer limits, layer validation, frequency curve
  interpolation, curve hashes, output naming, and note-event PCM/WAV synthesis.
- `internal/storage`: SQLite schema, connection pragmas, token hashing, path
  helpers, and workspace cleanup/cascade behavior.
- `internal/workspace`: workspace token lifecycle, state payloads, upload queue
  operations, limits, config persistence, SQLite-backed synthesis jobs, WAV
  output cleanup, and renderer form payload conversion.
- `testdata/python-baseline`: frozen Python parity fixtures for renderer
  conformance testing.

Renderer note: layer `volume` and frequency-curve gain are applied before the
final peak normalisation step. A single-layer volume change, or a flat curve
that only changes every note by the same gain, can therefore produce
byte-identical WAV output. Waveform type, pulse duty, non-flat curves, and
relative gains between multiple audible layers still change the rendered audio.

Run the current Go checks from this directory:

```bash
go test ./...
```

Start the server:

```bash
cd backend
PORT=8000 go run ./cmd/server
```
