# Web API contract

Language/语言: English | [简体中文](./api-contract.zh-Hans.md)

This document describes the browser-facing API boundary for the Go OctaBit
backend. The production web frontend is the Vue app in `frontend/`, served by
Caddy from its Vite `dist` build. The Go service is the private backend API,
workspace/synthesis service, preview provider, and runtime renderer. The
machine-readable contract lives in [`openapi.yaml`](./openapi.yaml).

## Scope

- Production frontend code should use the `/api/*` routes.
- In production, Caddy should reverse proxy `/api/*` and `/static/previews/*`
  to the Go backend on `127.0.0.1:8000`.
- Anonymous temporary workspaces use a random HttpOnly cookie named
  `octabit_workspace`.
- SQLite stores workspace metadata. MIDI and WAV blobs stay on the filesystem.
- Browser-visible `file_id` and `job_id` values are random opaque UUID-hex
  identifiers. Internal numeric SQLite ids are never returned by the API.
- Routes containing `<file_id>` or `<job_id>` enforce ownership through:
  cookie token -> token hash -> active workspace row -> owned resource row.

## Configuration

- `WEB_SYNTHESISE_JOB_ROOT`: workspace database, upload, and WAV storage root.
  Default: system temp directory plus `octabit-jobs`.
- `WEB_WORKSPACE_TTL_SECONDS`: anonymous workspace retention after last
  activity. Default: `86400`.
- `WEB_WORKSPACE_MAX_QUEUED_FILES`: queued MIDI file cap per workspace. Default:
  `20`.
- `WEB_WORKSPACE_MAX_UPLOAD_BYTES`: total active MIDI upload bytes per
  workspace. Default: `104857600`.
- `WEB_WORKSPACE_MAX_CONVERTED_FILES`: active converted job cap per workspace.
  Default: `20`.
- `WEB_MAX_UPLOAD_BYTES`: per-request upload cap. Default: `20971520`.
- Supported upload extensions: `.mid`, `.midi`.
- Supported sample rates: `44100`, `48000`, `96000`.

SQLite connections are opened per request or operation, enable
`PRAGMA foreign_keys=ON`, use WAL journal mode, and set `PRAGMA busy_timeout=5000`
to reduce multi-worker lock failures.

## Common response types

`ApiError`:

```json
{
  "error": {
    "code": "invalid_layers",
    "message": "Layer 1 frequency_curve frequencies must be strictly increasing."
  }
}
```

`WorkspaceConfigV1`:

```json
{
  "schema": "octabit.workspace_config.v1",
  "sample_rate": 48000,
  "layers": [
    {
      "type": "pulse",
      "duty": 0.5,
      "volume": 1.0,
      "curve_enabled": false,
      "frequency_curve": [
        {"frequency_hz": 8.175798915643707, "gain_db": 0.0},
        {"frequency_hz": 12543.853951415975, "gain_db": 0.0}
      ]
    }
  ]
}
```

Validation:

- `sample_rate`: `44100`, `48000`, or `96000`
- `layers`: 1 to 4 layer objects
- `type`: `pulse`, `sine`, `sawtooth`, or `triangle`
- `duty`: `0.01` to `0.99`
- `volume`: `0.0` to `2.0`
- `curve_enabled`: boolean
- `frequency_curve`: renderer-validated frequency/gain points; ignored for
  rendering when `curve_enabled` is `false`

`WorkspaceStateResponse`:

```json
{
  "workspace": {
    "expires_at": 1770000000.0
  },
  "limits": {
    "max_queued_files": 20,
    "max_upload_bytes": 104857600,
    "max_converted_files": 20
  },
  "config": {
    "schema": "octabit.workspace_config.v1",
    "sample_rate": 48000,
    "layers": [
      {"type": "pulse", "duty": 0.5, "volume": 1.0, "curve_enabled": false, "frequency_curve": []}
    ]
  },
  "uploads": [
    {
      "file_id": "0123456789abcdef0123456789abcdef",
      "name": "lead.mid",
      "size": 12345,
      "created_at": 1770000000.0
    }
  ],
  "converted_files": [
    {
      "job_id": "abcdef0123456789abcdef0123456789",
      "name": "lead_pulse.wav",
      "source_name": "lead.mid",
      "size": 123456,
      "download_url": "/api/synthesis-jobs/abcdef0123456789abcdef0123456789/download",
      "delete_url": "/api/synthesis-jobs/abcdef0123456789abcdef0123456789",
      "created_at": 1770000000.0,
      "updated_at": 1770000001.0,
      "expires_at": 1770086401.0
    }
  ]
}
```

`WorkspaceUploadResponse`:

```json
{
  "upload": {
    "file_id": "0123456789abcdef0123456789abcdef",
    "name": "lead.mid",
    "size": 12345,
    "created_at": 1770000000.0
  }
}
```

`WorkspaceQueueResponse`:

```json
{
  "uploads": [
    {
      "file_id": "0123456789abcdef0123456789abcdef",
      "name": "lead.mid",
      "size": 12345,
      "created_at": 1770000000.0
    }
  ]
}
```

`WorkspaceConfigResponse`:

```json
{
  "config": {
    "schema": "octabit.workspace_config.v1",
    "sample_rate": 48000,
    "layers": [
      {"type": "pulse", "duty": 0.5, "volume": 1.0, "curve_enabled": false, "frequency_curve": []}
    ]
  }
}
```

