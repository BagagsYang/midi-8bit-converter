# OctaBit Pro

Private companion to the public [octabit](https://github.com/bagags/octabit) repository.

## Structure

```
octabit-pro/
├── backend/
│   ├── cmd/pro-server/main.go    ← staged entry point, built from octabit/backend module
│   └── overlays/                 ← private backend replacements copied into the staged build
├── frontend/
│   └── overlays/src/             ← copied over octabit/frontend/src/ in the staged build
│       ├── components/            ← replacement .vue files with pro features
│       ├── composables/           ← replacement composables with pro state
│       ├── types/                 ← extended type definitions
│       ├── lib.ts                 ← extended constants and helpers
│       └── i18n/                  ← locale catalogs with pro keys
└── deploy/
    ├── deploy.sh                  ← pull both repos → stage → overlay → build → restart
    └── Caddyfile                  ← production Caddy config
```

## Deploy

```bash
PUBLIC_REPO=/home/deploy/octabit PRO_REPO=/home/deploy/octabit-pro ./deploy/deploy.sh
```
