# 仓库结构

Language/语言: [English](./repository-layout.md) | 简体中文

该仓库是 OctaBit 的单体仓库。OctaBit 是一个用于将 MIDI 文件转换为 8-bit 风格音乐的简单 Web 工具。当前生产
Web 前端是 `frontend/` 中的 Vue 应用，并从 Vite `dist` 构建产物为 `octabit.cc`
提供服务。主后端是 `backend/` 中的 Go 服务，负责稳定的 `/api/*` 合约、工作区/合成服务、预览和运行时渲染器。

## 顶层结构

| 路径 | 用途 |
| --- | --- |
| `AGENTS.md` | 面向编码代理和本地工作流的仓库说明。 |
| `README.md`, `README.zh-Hans.md` | 根项目概览、设置说明、应用入口和仓库许可证摘要。 |
| `LICENSE.md` | 仓库 AGPL 许可证文本。 |
| `frontend/` | 生产 Vue/Vite 前端。 |
| `backend/` | 主 Go 后端模块和冻结 Python baseline fixtures。 |
| `assets/previews/` | 规范的波形预览 WAV 文件。 |
| `docs/` | API 契约、仓库结构说明、agent 本地化流程、许可证审计和评审报告。 |
| `deploy/production/` | Vue 生产路径的非 Docker 生产部署说明、辅助脚本和 Caddy 示例。 |
| `scripts/` | 显式本地维护脚本。 |
| `.gitignore`, `.gitattributes` | 仓库忽略和换行规则。 |
| `output/`, `tmp/` | 已跟踪的历史生成评审产物；两个路径都被忽略，用于未来生成输出。 |

构建输出、`.codex/`、`.sisyphus/`、`.DS_Store` 和各应用的 `build/` 目录等
本地目录不属于维护中的源码结构。

### `backend/`

主 Go 后端运行时。它通过冻结的 Python baseline fixtures 满足 OpenAPI 契约，同时让常规运行时不再依赖 Python。

- `go.mod`：后端的 Go module。
- `cmd/server/`：进程启动、环境配置、SQLite workspace store 打开、日志、graceful shutdown 和 HTTP server wiring。
- `internal/config/`：环境变量解析和默认值。
- `internal/httpapi/`：health、workspace state/upload/queue/config API、workspace-backed `/api/synthesis-jobs` JSON 和 multipart render/poll/download/delete flows，以及 previews 的 HTTP routes、route tests 和 OpenAPI route-registration conformance。
- `internal/jobs/`：bounded render execution 和与 workspace-backed job flow 共享 queue 语义的内存 render job lifecycle tests。
- `internal/midi/`：使用 `gitlab.com/gomidi/midi/v2/smf` 提取 Standard MIDI File 音符。
- `internal/renderer/`：渲染限制、声音层校验、频率曲线插值、曲线 hash、输出命名和 note-event PCM/WAV synthesis。
- `internal/storage/`：SQLite schema、connection pragmas、token hashing、路径 helpers 和 workspace cleanup/cascade 行为。
- `internal/workspace/`：工作区 token lifecycle、state payloads、upload queue 操作、limits、配置持久化、SQLite-backed synthesis jobs、WAV output cleanup 和 renderer form payload 转换。
- `testdata/python-baseline/`：规范化 API transcripts、代表性 MIDI 输入、期望 WAV
  输出、parsed note-event fixtures、渲染器命名/hash 预期，以及工作区配置规范化案例。

## 应用目标

### `frontend/`

公开浏览器体验的生产 Vue/Vite 前端。

- `index.html`：Vite 应用外壳。
- `src/App.vue`：顶层 Vue 工作流和状态编排。
- `src/api/`：后端 `/api/*` 路由的类型化客户端。
- `src/components/`：上传队列、声音层编辑器、输出控制、头部控制、已转换文件和曲线编辑器组件。
- `src/i18n/`：英文、西班牙文、法文、日文、简体中文和繁体中文前端 catalog。
- `src/styles/app.css`：当前 OctaBit 视觉系统。
- `vite.config.ts`：开发环境中把 `/api` 和 `/static/previews` 代理到
  `http://127.0.0.1:8000`。
- `package.json` 和 `package-lock.json`：Vue/Vite 依赖元数据。

生产 Caddy 提供 `frontend/dist`，并把 API 和预览资源请求代理到 Go 后端。


## 共享核心和资源

### `assets/previews/`

Web 前端/后端路径使用的规范预览 WAV 资源。`assets/README.md` 记录了它们的预期
用途和来源说明。

## 文档和生成产物

- `docs/api-contract.md`、`docs/api-contract.zh-Hans.md` 和 `docs/openapi.yaml`：当前 Web API 契约、任务载荷和公开演示安全边界。
- `docs/repository-layout.md` 和 `docs/repository-layout.zh-Hans.md`：当前仓库结构的英文和简体中文说明。
- `docs/localisation.md`：面向 agent 的英文生产 UI 本地化及相关文档更新标准流程。
- `docs/licensing-audit.md`：面向仓库和发布规划的许可证与署名审计。
- `output/pdf/repo-structure-evaluation.pdf`、
  `tmp/pdfs/repo-structure-evaluation.html` 和
  `tmp/pdfs/rendered/repo-structure-evaluation.png`：已跟踪的历史生成评审产物。它们不是当前结构的事实来源。

## 构建和开发流程

除非某个文档另有说明，否则从仓库根目录运行命令。

常规检查：

```bash
cd backend && go test ./...
cd frontend && npm run build
```

Vue 开发时，先在 8000 端口运行 Go 后端，再启动 Vite dev server：

```bash
cd backend
PORT=8000 go run ./cmd/server
```

```bash
cd frontend
npm ci
npm run dev
```

Vue 生产构建：

```bash
cd frontend
npm ci
npm run build
```

当前非 Docker 生产路径私有运行 Go 后端于 `127.0.0.1:8000`，systemd 管理服务，Caddy 提供
`frontend/dist`，并将 `/api/*` 和 `/static/previews/*` 反向代理到该私有监听地址。工作区/任务目录、任务 TTL、最大上传大小和渲染 worker 设置应与当前合成任务行为保持一致。Caddy 生产和回滚示例见 `deploy/production/README.zh-Hans.md`。

## 依赖和打包边界

- Go 后端依赖位于 `backend/go.mod` 和 `backend/go.sum`。
- 生产前端 JavaScript 依赖位于 `frontend/package.json` 和
  `frontend/package-lock.json`。

## 归属边界

- 运行时渲染行为属于 `backend/internal/renderer/`。
- 生产 Web UI 属于 `frontend/`。
- 共享二进制/媒体资源属于 `assets/`。
- 仓库级文档、审计和评审记录属于 `docs/`。
- 部署专用文件属于 `deploy/`。
