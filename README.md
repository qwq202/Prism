## 🖼️ 预览

**原生联网搜索** — GPT 与 Gemini 系列模型直连互联网，无需中间层转发，实时获取最新信息，让回答始终与时俱进。

![image-20260508092832242](docs/image/image-20260508092832242.png)

**个性化持久记忆** — 对标 ChatGPT 官方同款记忆机制，跨会话保存用户偏好与个人信息，让模型真正"认识"每一位用户。

![image-20260508093038845](docs/image/image-20260508093038845.png)

**双窗口灵活配额** — 短周期（5 小时）+ 每周额度双窗口计费模式，两套配额独立计算、独立重置，更公平地分配用量，有效防止集中突发消耗。

![image-20260508093148510](docs/image/image-20260508093148510.png)

**强大管理后台** — 一站式后台管理中心，涵盖用户管理（含批量操作与直接创建）、渠道配置、订阅套餐、模型市场、公告通知等全维度运营能力，界面简洁直观。

![image-20260508093251949](docs/image/image-20260508093251949.png)

![image-20260508093321382](docs/image/image-20260508093321382.png)

**精细用量追踪** — 完整的请求日志与用量明细，Token 消耗、费用账单一览无遗，助力运营分析、成本核查与异常排查。

![image-20260508093442056](docs/image/image-20260508093442056.png)

**Tavily 搜索 & 任务编排** — 集成 Tavily 第三方搜索 API，配合任务编排模型实现更精准的检索增强生成能力，让复杂任务拆解与信息检索更高效。

![image-20260508093619270](docs/image/image-20260508093619270.png)

**多云对象存储适配** — 兼容 AWS S3、Cloudflare R2 及 MinIO 等主流对象存储协议，文件上传与附件管理轻松对接自有存储资源，无缝融入现有基础设施。

![image-20260508093726100](docs/image/image-20260508093726100.png)

## ✨ 相比原项目的新增与改进

以下内容根据本仓库提交历史整理，集中说明当前版本在原项目基础上新增、重构和强化的能力：

### 1. 模型与供应商能力扩展
- **OpenAI Responses / GPT-5 系列增强**：新增 GPT-5.5 OpenAI Responses 支持、GPT-5.4 Mini Reasoning 支持，并完善 OpenAI Reasoning Summaries 的开关、分段滑块、当前模型标签和本地化提示。
- **DeepSeek V4 / Thinking 适配**：适配 DeepSeek V4 模型，新增 Thinking 控制、努力程度分级、Pro Max 推理提醒、Prompt Cache 用量记录，并修复 Thinking 流式输出、工具流、空白保留和标签解析等细节。
- **xAI 能力补强**：支持 xAI Responses、联网搜索、Reasoning 摘要以及 Writable Memory 写入，为 Grok 类模型提供更完整的原生能力入口。
- **Xiaomi MiMo / MiniMax / GLM 扩展**：新增 Xiaomi Token Plan China 适配器、MiMo Thinking Toggle、小米模型头像和工具调用增量处理，同时补充 MiniMax Token Plan CN 与 GLM Coding Plan CN 渠道支持。
- **上下文窗口可配置**：新增模型上下文窗口滑块，并限制滑块范围，方便管理员按模型能力控制上下文长度。

### 2. 工具调用与对话体验升级
- **内置网页抓取工具**：新增 Fetch Webpage 工具，并可作为模型原生 Tool Use 暴露；支持更长工具链、工具调用状态展示、详情弹窗和工具结果后的最终回答流式输出。
- **工具流稳定性修复**：修复带工具调用的首轮流式聊天、DeepSeek 工具调用标记清洗、工具状态按钮尺寸和多轮工具会话收尾问题。
- **Thinking 内容体验优化**：默认折叠思考内容，修复非思考文本混入 Thinking 区域的问题，并统一 OpenAI / DeepSeek 等模型的推理提示展示。
- **语音与长文本输入增强**：新增浏览器原生语音识别输入，补充友好的识别错误提示；粘贴超长文本时自动转为附件文件，降低对话上下文污染。
- **登录态与操作体验优化**：修复浏览器重启、标签页会话中的登录态保持问题，并优化消息操作菜单的保持行为。

