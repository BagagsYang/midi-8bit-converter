# OctaBit Vue frontend

This is the intended production Vue 3 + TypeScript frontend for OctaBit. It is
a thin client over the Go API in `../backend/`; it does not duplicate
workspace, upload, synthesis, download, preview, theme, or language behaviour.

## Development

Start the Go backend on port 8000:

```bash
cd backend
PORT=8000 go run ./cmd/server
```

In another terminal:

```bash
cd frontend
npm ci
npm run dev
```

Open `http://127.0.0.1:5173/`.

During development, Vite proxies `/api/*` and `/static/previews/*` to
`http://127.0.0.1:8000`.

## Build

```bash
cd frontend
npm ci
npm run build
```

The production build output is `frontend/dist`. The production
model serves that directory directly with Caddy and reverse proxies `/api/*`
and `/static/previews/*` to the Go backend on `127.0.0.1:8000`. See
`../deploy/production/README.md`.
