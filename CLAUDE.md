# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common commands

- Build all: `make build`
- Build backend only: `make build-backend`
- Build frontend only: `make build-frontend` (runs `npm install` first)
- Run all backend tests: `go test ./...`
- Run tests for a single package: `go test ./internal/api/...`
- Run a single test: `go test ./internal/store/... -run TestSourceStore`
- Run frontend build/type-check: `npm --prefix web-ui run build`
- Run acceptance gate (requires backend running): `PROXYD_PASSWORD=admin ./scripts/acceptance.sh`
- Dev backend: `make dev-backend` (builds then runs with `config.example.yaml`)
- Dev frontend: `make dev-frontend` (Vite dev server with HMR at port 5173)
- Initialize DB (after build): `./build/proxyd -c config.example.yaml -init-db`
- CLI mihomo control: `./build/proxyd -c config.yaml -mihomo start|stop|restart|status`

## Architecture overview

Three-layer system: **mihomo** (Clash Meta, proxy protocol engine) ← **proxyd** (Go control plane) ← **Vue 3 UI** (management frontend).

### Backend (`cmd/`, `internal/`, `pkg/`)

- **`cmd/proxyd/main.go`** — entry point; parses CLI flags, loads config, bootstraps `internal/app`.
- **`internal/app/app.go`** — wires all components together and manages lifecycle (init → run → shutdown).
- **`pkg/config/config.go`** — loads YAML config file; `Config` struct covers Server, Database, Mihomo, Auth, Logging, Subscription, Policy, Scheduler sections.
- **`internal/types/types.go`** — all shared data types (Source, Revision, Runtime, AuditLog, ProxyInfo, etc.).

Key internal packages and their roles:

| Package | Role |
|---|---|
| `internal/store` | SQLite CRUD via 5 stores: SourceStore, SettingStore, RevisionStore, RuntimeStore, AuditStore |
| `internal/auth` | JWT generation/validation; credentials stored in settings table; **password hashing is MVP placeholder (plain text)** |
| `internal/core` | Mihomo process lifecycle (`manager.go`), Mihomo REST API client (`client.go`), binary auto-updater (`updater.go`) |
| `internal/source` | HTTP/file subscription fetching with retry logic |
| `internal/parser` | YAML parse, validate, merge multiple configs |
| `internal/renderer` | Merge sources + apply policy settings → runtime YAML |
| `internal/policy` | Generate proxy groups (select/url-test/fallback/load-balance) and rules |
| `internal/scheduler` | `robfig/cron` jobs for periodic source refresh |
| `internal/health` | Health checker exposed at `/health`; `StopPeriodicCheck()` is currently a no-op |
| `internal/api` | Gin router + handlers; WebSocket at `/api/v1/system/ws` |
| `internal/logx` | `go.uber.org/zap` with `lumberjack` log rotation |

### Database schema (`internal/store/schema.sql`)

Five tables: `sources` (type: http/file/local), `settings` (key-value, stores credentials), `revisions` (config version history), `runtime` (mihomo PID/status), `audit_logs`.

### API routes (all protected by JWT except where noted)

```
GET  /health, /ping                       # public
POST /api/v1/auth/login                   # public
GET  /api/v1/system/ws                    # public (WebSocket)

POST /api/v1/auth/logout|refresh
GET  /api/v1/auth/profile
PUT  /api/v1/auth/password

GET|PUT /api/v1/system/settings
GET     /api/v1/system/info|status|audit-logs

GET|POST        /api/v1/sources
GET|PUT|DELETE  /api/v1/sources/:id
POST            /api/v1/sources/:id/test|fetch

POST            /api/v1/config/generate|save|apply
GET             /api/v1/config
GET|DELETE      /api/v1/config/revisions/:id
POST            /api/v1/config/revisions/:id/rollback

POST /api/v1/policy/groups|rules|validate-rule|custom-group

GET  /api/v1/proxy/proxies|groups|rules|traffic|memory
POST /api/v1/proxy/proxies/:name/test
PUT  /api/v1/proxy/groups/:group
POST /api/v1/proxy/mihomo/:action          # start|stop|restart|status
```

### Config pipeline

Sources (fetch) → Parser (YAML parse) → Parser.Merge → Renderer (apply policy) → save Revision → apply to mihomo → AuditLog entry.

### Frontend (`web-ui/`)

Vue 3 + Vite + Pinia + Vue Router + Element Plus + TypeScript.

- **`src/api/request.ts`** — Axios instance; base URL `/api/v1`; JWT injected via request interceptor; 401 redirects to login.
- **`src/api/`** — one file per domain: `auth`, `system`, `source`, `config`, `policy`, `proxy`.
- **`src/stores/`** — Pinia stores: `user` (auth state), `system`, `source`, `proxy`.
- **`src/router/index.ts`** — navigation guard enforces auth; routes: `/login`, `/`, `/sources`, `/config`, `/proxies`, `/settings`.
- **`src/views/`** — Dashboard, Sources, Config, Proxies, Settings, Login.

Frontend dev server proxies API calls to backend; configure in `vite.config.ts`.

## Known issues / MVP placeholders

- `internal/auth/jwt.go` `HashPassword()` and `comparePassword()` — plain-text password storage; must replace with `bcrypt` before production.
- `internal/health/checker.go` `StopPeriodicCheck()` — empty no-op; health check ticker cannot be stopped once started.
- `proxyDelayCache` in handler has a 15-second TTL to deduplicate proxy latency test calls.

## Runtime data layout

```
data/
  db/proxyd.db          # SQLite database
  generated/            # Rendered runtime configs
  cache/                # Subscription cache
logs/                   # Application logs + acceptance reports
```

## Deployment

- Systemd unit template: `deployments/systemd/proxyd.service`
- Full deployment guide: `docs/deployment.md`
- API reference: `docs/api.md`
