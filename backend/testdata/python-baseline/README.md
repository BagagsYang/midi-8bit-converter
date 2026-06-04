# Python Baseline Fixtures

These fixtures freeze the Python renderer behavior for Go backend conformance
testing. They are parity inputs for Go tests, not a runtime dependency.

Current fixture groups:

- `api/implemented_routes.json`: API transcript for routes implemented by the
  Go backend.
- `api/workspace_flow.json`, `api/error_responses.json`, and
  `api/legacy_jobs.json`: API parity targets for workspace/job implementation.
- `renderer/expectations.json` and `renderer/*.wav`: renderer naming, curve,
  parsed note-event, and WAV-output parity targets.
