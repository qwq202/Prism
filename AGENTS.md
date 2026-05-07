# 仓库规范指南

## 仓库结构与模块组织
本仓库是 Go 后端 + 独立 React 客户端（`app/`）的组合。后端入口为 `main.go`，主要领域模块包括 `adapter/`（模型适配）、`admin/`、`auth/`、`channel/`、`manager/`、`middleware/`、`utils/`。前端源码位于 `app/src/`，按功能划分为 `components/`、`routes/`、`store/`、`admin/`、`assets/` 等目录。Tauri 桌面端打包位于 `app/src-tauri/`。部署相关文件在仓库根目录，包括 `Dockerfile`、`docker-compose*.yaml`、`nginx.conf`、`config.example.yaml`。

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


