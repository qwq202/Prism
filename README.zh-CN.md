<div align="center">

# 🔮 Prism

[![stars](https://img.shields.io/github/stars/qwq202/prism?style=flat-square&label=stars)](https://github.com/qwq202/prism/stargazers)
[![forks](https://img.shields.io/github/forks/qwq202/prism?style=flat-square&label=forks)](https://github.com/qwq202/prism/network/members)
[![license](https://img.shields.io/badge/license-Apache--2.0-green?style=flat-square)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black)](https://react.dev/)
[![Docker](https://img.shields.io/badge/Docker-latest-2496ED?style=flat-square&logo=docker&logoColor=white)](https://hub.docker.com/r/qunqin45/prism)

**Languages:** [English](./README.md) · **简体中文**

**统一接入、自主可控的 AI 网关与对话平台**

一个界面聚合 ChatGPT、Claude、Gemini 等主流模型，内置 OpenAI 标准网关、用户体系、
订阅计费与管理后台。填入密钥即可拥有与官方体验相当的私人 AI 站点，支持私有部署、
团队共享与对外服务。Docker 一键启动，无需从零搭建。

<br>

[![Docker](https://img.shields.io/badge/🐳_DOCKER-一键部署-2496ED?style=for-the-badge&logo=docker&logoColor=white)](#快速开始)
[![GitHub](https://img.shields.io/badge/📦_GITHUB-Releases-181717?style=for-the-badge&logo=github&logoColor=white)](https://github.com/qwq202/prism/releases)
[![Docs](https://img.shields.io/badge/📖_文档-常见问题-5733A6?style=for-the-badge)](#常见问题)

<br>

> 💡 **提示：** 首次空库启动会自动创建管理员 `root`。未设置 `ROOT_INITIAL_PASSWORD` 时，系统生成随机密码并写入启动日志。镜像地址为 `qunqin45/prism:latest`，开发与发布均使用 `main` 分支。

</div>

---

## 目录

- [📸 界面预览](#界面预览)
- [🚀 快速开始](#快速开始)
- [🤖 AI 一键部署](#ai-一键部署)
- [🧩 功能特性](#功能特性)
- [🤖 支持模型](#支持模型)
- [📦 部署方式](#部署方式)
- [🛠️ 本地开发](#本地开发)
- [⚙️ 配置说明](#配置说明)
- [❓ 常见问题](#常见问题)
- [🏗️ 技术栈](#技术栈)

---

## 界面预览

<details>
<summary><b>👉 点击展开截图（多图预警）</b></summary>

**联网搜索** — 主流模型走供应商原生搜索；其余模型通过 Tavily 自动检索增强。

![联网搜索](docs/image/image-20260508092832242.png)

**通用联网搜索** — 非原生搜索模型使用 [Tavily](https://tavily.com/) API，并可用任务模型智能提取关键词。

![通用联网搜索](docs/image/image-20260508093619270.png)

**持久记忆** — 跨会话保存用户偏好，让模型持续了解每位用户。

![持久记忆](docs/image/image-20260508093038845.png)

**模型市场表现** — 展示 TPS、延迟、成功率与可用率趋势，辅助选型。

![模型市场表现](docs/image/image-20260526145149103.png)

![模型市场详情](docs/image/image-20260526145230355.png)

**双窗口配额** — 短周期（5 小时）与每周额度独立计算、独立重置。

![双窗口配额](docs/image/image-20260508093148510.png)

**管理后台** — 用户、渠道、订阅、模型市场、公告等一站式运营。

![管理后台](docs/image/image-20260508093251949.png)

![管理后台详情](docs/image/image-20260508093321382.png)

**用量追踪** — 请求日志、Token 消耗与费用明细一目了然。

![用量追踪](docs/image/image-20260508093442056.png)

**对象存储** — 兼容 S3、Cloudflare R2、MinIO 等协议。

![对象存储](docs/image/image-20260508093726100.png)

</details>

---

## 快速开始

> [!NOTE]
> 运行 Prism 需要 [Docker](https://docs.docker.com/get-docker/)（含 Compose）。首次空库启动会自动创建管理员 `root`；若未设置 `ROOT_INITIAL_PASSWORD`，随机密码将输出到启动日志。

```shell
git clone --depth=1 --branch=main --single-branch https://github.com/qwq202/prism.git
cd prism
docker compose up -d
```

启动后访问 `http://localhost:8000`。如需自定义数据库密码、`SECRET`、`root` 初始密码等，将 `.env.example` 复制为 `.env` 后修改（`.env` 已被 git 忽略，勿提交真实密钥）。

```shell
docker compose ps                 # 检查容器状态
curl http://localhost:8000/health # 健康检查，返回 status: ok 即正常
```

**升级版本：**

```shell
docker compose down && docker compose pull && docker compose up -d
```

> [!TIP]
> 已启用 Watchtower 时镜像会自动更新，可跳过手动升级。

---

## AI 一键部署

不想手动操作？把下方提示词整段复制给 Claude、Cursor、ChatGPT 等 AI 助手，让它自动完成部署（点击代码块右上角的复制按钮即可）。

```text
请帮我部署 Prism（一个开箱即用的 AI 网关与对话平台）。

项目信息：
- 仓库：https://github.com/qwq202/prism
- Docker 镜像：qunqin45/prism:latest
- 访问地址：http://localhost:8000

请按以下步骤操作，每步执行后向我报告结果：
1. 确认已安装 Docker（含 Compose）；未安装则用官方脚本安装：curl -fsSL https://get.docker.com | sh
2. 克隆仓库：git clone --depth=1 --branch=main --single-branch https://github.com/qwq202/prism.git
3. 启动服务：cd prism && docker compose up -d（含 MySQL、Redis、Prism）
4. 健康检查：curl http://localhost:8000/health，返回 status: ok 即正常

部署完成后：
- 管理员用户名为 root；密码由 ROOT_INITIAL_PASSWORD 环境变量决定，未设置时从 docker compose logs 中读取随机密码
- 把访问地址和 root 密码告诉我

遇到端口占用、容器启动失败等问题，先排查解决再继续。
```

> [!NOTE]
> 需使用具备终端执行能力的 AI 助手（如 Cursor、Claude Code 等终端 Agent）。纯对话型 AI 会给出逐步指引，但无法直接执行命令。

---

## 功能特性

### 对话与多模态

- 🤖 **多模型对话** — 单一界面聚合 OpenAI、Claude、Gemini、DeepSeek、Grok 等主流模型，支持 Markdown / LaTeX / Mermaid 渲染与代码高亮。
- 💭 **思维过程展示** — 实时呈现 Reasoning / Thinking 内容（OpenAI、DeepSeek、xAI、MiMo 等）。
- 🧠 **持久记忆** — 跨会话保存用户偏好，模型持续理解上下文，无需重复说明。
- 🎨 **AI 绘画工作台** — 基于 DALL·E、Gemini 原生图片生成能力，文字描述直接出图，支持多工作区并行创作（`/drawing` 页面）。
- 📎 **文件与图片处理** — 解析 PDF / Office / 图片，支持粘贴上传、发送前预览与长文本自动转附件。
- 🔧 **工具调用与网页抓取** — 原生 Function Calling / Tool Use，内置 Fetch Webpage 内容提取。
- 🌐 **联网搜索** — 按模型自动选择最优路径：主流模型走供应商原生 Web Search，其余经 [Tavily](https://tavily.com/) API 检索增强，无需自建 SearXNG 等中间层（详见下表）。

#### 联网搜索路径

| 类型 | 适用模型 | 说明 |
|------|----------|------|
| **原生搜索** | OpenAI Responses（GPT-5 系列）、Gemini、xAI Grok | 直接调用供应商 Web Search / Google Search / X Search |
| **Tavily 增强** | 其他已开启联网的模型 | 经 [Tavily API](https://tavily.com/) 取实时结果，可配合任务模型提取关键词 |

在 **系统设置 → 联网搜索** 中配置 Tavily API Key、搜索深度、主题与结果数量。

### 网关与渠道

- 📡 **OpenAI 标准接口** — 统一的 `/v1/chat/completions` 网关，兼容任意 OpenAI 客户端接入。
- ⚖️ **多渠道负载均衡** — 按优先级、权重、用户分组调度，失败自动重试，保障可用性。
- 🔀 **模型映射与重定向** — 将用户请求的模型名透明映射到实际上游模型，支持 `!` 前缀隐藏原模型。
- 💾 **请求缓存** — 相同入参命中缓存时不计费，降低重复调用成本。
- 📊 **模型市场** — 基于真实调用数据展示 TPS、延迟、成功率、可用率，辅助模型选型。
- 🔄 **上游配置同步** — 一键同步渠道、模型列表与价格。

### 计费与运营

- 🎯 **弹性计费** — 按次 / 按 Token / 不计费三种模式，支持最小请求点数校验。
- 📅 **订阅制（双窗口额度）** — 短周期（5 小时）与每周额度独立计算、独立重置，套餐级额度池。
- 🎁 **礼品码 / 兑换码** — 批量生成，区分单用户限兑（礼品码）与多用户可兑（兑换码）。
- 📝 **用量追踪** — 完整请求日志、Token 消耗与费用明细。

### 管理后台

- 📈 **仪表盘与公告** — 实时运营数据，公告推送。
- 👥 **用户管理** — 批量操作、直接创建、分组与额度管控。
- 💼 **套餐与定价** — 订阅套餐、价格模板、渠道与模型市场配置。
- 🎨 **站点定制** — 名称 / Logo、SMTP 邮件、附件与对象存储（S3 / Cloudflare R2 / MinIO）。

### 体验细节

- 🌍 **多语言** — 中文 / English / 日本語 / Русский 国际化。
- 🌗 **主题** — 亮色 / 深色模式。
- 📱 **PWA** — 移动端「添加到主屏幕」，近似原生 App 体验。
- 🔐 **安全认证** — JWT 签发，可选 Passkey / WebAuthn 无密码登录，私有部署数据自主可控。
- 🔄 **多端同步与分享** — 对话跨设备同步，支持链接 / 图片分享。

---

## 支持模型

| 供应商 | 能力概览 |
|--------|----------|
| **OpenAI & Azure** | Vision、Function Calling、GPT-5 系列、Reasoning Summaries、原生 Web Search、文生图（DALL·E / gpt-image-1） |
| **Anthropic Claude** | Vision、Function Calling |
| **Google Gemini** | Vision、原生 Google Search / URL Context、原生图片生成 |
| **DeepSeek** | V4、Thinking 控制、Prompt Cache 统计 |
| **xAI Grok** | Responses API、原生 Web Search / X Search、Writable Memory、Reasoning |
| **Xiaomi MiMo** | Thinking Toggle、Token Plan China |
| **MiniMax** | Token Plan CN |
| **GLM** | Coding Plan CN |
| **LocalAI / Ollama** | OpenAI 兼容格式（本地模型） |

> 💡 只要符合 OpenAI API 格式，任何供应商都能接入。本地部署的模型用 Ollama / LocalAI 即可。

---

## 部署方式

> [!IMPORTANT]
> 首次启动（用户表为空）自动创建管理员账号，用户名 `root`。密码来源：`ROOT_INITIAL_PASSWORD` 环境变量或 `config.yaml` 的 `root.initial_password`（6–36 位，二选一，环境变量优先）；两者均未设置时系统生成 24 位随机密码并打印到启动日志（用 `docker compose logs` 查看）。完整规则见[常见问题](#常见问题)。

### 方式一：Docker Compose（推荐，最省心）

访问地址：`http://localhost:8000`

```shell
git clone --depth=1 --branch=main --single-branch https://github.com/qwq202/prism.git
cd prism
docker compose up -d
```

**数据与配置目录**（首次启动自动创建）：

| 路径 | 说明 |
|------|------|
| `./db` | MySQL 数据 |
| `./redis` | Redis 数据 |
| `./config` | 配置文件（自动生成 `config.yaml` 与随机 `secret`） |

> [!NOTE]
> MySQL 容器首次初始化后账号密码写入 `./db`。已有数据目录时再改 `.env` 中的 MySQL 密码不会自动迁移，需手动改库内密码或重新初始化。

<details>
<summary><b>使用稳定版镜像（MySQL 5.7 / Redis 8.2）</b></summary>

```shell
docker compose -f docker-compose.stable.yaml up -d
```

适合对数据库版本有兼容性要求的旧环境。

</details>

### 方式二：单容器 Docker（外置 MySQL / Redis）

已有自己的数据库？用这个。访问地址：`http://localhost:8094`

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

| 环境变量 | 说明 |
|----------|------|
| `SECRET` | JWT 密钥，至少 32 位随机字符串 |
| `ROOT_INITIAL_PASSWORD` | 空库首次启动的 `root` 密码（6–36 位）；不设则从日志读随机密码 |
| `SERVE_STATIC` | 是否由后端提供静态文件（默认 `true`） |

```shell
curl http://localhost:8094/health   # 健康检查
docker stop prism && docker rm prism && docker pull qunqin45/prism:latest  # 更新镜像
```

### 方式三：前后端分离

- **前端**：用 Nginx / Vercel 等静态托管，构建时设置 `VITE_BACKEND_ENDPOINT`（如 `https://api.example.com`）
- **后端**：设置 `SERVE_STATIC=false`，API 独立域名部署
- ⚠️ Prism 本体不支持 Vercel 全栈部署，仅可将前端部署至 Vercel

### 🍓 ARM 架构（树莓派 / Apple Silicon）

公开镜像 `qunqin45/prism:latest` 为 `linux/amd64`。ARM 机器可本地源码构建，或使用 BuildX 构建 `linux/arm64` 镜像。

---

## 本地开发

**前置依赖**：Go 1.25+、Node.js（pnpm）、MySQL 8、Redis 7

```shell
# 后端
go build .                    # 编译
go test ./...                 # 测试

# 前端
cd app && pnpm install        # 安装依赖
cd app && pnpm dev            # 启动开发服务器
cd app && pnpm lint           # ESLint 检查
cd app && pnpm build          # 生产构建 → app/dist

# 完整服务栈（数据库 + Redis + 后端）
docker compose up -d
```

后端入口为 `main.go`，前端源码在 `app/src/`，配置参考 [`config.example.yaml`](config.example.yaml)。

---

## 配置说明

最常用的配置项一览（完整内容见 [`config.example.yaml`](config.example.yaml)）：

| 项 | 说明 |
|----|------|
| `secret` | JWT 签名密钥，首次启动可自动生成 |
| `root.initial_password` | 管理员初始密码，等同环境变量 `ROOT_INITIAL_PASSWORD` |
| `serve_static` | 前后端同进程部署时保持 `true` |
| `system.general.backend` | 后端 API 地址；默认同域 `/api`，分离部署时填完整 URL |
| `ALLOW_ORIGINS` | 跨域白名单，逗号分隔域名（无需协议前缀） |
| `system.search.api_key` | Tavily API Key，供非原生搜索模型联网使用 |
| `system.search.depth` | Tavily 搜索深度：`basic` / `advanced` / `fast` / `ultra-fast` |
| `system.task.model` | 联网搜索关键词提取所用模型（可选） |

---

## 常见问题

<details>
<summary><b>聊天加载卡住 / 无响应</b></summary>

聊天走 WebSocket 通信（API 中转走 HTTP，不需要 WebSocket）。请确认你的反向代理（Nginx / Apache）、CDN 或端口转发已启用 WebSocket 支持。

</details>

<details>
<summary><b>🔑 管理员账号与初始密码</b></summary>

首次启动且**用户表为空**时，系统自动创建管理员账号：

| 项 | 值 |
|------|------|
| 用户名 | `root`（固定） |
| 邮箱 | `root@example.com` |
| 角色 | 管理员 |

**初始密码来源**（二选一，环境变量优先于配置文件）：

| 方式 | 写法 | 规则 |
|------|------|------|
| 环境变量 | `ROOT_INITIAL_PASSWORD` | 需 6–36 位 |
| 配置文件 | `config.yaml` 中的 `root.initial_password` | 需 6–36 位 |

- 两者**均未设置**或**长度不合规**时，系统生成 **24 位随机密码**，并在启动日志中以明文打印：
  ```
  [service] no user found, creating root user with generated password (username: root, password: <24位随机>); save it now ...
  ```
  使用 `docker compose logs` 检索该行即可获取。
- 密码**仅在用户表为空时生成一次**。此后修改环境变量或配置文件，均不会再次创建或更新 `root` 密码。

**查看 / 修改密码：**

1. 首次未设密码 → 运行 `docker compose logs` 查看日志中的随机密码
2. 已登录 → 后台 → 系统设置 → 修改 Root 密码；或在用户管理中修改
3. 无法登录时重置：
   - Compose：`docker compose exec chatnio prism root <new-password>`
   - 单容器：`docker exec prism prism root <new-password>`
   - 二进制：`./prism root <new-password>`

</details>

<details>
<summary><b>外部依赖</b></summary>

| 服务 | 用途 | 是否必需 |
|------|------|----------|
| **MySQL** | 用户、对话、配置等持久化数据 | ✅ 必需 |
| **Redis** | 鉴权、限流、订阅配额、验证码等 | ✅ 必需 |

</details>

<details>
<summary><b>计费与订阅</b></summary>

- **弹性计费（点数）**：通用按量计费，默认 10 点数 = 1 元，可在计费规则模板中调整
- **订阅**：固定价格 + 窗口额度；扣费使用点数（如 32 元计划需 ≥ 320 点数）
- 订阅分四个等级：普通用户 (0)、基础版 (1)、标准版 (2)、专业版 (3)，对应渠道用户分组

</details>

<details>
<summary><b>礼品码与兑换码</b></summary>

| 类型 | 特点 | 适用场景 |
|------|------|----------|
| **礼品码** | 同类型每用户仅能兑换一次 | 福利发放、宣传拉新 |
| **兑换码** | 同类型可被多用户兑换 | 发卡销售、批量购买 |

</details>

<details>
<summary><b>报错：user quota is not enough</b></summary>

这是最小请求点数限制：

- 不计费模型：无限制
- 按次计费：最小点数 = 单次请求点数
- 按 Token 计费：最小点数 = 1K 输入价 + 1K 输出价

解决：充值点数，或给用户增加额度。

</details>

<details>
<summary><b>模型映射</b></summary>

渠道内格式为 `[from]>[to]`，每行一条。`from` 是用户请求的模型，`to` 是实际上游模型。

```
gpt-4-all>gpt-4          # 映射 gpt-4-all 到 gpt-4
!gpt-4-all>gpt-4         # 加 ! 前缀：该渠道不暴露 gpt-4，仅暴露 gpt-4-all
```

</details>

<details>
<summary><b>接入支付</b></summary>

在系统设置中配置购买链接（发卡地址）；兑换码在后台批量生成。

</details>

---

## 技术栈

| 层 | 技术 |
|----|------|
| **前端** | React 19、Redux Toolkit、Radix UI、Tailwind CSS、Vite |
| **后端** | Go 1.25、Gin、MySQL、Redis |
| **部署** | Docker、Docker Compose、PWA、WebSocket |

---

## 🤝 支持

- Bug 反馈与功能建议：[提交 Issue](https://github.com/qwq202/prism/issues)
- 版本更新与发布说明：[GitHub Releases](https://github.com/qwq202/prism/releases)

## 🙏 致谢

本项目基于 [coai](https://github.com/coaidev/coai) 进行二次开发，感谢原作者的基础工作与贡献。

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
