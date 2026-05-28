# Vue Production Deployment

This is the intended non-Docker production path for `octabit.cc`.

- Caddy serves the built Vue 3 frontend from `frontend/dist`.
- Caddy reverse proxies `/api/*`, `/static/previews/*`, and `/synthesise*` to
  the Go backend on `127.0.0.1:8000`.
- The Go backend remains private and owns workspace, upload, synthesis,
  download, preview asset, and legacy compatibility routes.
- The legacy Flask stack remains in the repository for fallback reference and
  fixture regeneration, not as the normal production path.

The Docker files in `deploy/web-flask/` are an alternate legacy Flask fallback
path. Do not introduce Docker into the current production cutover unless the
production plan changes.

## One-Time Server Shape

Use a repository checkout such as `/home/deploy/octabit`, the `octabit-web`
systemd service for the Go backend, and Caddy as the public server.

The Go backend should stay private:

```bash
cd /home/deploy/octabit/backend
go build -o octabit-server ./cmd/server
PORT=8000 WEB_SYNTHESISE_JOB_ROOT=/var/lib/octabit ./octabit-server
```

Install Node.js and npm from the server's normal package source before the Vue
cutover. The Vue dependency install should use the lockfile:

```bash
cd /home/deploy/octabit/frontend
npm ci
npm run build
```

## Caddy Routing

Use `Caddyfile.vue-production` as the production model:

```caddyfile
octabit.cc {
	encode zstd gzip

	handle /api/* {
		reverse_proxy 127.0.0.1:8000
	}

	handle /static/previews/* {
		reverse_proxy 127.0.0.1:8000
	}

	handle /synthesise* {
		reverse_proxy 127.0.0.1:8000
	}

	handle {
		root * /home/deploy/octabit/frontend/dist
		try_files {path} /index.html
		file_server
	}
}
```

This keeps the Vue app as the public frontend while preserving the API,
preview audio route, and legacy synthesis routes. The `try_files` fallback is
for Vue/Vite browser routes and should not catch API requests because those are
handled first.

## Deployment Flow

From the production VM:

```bash
cd /home/deploy/octabit
git fetch --prune origin
git checkout main
git pull --ff-only origin main
cd backend
go build -o octabit-server ./cmd/server
cd /home/deploy/octabit
cd frontend
npm ci
npm run build
cd /home/deploy/octabit
sudo systemctl restart octabit-web
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

The default helper script target is `main`:

```bash
deploy/production/deploy-vue-production.sh
```

Set `APP_DIR=/path/to/octabit` if the production checkout uses a different
path.

The helper prints the commit it deployed, derives expected UI locales from
`frontend/src/i18n/*.json`, verifies the local Vue bundle contains every
`toolbar.language_option.<locale>` marker, then fetches `PUBLIC_URL` after
Caddy reload and verifies the public JavaScript bundle too. `PUBLIC_URL`
defaults to `https://octabit.cc`; set `PUBLIC_URL=` to skip the public check for
private dry runs.

If Caddy serves a separate static root, set `WEB_ROOT` so the helper publishes
`frontend/dist/` there before reloading Caddy. For example, the current VM can
use:

```bash
WEB_ROOT=/var/www/octabit deploy/production/deploy-vue-production.sh
```

`WEB_ROOT` must be a dedicated static web root. The helper uses
`rsync --delete`, so do not point it at a directory that contains unrelated
files.

## Smoke Checks

Run local checks on the VM:

```bash
curl -fsS http://127.0.0.1:8000/api/health
test -f /home/deploy/octabit/frontend/dist/index.html
```

Run public checks after Caddy reload:

```bash
curl -fsS https://octabit.cc/
curl -fsS https://octabit.cc/api/health
curl -fsSI https://octabit.cc/static/previews/pulse_50.wav
```

Then use a browser to upload a small MIDI file, verify the workspace survives a
refresh, run synthesis, download the WAV, change theme/language, and clear the
queued/converted files.

## Rollback

If a new Vue build fails after deployment, keep the Go backend running and roll
the checkout back to the previous known-good revision, then rebuild
`frontend/dist` and reload Caddy:

```bash
cd /home/deploy/octabit
git checkout <previous-good-revision>
cd frontend
npm ci
npm run build
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

If the Go backend deployment fails, restart the previous systemd unit or binary
revision before reloading Caddy:

```bash
sudo systemctl restart octabit-web
curl -fsS http://127.0.0.1:8000/api/health
```