### 3. 订阅与额度机制重构
- **只保留窗口额度计费模式**：订阅套餐从按次计费迁移为窗口额度机制，核心支持类似“短周期窗口 + 每周窗口”的双窗口额度。
- **短周期与每周额度双窗口**：支持 5 小时额度、每周额度、独立开关、独立重置、用量展示和倒计时显示；管理员可分别重置小时限制与周限制。
- **套餐级额度池**：订阅点数从按模型分组使用改为套餐级额度池，窗口额度套餐不配置可访问模型时默认覆盖全部模型。
- **扣费一致性与并发安全**：修复短窗口和周窗口扣除回滚、读路径覆盖扣点、最终扣费失败免费放行等问题，并将关键额度消耗改为原子化处理。
- **套餐售卖与降级逻辑**：支持三个订阅等级分别开启或关闭售卖；修复跨计费类型降级导致套餐丢失、免费降级丢失到期时间、窗口额度切换后周进度条不显示等问题。
- **订阅 UI 清理**：移除按次计费入口、额度标签、重复价格标签和订阅同步上游入口；优化钱包页默认月度计划、空用量隐藏、周额度保存回显和窗口额度文案。

### 4. 管理后台与运营能力增强
- **用户管理增强**：新增后台直接添加用户入口，支持批量封禁、解封、点数变更等操作，并统一批量操作按钮尺寸与交互文案。
- **订阅管理增强**：新增套餐是否上架售卖控制，补充短周期额度、周额度、可访问模型分组、折扣设置等窗口额度配置。
- **模型市场展示优化**：模型卡片中过长的模型名、模型 ID、标签和时间文本会自动省略，避免撑破卡片布局。
- **国际化补全**：补充批量操作成功等后台提示文案，修复部分管理端和推理设置文案未本地化的问题。
- **品牌与仓库整理**：完成新的品牌命名和仓库命名调整，补充 `.DS_Store` 忽略规则，清理无用模型模板、旧模型别名和未使用能力开关。

### 5. 安全性与稳定性加固
- **鉴权安全**：强化 JWT 解析、认证密钥处理、认证比较逻辑，停止在会话中保存认证密码，并修复浏览器端 Token 与 URL 处理风险。
- **限流与请求安全**：修复 Forwarded IP 绕过限流的问题，收紧 CORS 默认行为，并加固计费查询过滤与管理员日志路径访问。
- **运行时与迁移安全**：增强运行时和数据库迁移的安全边界，限制异步聊天和预热任务范围，避免后台任务无界增长。
- **异常场景兜底**：修复空响应、消息索引越界、路由类型、系统配置类型、组件类型构建错误等问题，提升生产环境容错性。

