# OctaBit Pro

Private companion to the public [octabit](https://github.com/bagags/octabit) repository.

## Structure

```
octabit-pro/
├── backend/
│   └── cmd/pro-server/main.go   ← injected into octabit/backend at deploy time, built from octabit/backend module
├── frontend/
│   └── overlays/src/             ← copied over octabit/frontend/src/ at deploy time
│       ├── components/            ← replacement .vue files with pro features
│       ├── composables/           ← replacement composables with pro state
│       ├── types/                 ← extended type definitions
│       ├── lib.ts                 ← extended constants and helpers
│       └── i18n/                  ← locale catalogs with pro keys
└── deploy/
    ├── deploy.sh                  ← pull both repos → inject → build → restart
    └── Caddyfile                  ← production Caddy config
```

## Deploy

```bash
PUBLIC_REPO=/home/deploy/octabit PRO_REPO=/home/deploy/octabit-pro ./deploy/deploy.sh
```
