# Backend Core

- `backend/` is the primary backend.
- Module: `octabit/backend`.
- Server entrypoint: `backend/cmd/server/main.go`; run with `PORT=8000 go run ./cmd/server` from `backend/`.
- `main` loads config from env, opens workspace storage using `cfg.JobRoot`, creates `workspace.Service`, wires `httpapi.NewRouterWithOptions`, and serves via `http.Server` with graceful shutdown.
- Keep runtime synthesis behavior in `backend/internal/renderer/`.
- Go tests live with backend packages and testdata under `backend/testdata/`.