## 📝 功能
1. 🤖️ **丰富模型支持**: 多模型服务商支持 (OpenAI / Anthropic / Gemini / DeepSeek / xAI / Xiaomi MiMo / MiniMax / GLM 等十余种格式兼容 & 私有化 LLM 支持)
2. 🤯 **美观 UI 设计**: UI 兼容 PC / Pad / 移动三端，遵循 [Shadcn UI](https://ui.shadcn.com) & [Tremor Charts](https://blocks.tremor.so) 设计规范，丰富美观的界面设计和后台仪表盘
3. 🎃 **完整 Markdown 支持**: 支持 **LaTeX 公式** / **Mermaid 思维导图** / 表格渲染 / 代码高亮 / 图表绘制 / 进度条等进阶 Markdown 语法支持
4. 👀 **多主题支持**: 支持多种主题切换，包含亮色主题的**明亮模式**和暗色主题的**深色模式**。 👉 [自定义配色](https://github.com/coaidev/coai/blob/main/app/src/assets/globals.less)
5. 📚 **国际化支持**: 支持国际化，支持多语言切换 🇨🇳 🇺🇸 🇯🇵 🇷🇺 👉 欢迎贡献翻译 [Pull Request](https://github.com/coaidev/coai/pulls) 
6. 🎨 **文生图支持**: 支持多种文生图模型: **OpenAI DALL-E**✅ & Stable Diffusion✅ 等，含精美绘图工作台界面
7. 📡 **强大对话同步**: **用户 0 成本对话跨端同步支持**，支持**对话分享** (支持链接分享 & 保存为图片 & 分享管理), **无需 WebDav / WebRTC 等依赖和复杂学习成本**
8. 🎈 **模型市场 & 预设系统**: 支持后台可自定义的模型市场, 可提供模型介绍、标签等参数, 站长可根据情况自定义模型简介。同时支持预设系统，包含 **自定义预设** 和 **云端同步** 功能。
9. 📖 **丰富文件解析**: **开箱即用**, 支持**所有模型**的文件解析 (PDF / Docx / Pptx / Excel / 图片等格式解析), **支持更多云端图片存储方案** (S3 / R2 / MinIO 等), **支持 OCR 图片识别** 👉 详情参见项目 [CoAI.Dev Blob Service](https://github.com/coaidev/blob-service) (支持 Vercel / Docker 一键部署)
10. 🌏 **全模型联网搜索**: 基于 [SearXNG](https://github.com/searxng/searxng) 开源引擎, 支持 Google / Bing / DuckDuckGo / Yahoo / WikiPedia / Arxiv / Qwant 等丰富搜索引擎搜索, 支持安全搜索模式, 内容截断, 图片代理, 测试搜索可用性等功能。
11. 💕 **渐进式 Web 应用 (PWA)**: 支持 PWA 应用 & 支持桌面端 (桌面端基于 [Tauri](https://github.com/tauri-apps/tauri))
12. 🤩 **齐全后台管理**: 支持美观丰富的仪表盘, 公告 & 通知管理, 用户管理 (含**批量操作**与**管理员直接创建用户**), 订阅管理, 礼品码 & 兑换码管理, 价格设定, 订阅设定, 自定义模型市场, 自定义站点名称 & Logo, SMTP 发件设置, **附件管理**等功能
13. 🤑 **多种计费方式**: 支持 💴 **订阅制** 和 💴 **弹性计费** 两种计费方式, 弹性计费支持 次数计费 / Token 计费 / 不计费 / 可匿名调用 和 **最小请求点数** 检测等强大功能；订阅支持**灵活配额窗口**设置与**计划可用性控制**
14. 🎉 **创新模型缓存**: 支持开启模型缓存：即同一个请求入参 Hash 下, 如果之前已请求过, 将直接返回缓存结果 (击中缓存将不计费), 减少请求次数。可自行自定义是否缓存的模型、缓存时间、多种缓存结果数等高级缓存设置
15. 😎 **优秀渠道管理**: 自写优秀渠道算法, 支持⚡ **多渠道管理**, 支持🥳**优先级**设置渠道的调用顺序, 支持🥳**权重**设置同一优先级下的渠道均衡负载分配概率, 支持🥳**用户分组**, 🥳**失败自动重试**, 🥳**模型重定向**, 🥳**内置上游隐藏**, 🥳**渠道状态管理**等强大**企业级功能**
16. ⭐ **OpenAI API 分发 & 中转系统**: 支持以 **OpenAI API** 标准格式调用各种大模型, 集成强大的渠道管理功能, 仅需部署一个站点即可实现同时发展 B/C 端业务💖
17. 🧠 **原生 Reasoning / Thinking 支持**: 支持 OpenAI Reasoning Summaries、DeepSeek Thinking 控制（含努力程度分级）、xAI 推理摘要流式输出、Xiaomi MiMo Thinking Toggle，以及**可配置上下文窗口**滑块
18. 🛠️ **模型工具调用 (Tool Use)**: 支持实时工具调用状态展示与详情弹窗，内置**网页抓取工具** (Fetch Webpage)，可作为模型原生工具暴露，支持长链工具调用链
19. 🧬 **持久化记忆与个性化**: 支持用户个性化设置与**持久化记忆系统**，xAI Writable Memory 写入，支持在对话中自动注入当前模型信息与客户端设备上下文
20. 🎙️ **浏览器语音识别 (STT)**: 集成浏览器原生语音识别，支持语音输入转文字，并提供友好的识别错误提示
21. 📋 **长文本粘贴为文件**: 粘贴超长文本时自动转换为附件文件，避免上下文污染
22. 👌 **快速同步上游**: 渠道设置、模型市场、价格设定等设置都可快速同步上游站点，以此基础修改自己的站点配置，快速搭建自己的站点，省时省力，一键同步，快速上线
23. 👋 **SEO 优化**: 支持 SEO 优化，支持自定义站点名称、站点 Logo 等 SEO 优化设设置使搜索引擎更快的爬取，你的站点与众不同👋
24. 🎫 **多种兑换码体系**: 支持多种兑换码体系，支持礼品码和兑换码，支持批量生成，礼品码适合宣传分发，兑换码适合发卡销售，礼品码一个类型的多个码一个用户仅能兑换一个码，在宣传中一定程度上减少一个用户兑换多次的情况😀
25. 🥰 **商用友好协议**: 采用 **Apache-2.0** 开源协议, 商用二开 & 分发友好 (也请遵守 Apache-2.0 协议的规定, 请勿用于违法用途)


## 🔨 支持模型
1. OpenAI & Azure OpenAI *(✅ Vision ✅ Function Calling ✅ GPT-5 全系列 ✅ Reasoning Summaries)*
2. Anthropic Claude *(✅ Vision ✅ Function Calling ✅ 最新 Claude API)*
3. Google Gemini & PaLM2 *(✅ Vision)*
4. 深度求索 DeepSeek AI *(✅ DeepSeek V4 ✅ Thinking 控制 ✅ Prompt Cache 统计)*
5. xAI Grok *(✅ Responses API ✅ 联网搜索 ✅ Writable Memory ✅ Reasoning)*
6. Xiaomi MiMo *(✅ Thinking Toggle ✅ Token Plan China 适配)*
7. MiniMax *(✅ Token Plan CN 渠道)*
8. GLM *(✅ Coding Plan CN 渠道)*
9. LocalAI / Ollama (👉 OpenAI)

## 👻 中转 OpenAI 兼容 API
   - [x] Chat Completions _(/v1/chat/completions)_
   - [x] Image Generation _(/v1/images)_
   - [x] Model List _(/v1/models)_
   - [x] Dashboard Billing _(/v1/billing)_


## 📦 部署方式
> [!TIP]
> **部署成功后, 管理员账号为 `root`, 密码默认为 `chatnio123456`**

### ⚡ Docker Compose 安装 (推荐)
> [!NOTE]
> 运行成功后, 宿主机映射地址为 `http://localhost:8000`

正式版适合生产环境和长期使用，预览版包含最新修复与实验性改动，适合提前验证已知问题修复。

**部署正式版**

```shell
git clone --depth=1 --branch=main --single-branch https://github.com/qwq202/prism.git
cd prism
docker compose up -d # 运行服务
# 如需使用 watchtower 自动更新, 请使用 docker compose -f docker-compose.watch.yaml up -d 替代
```

**部署预览版**

```shell
git clone --depth=1 --branch=Preview --single-branch https://github.com/qwq202/prism.git prism-preview
cd prism-preview
PRISM_IMAGE_TAG=preview docker compose up -d # 运行预览版服务
# 如需使用 watchtower 自动更新, 请使用 PRISM_IMAGE_TAG=preview docker compose -f docker-compose.watch.yaml up -d 替代
```

版本更新（_开启 Watchtower 自动更新的情况下, 无需手动更新_）：
```shell
docker compose down
docker compose pull
docker compose up -d
```

预览版更新时请保留镜像标签：

```shell
docker compose down
PRISM_IMAGE_TAG=preview docker compose pull
PRISM_IMAGE_TAG=preview docker compose up -d
```

> - MySQL 数据库挂载目录项目 ~/**db**
> - Redis 数据库挂载目录项目 ~/**redis**
> - 配置文件挂载目录项目 ~/**config**

### ⚡ Docker 安装 (轻量运行时, 常用于外置 _MYSQL/RDS_ 服务)
> [!NOTE]
> 运行成功后, 宿主机地址为 `http://localhost:8094`

正式版镜像为 `qunqin45/prism:latest`，预览版镜像为 `qunqin45/prism:preview`。下方示例默认使用正式版，如需预览版请把最后一行镜像标签改为 `qunqin45/prism:preview`。

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
   -e SECRET=secret \
   -e SERVE_STATIC=true \
   qunqin45/prism:latest
```

> - *--network host* 指使用宿主机网络, 使 Docker 容器使用宿主机的网络, 可自行修改
> - SECRET: JWT 密钥, 自行生成随机字符串修改
> - SERVE_STATIC: 是否启用静态文件服务 (正常情况下不需要更改此项)
> - *-v ~/config:/config* 挂载配置文件, *-v ~/logs:/logs* 挂载日志文件, *-v ~/storage:/storage* 挂载附加功能的生成文件
> - 需配置 MySQL 和 Redis 服务, 请自行参考上方信息修改环境变量

版本更新：
```shell
docker stop prism
docker rm prism
docker pull qunqin45/prism:latest
```
预览版请改为 `docker pull qunqin45/prism:preview`。拉取完成后按上方 `docker run` 命令重新启动容器。

### 🖥️ 桌面端打包
桌面端基于 Tauri，只打包现有前端界面，运行时连接已部署的 Prism 服务端。

```shell
cd app
pnpm install
pnpm tauri:build
```

开发调试可使用：

```shell
cd app
pnpm tauri:dev
```

桌面端默认连接官方服务 `https://prism.qunqin.org`。如需连接自建服务，可在打包或开发时通过 `VITE_DESKTOP_BACKEND_ENDPOINT=https://example.com pnpm tauri:build` 覆盖；也可以在桌面端启动后到设置中的“服务端地址”里修改，保存后会自动重新载入。该地址填写站点根地址即可，程序会自动使用对应的 `/api` 接口地址。

## ❓ 常见问题 Q&A
1. **为什么我部署后的站点可以访问页面, 可以登录注册, 但是无法使用聊天 (一直在转圈)？**
   - 聊天等此类功能通过 websocket 进行通信, 请确保你的服务支持 websocket。 (Tip: 中转通过 Http 实现, 无需 websocket 支持)
   - 如果你使用了 Nginx, Apache 等反向代理, 请确保已配置 websocket 支持。
   - 如果使用了端口映射, 端口转发, CDN, API Gateway 等服务, 请确保你的服务支持并开启 websocket。
2. **此项目有什么外部依赖？**
   - MySQL: 存储用户信息, 对话记录, 管理员信息等持久化数据。
   - Redis: 存储用户快速鉴权信息, IP 速率限制, 订阅配额, 邮箱验证码等数据。
   - 环境未配置好的情况下, 会导致服务无法正常运行, 请确保你的 MySQL 和 Redis 服务已正常运行 (Docker 部署, 编译部署需自行搭建外部服务)。
3. **我的机器为 ARM 架构, 该项目支持 ARM 架构吗？**
   - 当前公开镜像 `qunqin45/prism:latest` 和 `qunqin45/prism:preview` 由 GitHub Actions 自动构建并发布 `linux/amd64` 版本。
   - ARM 机器可在本机源码构建，或自行使用 BuildX 构建 `linux/arm64` 镜像；如果你使用 x86 机器编译, 请使用 `GOARCH=arm64 go build -o prism` 进行交叉编译并上传至 ARM 机器上运行。
4. **如何修改 Root 默认密码？**
   - 请点击右上角头像或侧边栏底部用户框进入后台管理, 点击系统设置下常规设置操作栏的 修改 Root 密码 进行修改。或者选择在 用户管理 中选定 root 用户进行修改密码操作。
5. **系统设置中的后端域名是什么？**
   - 后端域名是指后端 API 服务的地址, 默认为你访问站点后加 `/api` 的地址, 如 `https://example.com/api` 。
   - 如果设置为非 *SERVE_STATIC* 模式, 开启前后端分离部署, 请将后端域名设置为你的后端 API 服务地址, 如 `https://api.example.com`。
6. **如何配置支付方式？**
   - CoAI.Dev 开源版支持发卡模式, 设置系统设置中的购买链接为你的发卡地址即可。卡密可通过用户管理中兑换码管理中批量生成。
7. **礼品码和兑换码有什么区别？**
   - 礼品码一种类型只能一个用户只能绑定一次, 而非 aff code, 发福利等方式可使用礼品码, 可在头像下拉菜单中的礼品码中兑换。
   - 兑换码一种类型可以多个用户绑定, 可作为正常购买和发卡使用, 可在用户管理中的兑换码管理中批量生成, 在头像下拉菜单的点数（菜单第一个）内输入兑换码进行兑换。
   - 一个例子：比如我发了一个类型为 *新年快乐* 的福利, 此时推荐使用礼品码, 假设发放 100 个 66 点数, 如果为兑换码, 手快的一个用户就批量把所有兑换码的 6600 点数都用完了, 而礼品码则可以保证每个用户只能使用一次 (获得 66 点数)。
   - 而搭建发卡的时, 如果用礼品码, 因为一个类型只能兑换一次, 购买多个礼品码会导致兑换失败, 而兑换码则可以在此场景下使用。
8. **该项目支持 Vercel 部署吗？**
   - CoAI.Dev 本身并不支持 Vercel 部署, 但是你可以使用前后端分离模式,  Vercel 部署前端部分, 后端部分使用 Docker 部署或编译部署。
10. **前后端分离部署模式是什么？**
    - 正常情况下, 前后端在同一服务内, 后端地址为 `/api`。前后端分离部署指前端和后端分别部署在不同的服务上, 前端服务为静态文件服务, 后端服务为 API 服务。
      - 举个例子, 前端使用 Nginx (或 Vercel 等) 部署, 部署的域名为 `https://www.chatnio.net`。
      - 后端使用 Docker 部署, 部署的域名为 `https://api.chatnio.net`。
    - 此种部署方式需自行打包前端, 配置环境变量 `VITE_BACKEND_ENDPOINT` 为你的后端地址, 如 `https://api.chatnio.net`。
    - 配置后端环境变量的 `SERVE_STATIC=false` 使后端服务不提供静态文件服务。
11. **弹性计费和订阅详解**
    - 弹性计费, 即 `点数`, 其图标类似于**云**, 模型计费通用方式, 为了防止虚假汇率, 写死 10 点数 = 1 元, 汇率可以在计费规则中的 **应用内置模板** 中自定义汇率。
    - 订阅, 即订阅计划, 为固定价格计费方式按次配额, 订阅计费扣取点数 (举例: 如果站点的用户想订阅 32 元的计划, 则需要保证点数大于等于 320 点数)
    - 订阅是 Item 的组合, 每个 Item 都可设置涵盖的模型, 订阅配额 (-1 为无限使用), 名称, ID (用于区分不同的 Item), 图标等。可在后台的订阅管理中进行操作, 是否开启订阅, 订阅价格等, 修改每个订阅等级的 Item, 以及支持直接导入其他订阅等级的 Item。
    - 订阅支持分层并写死为三个等级。 等级分别为: _普通用户 (0)_, _基础版订阅 (1)_, _标准版订阅 (2)_, _专业版订阅 (3)_, 订阅等级即为用户分组, 可在渠道管理中进行高级设置, 选择勾选可使用此模型的用户分组。
    - 订阅配额设置, 可在订阅管理中进行操作, 是否支持中转 API (默认关闭)
12. **可请求最小点数检测 `user quota is not enough` 详解**
    - 为防止站点用户滥用站点模型, 当请求点数低于最小请求点数时将返回点数不足的错误信息, 大于等于最小请求点数时将正常请求。
    - 模型的最小可请求点数规则: 
        - 不计费模型无限制
        - 次数计费模型最小点数为该模型的 1 次请求点数 (e.g. 若一个模型的单次请求点数为 0.1 点数, 则最小请求点数为 0.1 点数)
        - Token 弹性计费模型为 1K 输入 Tokens 价格 + 1K 输出 Tokens 价格 (e.g. 若一个模型的 1K 输入 Tokens 价格为 0.05 点数, 1K 输出 Tokens 价格 0.1 点数, 则最小请求点数为 0.15 点数)
13. **为何我的 GPT-4-All 等逆向模型无法使用上传文件中的图片?**
    - 上传模型图片为 Base64 格式, 如果逆向不支持 Base64 格式, 请使用 URL 直链而非上传文件做法。
14. **如何开始域名严格跨域检测?**
    - 正常情况下，后端对所有域名开放跨域。如果非特殊需求，无需开启严格跨域检测。
    - 如果需要开启严格跨域检测，可以在后端环境变量中 并配置 `ALLOW_ORIGINS`, 如 `ALLOW_ORIGINS=chatnio.net,chatnio.app` （不需要加协议前缀, www 解析无需手动添加, 后端将自动识别并允许跨域）, 这样就会支持严格跨域检测 (如 *http://www.chatnio.app*, *https://chatnio.net* 等将会被允许, 其他域名将会被拒绝)。
    - 即使在开启严格跨域检测的情况下, /v1 接口会被仍然允许所有域的跨域请求, 以保证中转 API 的正常使用。
15. **模型映射功能是如何使用的？**
    - 渠道内的模型映射格式为 `[from]>[to]`, 多个映射之间换行, **from** 为请求的模型, **to** 为真实向上游发送的模型并且需要上游真实支持
    - 如: 我有一个逆向渠道, 填写 `gpt-4-all>gpt-4`, 则我的用户请求 **gpt-4-all** 模型到该渠道时, 后端则会模型映射至 **gpt-4** 向该渠道请求 **gpt-4**, 此时该渠道支持 2 个模型, **gpt-4** 和 **gpt-4-all** (本质上都为 **gpt-4**)
    - 如果我不想让我的这个逆向渠道影响到 **gpt-4** 的渠道组, 可以加前缀 `!gpt-4-all>gpt-4`, 该渠道 **gpt-4** 则会被忽略, 此时该渠道将只支持 1 个模型, **gpt-4-all** (但本质上为 **gpt-4**)

## 📦 技术栈
- 🥗 前端: React + Redux + Radix UI + Tailwind CSS
- 🍎 后端: Golang + Gin + Redis + MySQL
- 🍒 应用技术: PWA + WebSocket

## ❤ 支持
如果您觉得这个项目对您有所帮助, 您可以点个 Star 支持一下！