`SynthesisJobResponse`:

```json
{
  "job_id": "abcdef0123456789abcdef0123456789",
  "status": "ready",
  "source_name": "lead.mid",
  "created_at": 1770000000.0,
  "updated_at": 1770000001.0,
  "expires_at": 1770086401.0,
  "download_name": "lead_pulse.wav",
  "size_bytes": 123456,
  "download_url": "/api/synthesis-jobs/abcdef0123456789abcdef0123456789/download",
  "delete_url": "/api/synthesis-jobs/abcdef0123456789abcdef0123456789"
}
```

Statuses are `queued`, `rendering`, `ready`, `failed`, and `expired`.

## Endpoints

### `GET /api/health`

Receives: no body.

Success:

- Status: `200`
- Type: JSON

```json
{
  "status": "ok",
  "service": "octabit-web"
}
```

### `GET /api/workspace`

Receives: no body.

Behavior:

- Missing, invalid, unknown, or expired workspace cookies create a fresh empty
  workspace.
- A new workspace response sets the `octabit_workspace` cookie.

Success:

- Status: `200`
- Type: `WorkspaceStateResponse`

### `POST /api/workspace/uploads`

Receives:

- Type: `multipart/form-data`
- `midi_file`: exactly one `.mid` or `.midi` file

Success:

- Status: `201`
- Type: `WorkspaceUploadResponse`

Errors:

- `400` with code `missing_midi_file`
- `400` with code `no_selected_file`
- `400` with code `empty_midi_file`
- `409` with code `workspace_queue_limit`
- `410` with code `workspace_expired`
- `413` with code `upload_too_large`
- `413` with code `workspace_upload_bytes_limit`
- `415` with code `unsupported_file_type`
- `500` with code `internal_error`

### `DELETE /api/workspace/uploads/<file_id>`

Receives: path `file_id`, no body.

Success:

- Status: `204`
- Body: empty

Errors:

- `400` with code `invalid_file_id`
- `404` with code `not_found`
- `410` with code `workspace_expired`

### `PATCH /api/workspace/queue`

Receives:

```json
{
  "file_ids": ["0123456789abcdef0123456789abcdef"]
}
```

The request must contain the exact active upload id set once each.

Success:

- Status: `200`
- Type: `WorkspaceQueueResponse`

Errors:

- `400` with code `invalid_file_id`
- `410` with code `workspace_expired`
- `422` with code `invalid_queue`

### `PUT /api/workspace/config`

Receives: `WorkspaceConfigV1`.

Success:

- Status: `200`
- Type: `WorkspaceConfigResponse`

Errors:

- `410` with code `workspace_expired`
- `422` with code `invalid_workspace_config`

### `POST /api/synthesis-jobs`

Creates a render job.

Preferred workspace request:

```json
{
  "file_id": "0123456789abcdef0123456789abcdef",
  "config": {
    "schema": "octabit.workspace_config.v1",
    "sample_rate": 48000,
    "layers": [
      {"type": "pulse", "duty": 0.5, "volume": 1.0, "curve_enabled": false, "frequency_curve": []}
    ]
  }
}
```

Compatibility request:

- Type: `multipart/form-data`
- `midi_file`: `.mid` or `.midi` file
- `rate`: `44100`, `48000`, or `96000`
- `layers_json`: JSON array of renderer layer objects

Success:

- Status: `202`
- Type: `SynthesisJobResponse`

Errors:

- `400` with code `invalid_file_id`
- `400` with code `missing_midi_file`
- `400` with code `no_selected_file`
- `400` with code `empty_midi_file`
- `404` with code `not_found`
- `409` with code `workspace_converted_limit`
- `410` with code `workspace_expired`
- `413` with code `upload_too_large`
- `415` with code `unsupported_file_type`
- `429` with code `render_queue_full`
- `422` with code `invalid_sample_rate`
- `422` with code `invalid_layers`
- `422` with code `invalid_workspace_config`
- `500` with code `internal_error`

### `GET /api/synthesis-jobs/<job_id>`

Receives: path `job_id`, no body.

Success:

- Status: `200`
- Type: `SynthesisJobResponse`

Errors and terminal states:

- `400` with code `invalid_job_id`
- `404` with code `not_found`
- `410` with code `workspace_expired`
- `410` JSON with `{"job_id": "...", "status": "expired"}` when the owned job expired

### `GET /api/synthesis-jobs/<job_id>/download`

Receives: path `job_id`, no body.

Success:

- Status: `200`
- Type: WAV file attachment

Errors and terminal states:

- `400` with code `invalid_job_id`
- `400` JSON `SynthesisJobResponse` when the owned job failed
- `404` with code `not_found`
- `409` JSON `SynthesisJobResponse` when the owned job is not ready yet
- `410` with code `workspace_expired`
- `410` JSON with `{"job_id": "...", "status": "expired"}` when the owned job expired

### `DELETE /api/synthesis-jobs/<job_id>`

Receives: path `job_id`, no body.

Success:

- Status: `204`
- Body: empty

Errors:

- `400` with code `invalid_job_id`
- `404` with code `not_found`
- `410` with code `workspace_expired`

