# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

Prism is an AI gateway and chat platform. The backend is written in Go and exposes an OpenAI-compatible API; the frontend is a React SPA. Backend and frontend are usually served from the same Go process (`serve_static: true`), but can also be deployed separately.

- Backend module: `chat` (Go 1.25, Gin, MySQL, Redis)
- Frontend: React 19 + TypeScript + Vite + Tailwind CSS + Redux Toolkit (in `app/`)
- Deployment: Docker Compose or single container; image `qunqin45/prism:latest`

## Repository structure

- `main.go` — backend entry point; wires modules, registers routes, and starts the Gin server.
- `app/` — React frontend source (`app/src/`) and Vite config.
- `adapter/` — vendor-specific LLM adapters (OpenAI, Claude, Gemini, DeepSeek, xAI, etc.).
- `channel/` — channel manager, load balancing, model mapping, pricing rules, plans, system config.
- `manager/` — chat/completion handlers, WebSocket API, image/video generation, billing/usage hooks, conversation and memory subpackages.
- `auth/` — authentication, quotas, subscriptions, invitations, redeem codes, passkeys.
- `admin/` — admin dashboard endpoints and analytics.
- `billing/` — usage records and model performance metrics.
- `addition/` — auxiliary features: article generation, web fetch, title auto-generation.
- `middleware/` — auth, CORS, throttling, client IP utilities.
- `connection/` — MySQL/Redis connections, migrations, background workers.
- `utils/` — shared helpers: config I/O, buffer/usage counting, storage, tokenizer, websockets.
- `globals/` — shared types, constants, capabilities, SQL helpers.
- `cli/` — command-line subcommands invoked via the compiled binary.

## Common commands

### Backend

```bash
# Build the backend binary
go build .

# Run all Go tests (currently known to fail in utils/image.go; treat as a known issue)
go test ./...

# Run a specific package test
go test ./channel
go test -run TestName ./utils

# Reset the root password without restarting
./prism root <new-password>

# Generate an invitation code
./prism invite
```

### Frontend

```bash
cd app

# Install dependencies and start the Vite dev server
pnpm install && pnpm dev

# Lint with ESLint
pnpm lint

# Format TypeScript/React files
pnpm prettier

# Production build (outputs to app/dist)
pnpm build

# Faster build without type checking
pnpm fast-build
```

### Full stack

```bash
# Start MySQL, Redis, and the app via Docker Compose
docker compose up -d

# Use the stable prebuilt image instead of building locally
docker compose -f docker-compose.stable.yaml up -d

# Health check
curl http://localhost:8000/health
```

## Architecture notes

### Request flow

1. The client connects to `/api/chat` via WebSocket (`manager.ChatAPI`).
2. `manager/chat.go` loads the conversation, checks subscription/quota, builds `adaptercommon.ChatProps`, and calls `channel.NewChatRequestWithCache`.
3. `channel` selects an upstream channel (priority/weight/group aware) and routes to the vendor adapter in `adapter/`.
4. The adapter streams chunks back through `utils.Buffer`, which counts input/output tokens and computes quota in real time.
5. `manager/chat.go` enforces the realtime quota limiter, persists the assistant message, and optionally triggers tool rounds (memory, web search, fetch webpage).

### Channel and adapter wiring

- `channel/manager.go` maintains `ConduitInstance`, `ChargeInstance`, `SystemInstance`, and `PlanInstance` globals initialized by `channel.InitManager()` in `main()`.
- `channel.Channel` defines a single upstream (endpoint, secret, models, mapper, group, proxy, priority, weight, retry). `Load()` compiles model mapping rules like `from>to` and `!from>to` into `Reflect`/`ExcludeModels`/`HitModels`.
- `adapter/adapter.go` maps each `channel.Type` to a factory function; adapters implement `CreateStreamChatRequest` and optionally video requests.

### Configuration

- Runtime config is loaded from `config/config.yaml` by `utils.ReadConf()`.
- On first start, `config.example.yaml` is copied to `config/config.yaml` and a random `secret` is generated.
- Config changes made through admin endpoints are persisted via `utils.SaveConfig()`, which rewrites `config.yaml` atomically with a `.bak` backup.
- Environment variables override config keys when uppercase with `_` instead of `.`, e.g. `MYSQL_PASSWORD`, `SECRET`, `ROOT_INITIAL_PASSWORD`, `SERVE_STATIC`.

### Routing layout

- `registerApiRouter` in `main.go` mounts all modules under `/api` when `serve_static` is true, or at root when false.
- `utils.RegisterStaticRoute` serves `app/dist` and rewrites `/v1/*` and `/attachments/*` to `/api/*`.
- `auth.Register`, `admin.Register`, `manager.Register`, etc. define their own route groups.

### State and storage

- MySQL stores users, conversations, messages, billing records, attachments, invitations/redeems, and market data.
- Redis is used for auth sessions/rate limits, subscription quota windows, verification codes, and request caching.
- Frontend state is managed with Redux Toolkit; chat history and the current conversation are cached to `localforage`.

## Development conventions

### Code style

- Backend: `gofmt`, lowercase package names, files named `router.go`, `controller.go`, `types.go` per package.
- Frontend: 2-space indentation per `app/.prettierrc.json`; React components use PascalCase (`ChatInterface.tsx`); shared UI components in `app/src/components/ui/` use lowercase kebab-case (`alert-dialog.tsx`).

### Commits and branches

- Use Conventional Commits (`feat:`, `fix:`, `chore:`).
- `main` is the only active development and release branch; commit fixes and features directly to `main`.
- Make a separate commit for each independent round of changes rather than batching unrelated work.
- Releases are semantic-version tags (`v1.0.0`) cut from `main`; Docker image `qunqin45/prism:latest` tracks `main`.

### Validation checklist

Before considering a change complete, run:

```bash
go build .
go test ./...
cd app && pnpm lint
cd app && pnpm build
```

`go test ./...` currently fails on `utils/image.go` — this is a known issue, not a regression introduced by most changes.

## Important configuration values

- `secret` — JWT signing key; must be at least 32 random bytes. A weak value logs a warning and delays startup.
- `root.initial_password` / `ROOT_INITIAL_PASSWORD` — initial `root` password on empty database; if unset, a random password is printed in the logs.
- `serve_static` / `SERVE_STATIC` — `true` when the Go process serves the frontend; `false` for API-only deployments.
- `system.general.backend` — backend base URL used in generated links; keep `/api` or empty for same-domain.
- `system.search.api_key` — Tavily key for non-native web search.
- `system.task.model` — model used to extract keywords for Tavily search.
