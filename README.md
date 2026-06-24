<div align="center">

# 🔮 Prism

[![stars](https://img.shields.io/github/stars/qwq202/prism?style=flat-square&label=stars)](https://github.com/qwq202/prism/stargazers)
[![forks](https://img.shields.io/github/forks/qwq202/prism?style=flat-square&label=forks)](https://github.com/qwq202/prism/network/members)
[![license](https://img.shields.io/badge/license-Apache--2.0-green?style=flat-square)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black)](https://react.dev/)
[![Docker](https://img.shields.io/badge/Docker-latest-2496ED?style=flat-square&logo=docker&logoColor=white)](https://hub.docker.com/r/qunqin45/prism)

**Languages:** **English** · [简体中文](./README.zh-CN.md)

**A unified, self-hosted AI gateway & chat platform**

Aggregate ChatGPT, Claude, Gemini and other leading models in a single interface, with a
built-in OpenAI-compatible gateway, user management, subscription billing and admin
dashboard. Add your API keys to spin up a private AI site with an experience on par with the
official products — fully self-hosted, ready for team use or public services. One Docker
command to launch, no assembly required.

<br>

[![Docker](https://img.shields.io/badge/🐳_DOCKER-Deploy_in_minutes-2496ED?style=for-the-badge&logo=docker&logoColor=white)](#quick-start)
[![GitHub](https://img.shields.io/badge/📦_GITHUB-Releases-181717?style=for-the-badge&logo=github&logoColor=white)](https://github.com/qwq202/prism/releases)
[![Docs](https://img.shields.io/badge/📖_DOCS-FAQ-5733A6?style=for-the-badge)](#faq)

<br>

> 💡 **Tip:** On first launch with an empty database, an admin `root` account is created automatically. If `ROOT_INITIAL_PASSWORD` is not set, a random password is generated and written to the startup logs. Image: `qunqin45/prism:latest`. Development and releases use the `main` branch.

</div>

---

## Table of Contents

- [📸 Screenshots](#screenshots)
- [🚀 Quick Start](#quick-start)
- [🤖 AI-Powered Deployment](#ai-powered-deployment)
- [🧩 Features](#features)
- [🤖 Supported Models](#supported-models)
- [📦 Deployment](#deployment)
- [🛠️ Local Development](#local-development)
- [⚙️ Configuration](#configuration)
- [❓ FAQ](#faq)
- [🏗️ Tech Stack](#tech-stack)

---

## Screenshots

<details>
<summary><b>👉 Click to expand screenshots</b></summary>

**Web Search** — Native search for leading models; Tavily-powered retrieval for the rest.

![Web Search](docs/image/image-20260508092832242.png)

**Universal Web Search** — Non-native-search models use the [Tavily](https://tavily.com/) API, with smart keyword extraction via a task model.

![Universal Web Search](docs/image/image-20260508093619270.png)

**Persistent Memory** — Saves user preferences across sessions so the model keeps learning about each user.

![Persistent Memory](docs/image/image-20260508093038845.png)

**Model Market Performance** — Shows TPS, latency, success rate and availability trends to guide model selection.

![Model Market Performance](docs/image/image-20260526145149103.png)

![Model Market Details](docs/image/image-20260526145230355.png)

**Dual-Window Quota** — Short-cycle (5h) and weekly limits are computed and reset independently.

![Dual-Window Quota](docs/image/image-20260508093148510.png)

**Admin Dashboard** — Users, channels, subscriptions, model market and announcements in one place.

![Admin Dashboard](docs/image/image-20260508093251949.png)

![Admin Dashboard Details](docs/image/image-20260508093321382.png)

**Usage Tracking** — Request logs, token consumption and cost breakdowns at a glance.

![Usage Tracking](docs/image/image-20260508093442056.png)

**Object Storage** — Compatible with S3, Cloudflare R2, MinIO and more.

![Object Storage](docs/image/image-20260508093726100.png)

</details>

---

## Quick Start

> [!NOTE]
> Running Prism requires [Docker](https://docs.docker.com/get-docker/) (with Compose). On first launch with an empty database, an admin `root` account is created; if `ROOT_INITIAL_PASSWORD` is unset, a random password is written to the startup logs.

```shell
git clone --depth=1 --branch=main --single-branch https://github.com/qwq202/prism.git
cd prism
docker compose up -d
```

After launch, open `http://localhost:8000`. To customize the database password, `SECRET`, initial `root` password and more, copy `.env.example` to `.env` and edit it (`.env` is git-ignored — never commit real secrets).

```shell
docker compose ps                 # check container status
curl http://localhost:8000/health # health check; status: ok means healthy
```

**Upgrade:**

```shell
docker compose down && docker compose pull && docker compose up -d
```

> [!TIP]
> If Watchtower is enabled, images update automatically and manual upgrade can be skipped.

---

## AI-Powered Deployment

Prefer not to do it manually? Copy the prompt below into Claude, Cursor, ChatGPT or another AI assistant and let it deploy for you (use the copy button at the top-right of the code block).

```text
Deploy Prism — a self-hosted AI gateway & chat platform — for me.

Project info:
- Repo: https://github.com/qwq202/prism
- Docker image: qunqin45/prism:latest
- Access URL: http://localhost:8000

Steps (report the result after each one):
1. Ensure Docker (with Compose) is installed; if not, install via the official script: curl -fsSL https://get.docker.com | sh
2. Clone the repo: git clone --depth=1 --branch=main --single-branch https://github.com/qwq202/prism.git
3. Start the stack: cd prism && docker compose up -d (includes MySQL, Redis and Prism)
4. Health check: curl http://localhost:8000/health — expect status: ok

When done:
- The admin username is root; the password comes from the ROOT_INITIAL_PASSWORD env var, or a random one printed in docker compose logs if unset
- Tell me the access URL and the root password

Resolve any issues (port conflicts, container failures, etc.) before continuing.
```

> [!NOTE]
> Requires an AI assistant with terminal access (e.g. Cursor, Claude Code, or other terminal-capable agents). Chat-only AIs will guide you step by step but cannot run commands directly.

---

## Features

### Chat & Multimodal

- 🤖 **Multi-model chat** — Aggregate OpenAI, Claude, Gemini, DeepSeek, Grok and more in a single interface, with Markdown / LaTeX / Mermaid rendering and syntax highlighting.
- 💭 **Reasoning display** — Live Reasoning / Thinking content (OpenAI, DeepSeek, xAI, MiMo, etc.).
- 🧠 **Persistent memory** — Saves user preferences across sessions so the model keeps understanding your context without repetition.
- 🎨 **AI drawing workspace** — Native image generation via DALL·E and Gemini; turn text prompts into images with multiple parallel workspaces (`/drawing` page).
- 📎 **File & image handling** — Parse PDF / Office / images, with paste-to-upload, pre-send preview and auto-conversion of long text to attachments.
- 🔧 **Tool calls & web fetch** — Native Function Calling / Tool Use, plus a built-in Fetch Webpage content extractor.
- 🌐 **Web search** — Picks the optimal path per model: leading models use the provider's native Web Search, while others are augmented via the [Tavily](https://tavily.com/) API — no need to self-host SearXNG or similar layers (see table below).

#### Web Search Routing

| Type | Applicable models | Description |
|------|-------------------|-------------|
| **Native search** | OpenAI Responses (GPT-5 series), Gemini, xAI Grok | Calls the provider's native Web Search / Google Search / X Search directly |
| **Tavily augmented** | Other models with web search enabled | Fetches real-time results via the [Tavily API](https://tavily.com/), optionally with keyword extraction by a task model |

Configure the Tavily API key, search depth, topic and result count under **System Settings → Web Search**.

### Gateway & Channels

- 📡 **OpenAI-compatible API** — A unified `/v1/chat/completions` gateway that works with any OpenAI client.
- ⚖️ **Multi-channel load balancing** — Scheduling by priority, weight and user group, with automatic failover and retry.
- 🔀 **Model mapping & redirection** — Transparently remap a requested model name to an upstream model; the `!` prefix hides the original.
- 💾 **Request caching** — Identical requests hit the cache at no charge, cutting repeat-call costs.
- 📊 **Model market** — Real call-data metrics (TPS, latency, success rate, availability) to guide model selection.
- 🔄 **Upstream sync** — One-click sync of channels, model lists and pricing.

### Billing & Operations

- 🎯 **Flexible billing** — Per-request / per-token / free modes, with a minimum-points check.
- 📅 **Subscriptions (dual-window quota)** — Short-cycle (5h) and weekly quotas computed and reset independently, with per-plan quota pools.
- 🎁 **Gift codes / redemption codes** — Bulk generation; single-user-limited gift codes vs. multi-user redemption codes.
- 📝 **Usage tracking** — Full request logs, token consumption and cost breakdowns.

### Admin Dashboard

- 📈 **Dashboard & announcements** — Real-time operations data and announcement pushes.
- 👥 **User management** — Bulk operations, direct creation, grouping and quota control.
- 💼 **Plans & pricing** — Subscription plans, price templates, channel and model-market configuration.
- 🎨 **Site customization** — Name / logo, SMTP email, attachments and object storage (S3 / Cloudflare R2 / MinIO).

### Experience

- 🌍 **Multilingual** — 中文 / English / 日本語 / Русский.
- 🌗 **Themes** — Light / dark mode.
- 📱 **PWA** — "Add to Home Screen" on mobile for a near-native app experience.
- 🔐 **Secure auth** — JWT signing, optional Passkey / WebAuthn passwordless login; self-hosted data stays under your control.
- 🔄 **Cross-device sync & sharing** — Conversations sync across devices, with link / image sharing.

---

## Supported Models

| Provider | Capabilities |
|----------|--------------|
| **OpenAI & Azure** | Vision, Function Calling, GPT-5 series, Reasoning Summaries, native Web Search, image generation (DALL·E / gpt-image-1) |
| **Anthropic Claude** | Vision, Function Calling |
| **Google Gemini** | Vision, native Google Search / URL Context, native image generation |
| **DeepSeek** | V4, Thinking control, Prompt Cache stats |
| **xAI Grok** | Responses API, native Web Search / X Search, Writable Memory, Reasoning |
| **Xiaomi MiMo** | Thinking Toggle, Token Plan China |
| **MiniMax** | Token Plan CN |
| **GLM** | Coding Plan CN |
| **LocalAI / Ollama** | OpenAI-compatible format (local models) |

> 💡 Any provider that conforms to the OpenAI API format can be integrated. For locally deployed models, use Ollama or LocalAI.

---

## Deployment

> [!IMPORTANT]
> First launch (empty user table) auto-creates an admin account, username `root`. Password source: the `ROOT_INITIAL_PASSWORD` env var or `root.initial_password` in `config.yaml` (6–36 chars, either one; env var takes precedence); if neither is set, a 24-char random password is generated and printed to the startup logs (view with `docker compose logs`). Full details in the [FAQ](#faq).

### Option 1: Docker Compose (recommended)

Access at `http://localhost:8000`

```shell
git clone --depth=1 --branch=main --single-branch https://github.com/qwq202/prism.git
cd prism
docker compose up -d
```

**Data & config directories** (auto-created on first launch):

| Path | Description |
|------|-------------|
| `./db` | MySQL data |
| `./redis` | Redis data |
| `./config` | Config files (auto-generates `config.yaml` and a random `secret`) |

> [!NOTE]
> After the MySQL container initializes for the first time, credentials are persisted in `./db`. If a data directory already exists, changing the MySQL password in `.env` will not migrate it automatically — update the in-database password or reinitialize.

<details>
<summary><b>Use the stable image (MySQL 5.7 / Redis 8.2)</b></summary>

```shell
docker compose -f docker-compose.stable.yaml up -d
```

For legacy environments with database-version compatibility requirements.

</details>

### Option 2: Single-container Docker (external MySQL / Redis)

Already have your own database? Use this. Access at `http://localhost:8094`

```shell
docker run -d --name prism \
  --network host \
  -v ~/config:/config \
  -v ~/logs:/logs \
  -v ~/storage:/storage \
  -e MYSQL_HOST=localhost \
  -e MYSQL_PORT=3306 \
  -e MYSQL_DB=prism \
  -e MYSQL_USER=root \
  -e MYSQL_PASSWORD=your_mysql_password \
  -e REDIS_HOST=localhost \
  -e REDIS_PORT=6379 \
  -e SECRET=replace_with_a_random_32_byte_string \
  -e ROOT_INITIAL_PASSWORD=replace_with_a_strong_initial_password \
  -e SERVE_STATIC=true \
  qunqin45/prism:latest
```

| Env var | Description |
|---------|-------------|
| `SECRET` | JWT signing key, at least 32 random characters |
| `ROOT_INITIAL_PASSWORD` | `root` password for first empty-DB launch (6–36 chars); if unset, read the random one from logs |
| `SERVE_STATIC` | Whether the backend serves static files (default `true`) |

```shell
curl http://localhost:8094/health   # health check
docker stop prism && docker rm prism && docker pull qunqin45/prism:latest  # update image
```

### Option 3: Decoupled frontend/backend

- **Frontend**: host statically via Nginx / Vercel etc., set `VITE_BACKEND_ENDPOINT` at build time (e.g. `https://api.example.com`)
- **Backend**: set `SERVE_STATIC=false` and deploy the API on its own domain
- ⚠️ Prism itself does not support full-stack Vercel deployment; only the frontend can go to Vercel

### 🍓 ARM architecture (Raspberry Pi / Apple Silicon)

The public image `qunqin45/prism:latest` is `linux/amd64`. On ARM machines, build from source locally or use BuildX to produce a `linux/arm64` image.

---

## Local Development

**Prerequisites**: Go 1.25+, Node.js (pnpm), MySQL 8, Redis 7

```shell
# Backend
go build .                    # compile
go test ./...                 # tests

# Frontend
cd app && pnpm install        # install deps
cd app && pnpm dev            # dev server
cd app && pnpm lint           # ESLint
cd app && pnpm build          # production build → app/dist

# Full stack (database + Redis + backend)
docker compose up -d
```

The backend entry point is `main.go`; frontend source lives in `app/src/`. See [`config.example.yaml`](config.example.yaml) for configuration.

---

## Configuration

The most commonly used options (full reference in [`config.example.yaml`](config.example.yaml)):

| Key | Description |
|-----|-------------|
| `secret` | JWT signing key; auto-generated on first launch |
| `root.initial_password` | Initial admin password; equivalent to the `ROOT_INITIAL_PASSWORD` env var |
| `serve_static` | Keep `true` when serving frontend and backend from the same process |
| `system.general.backend` | Backend API URL; defaults to same-origin `/api`, set a full URL for decoupled deployment |
| `ALLOW_ORIGINS` | Strict CORS allowlist, comma-separated domains (no protocol prefix) |
| `system.search.api_key` | Tavily API key for non-native-search models |
| `system.search.depth` | Tavily search depth: `basic` / `advanced` / `fast` / `ultra-fast` |
| `system.task.model` | Model used for search keyword extraction (optional) |

---

## FAQ

<details>
<summary><b>Chat hangs / no response</b></summary>

Chat uses WebSocket (API relay uses plain HTTP and doesn't need WebSocket). Make sure your reverse proxy (Nginx / Apache), CDN or port forwarding has WebSocket support enabled.

</details>

<details>
<summary><b>🔑 Admin account & initial password</b></summary>

On first launch, when the **user table is empty**, an admin account is created automatically:

| Field | Value |
|-------|-------|
| Username | `root` (fixed) |
| Email | `root@example.com` |
| Role | Administrator |

**Initial password sources** (either one; the env var takes precedence over the config file):

| Method | Key | Rule |
|--------|-----|------|
| Env var | `ROOT_INITIAL_PASSWORD` | 6–36 chars |
| Config file | `root.initial_password` in `config.yaml` | 6–36 chars |

- If **neither is set** or the value is **out of range**, a **24-character random password** is generated and printed in plaintext to the startup logs:
  ```
  [service] no user found, creating root user with generated password (username: root, password: <24-char-random>); save it now ...
  ```
  Retrieve it with `docker compose logs`.
- The password is **generated only once**, when the user table is empty. Changing the env var or config file afterwards will not recreate or update the `root` password.

**View / change the password:**

1. Didn't set one on first launch → read the random password from `docker compose logs`
2. Already signed in → Dashboard → System Settings → Change Root Password; or edit under User Management
3. Reset when locked out:
   - Compose: `docker compose exec chatnio prism root <new-password>`
   - Single container: `docker exec prism prism root <new-password>`
   - Binary: `./prism root <new-password>`

</details>

<details>
<summary><b>External dependencies</b></summary>

| Service | Purpose | Required? |
|---------|---------|-----------|
| **MySQL** | Persistent data: users, conversations, config | ✅ Required |
| **Redis** | Auth, rate limiting, subscription quotas, verification codes | ✅ Required |

</details>

<details>
<summary><b>Billing & subscriptions</b></summary>

- **Flexible billing (points)**: general pay-as-you-go, default 10 points = 1 CNY, adjustable in billing-rule templates
- **Subscriptions**: fixed price + windowed quota; charges use points (e.g. a 32 CNY plan requires ≥ 320 points)
- Four subscription tiers: Regular (0), Basic (1), Standard (2), Pro (3), mapped to channel user groups

</details>

<details>
<summary><b>Gift codes vs. redemption codes</b></summary>

| Type | Trait | Use case |
|------|-------|----------|
| **Gift code** | Each user can redeem a given type only once | Promotions, giveaways |
| **Redemption code** | A given type can be redeemed by many users | Card sales, bulk purchases |

</details>

<details>
<summary><b>Error: user quota is not enough</b></summary>

This is the minimum-request-points check:

- Free models: no limit
- Per-request billing: min points = per-request cost
- Per-token billing: min points = 1K input price + 1K output price

Fix: top up points, or grant the user more quota.

</details>

<details>
<summary><b>Model mapping</b></summary>

Inside a channel, use the format `[from]>[to]`, one per line. `from` is the model the user requests, `to` is the actual upstream model.

```
gpt-4-all>gpt-4          # map gpt-4-all to gpt-4
!gpt-4-all>gpt-4         # the ! prefix hides gpt-4 on this channel, exposing only gpt-4-all
```

</details>

<details>
<summary><b>Payment integration</b></summary>

Configure a purchase link (card-sale URL) in System Settings; redemption codes are generated in bulk from the dashboard.

</details>

---

## Tech Stack

| Layer | Stack |
|-------|-------|
| **Frontend** | React 19, Redux Toolkit, Radix UI, Tailwind CSS, Vite |
| **Backend** | Go 1.25, Gin, MySQL, Redis |
| **Deployment** | Docker, Docker Compose, PWA, WebSocket |

---

## 🤝 Support

- Bug reports & feature requests: [open an Issue](https://github.com/qwq202/prism/issues)
- Release notes & updates: [GitHub Releases](https://github.com/qwq202/prism/releases)

## 🙏 Acknowledgements

This project is a secondary development based on [coai](https://github.com/coaidev/coai). Sincere thanks to the original authors for the foundational work.

## Star History

<a href="https://star-history.com/#qwq202/prism&Date">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=qwq202/prism&type=Date&theme=dark" />
    <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=qwq202/prism&type=Date" />
    <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=qwq202/prism&type=Date" />
  </picture>
</a>

<div align="center">

<sub>Made with ❤️ · Apache-2.0 License</sub>

</div>
