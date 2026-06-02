# Python Baseline Fixtures

These fixtures freeze the current Flask backend and Python renderer behavior for
the Go backend migration. They are parity inputs for future Go tests, not a new
runtime dependency.

Regenerate them only when intentionally updating the Python baseline:

```bash
./.venv/bin/python3 scripts/generate_python_parity_fixtures.py
```

The generator rewrites this directory from the current checkout. Routine
post-migration Go tests should read the generated fixture files directly and
must not require Python unless they are explicitly fixture-regeneration tests.

Current fixture groups:

- `api/implemented_routes.json`: Python baseline transcript for routes already
  implemented by the Go backend.
- `api/workspace_flow.json`, `api/error_responses.json`, and
  `api/legacy_jobs.json`: fuller API parity targets for future workspace/job
  implementation.
- `renderer/expectations.json` and `renderer/*.wav`: renderer naming, curve,
  parsed note-event, and WAV-output parity targets.
