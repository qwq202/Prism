# 仓库规范指南

## 仓库结构与模块组织
本仓库是 Go 后端 + 独立 React 客户端（`app/`）的组合。后端入口为 `main.go`，主要领域模块包括 `adapter/`（模型适配）、`admin/`、`auth/`、`channel/`、`manager/`、`middleware/`、`utils/`。前端源码位于 `app/src/`，按功能划分为 `components/`、`routes/`、`store/`、`admin/`、`assets/` 等目录。部署相关文件在仓库根目录，包括 `Dockerfile`、`docker-compose*.yaml`、`nginx.conf`、`config.example.yaml`。

## 构建、测试与开发命令
- `go build .`：在仓库根目录构建后端可执行文件。
- `go test ./...`：执行后端包检查。当前该命令在 `utils/image.go` 处会失败，视为“待修复的必经检查”而非通过基线。
- `cd app && pnpm install && pnpm dev`：本地启动 Vite 前端开发环境。
- `cd app && pnpm lint`：执行 TypeScript/React 的 ESLint 检查。
- `cd app && pnpm build`：构建生产前端包，输出到 `app/dist`。
- `docker-compose up -d`：本地启动完整服务栈；若使用稳定版镜像，请改用 `docker-compose -f docker-compose.stable.yaml up -d`。

## 编码风格与命名规范
后端遵循 `gofmt` 格式化和惯用的、全小写的 Go 包名。后端文件按职责拆分，保持已有命名风格，例如 `router.go`、`controller.go`、`types.go`。前端使用 TypeScript，缩进 2 个空格，遵循 `app/.prettierrc.json` 规则。React 组件文件采用 `PascalCase` 命名，如 `ChatInterface.tsx`；公共 UI 组件在 `app/src/components/ui/` 内使用小写短横线命名，如 `alert-dialog.tsx`。

## 测试规范
本仓库未配置专门的前端测试运行器，且目前多数 Go 包没有 `_test.go` 文件。当前阶段请至少通过 `go test ./...`、`cd app && pnpm lint`、`cd app && pnpm build` 来验证变更。新增 Go 测试请与目标包同目录创建 `*_test.go`，新增前端测试请尽量贴近对应功能目录组织。

## 提交与拉取请求规范
最近提交记录使用 Conventional Commit 约定，常见前缀如 `feat:`、`chore:`，请继续沿用，并保持标题简洁明确。Pull Request 需包含清晰改动说明、受影响模块（如 `adapter`、`admin`、`app` 等）、必要时关联 issue，并在界面改动时附上截图或短视频。若涉及配置、数据结构或部署参数，请在说明中单独标出，方便评审进行端到端验证。

## 分支与发布规程
- `main` 是当前唯一的日常开发与发布分支。所有代码、配置、文档等实际修改默认直接提交到 `main`，不再使用 `Preview` 中转。
- 已知 bug、快速修复、小范围调整和普通功能改动都直接在 `main` 完成；必要时在提交前通过本地构建、测试或手动验证确认稳定。
- 正式 Release 只从 `main` 创建。发布时使用语义化版本标签，例如 `v1.0.0`、`v1.0.1`、`v1.1.0`。
- 版本号规则：补丁修复使用 `vX.Y.Z+1`，兼容性功能新增使用 `vX.Y+1.0`，破坏性变更使用新的主版本号。
- Docker 镜像发布规则：`main` 对应正式镜像 `qunqin45/prism:latest`；正式 tag 版本应同时发布 `qunqin45/prism:<version>`。
- GitHub Release 内容需要包含：新增功能、修复问题、兼容性或数据库/配置变更、Docker 部署方式、升级注意事项。
- 正式发布前至少确认：`go build .`、`cd app && pnpm build`、Docker Actions、README 部署说明和目标 Docker 镜像拉取均正常。

## 变更留痕要求
所有实际代码或配置修改都必须保留清晰留痕。每完成一轮独立修改后，需要立即创建一次 Git 提交，不要将多轮无关修改长期堆积在工作区中再统一提交。
