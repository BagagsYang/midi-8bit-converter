# 仓库结构

Language/语言: [English](./repository-layout.md) | 简体中文

该仓库是 OctaBit 的单体仓库。OctaBit 是一个用于将 MIDI 文件转换为 8-bit 风格音乐的简单 Web 工具。当前生产
Web 前端是 `frontend/` 中的 Vue 应用，并从 Vite `dist` 构建产物为 `octabit.cc`
提供服务。主后端是 `backend/` 中的 Go 服务，负责稳定的 `/api/*` 合约、工作区/合成服务、兼容路由、预览和运行时渲染器。Flask 和 Python 渲染器保留在 `legacy/` 下，用于 fixture 重新生成和回退参考。原生 macOS 和 Windows 应用已 deprecated/paused，不再作为活跃开发目标；代码保留用于参考或未来可能的恢复。

## 顶层结构

| 路径 | 用途 |
| --- | --- |
| `AGENTS.md` | 面向编码代理和本地工作流的仓库说明。 |
| `README.md`, `README.zh-CN.md` | 根项目概览、设置说明、应用入口和仓库许可证摘要。 |
| `LICENSE.md` | 仓库 AGPL 许可证文本。 |
| `frontend/` | 生产 Vue/Vite 前端。 |
| `backend/` | 主 Go 后端模块和冻结 Python baseline fixtures。 |
| `legacy/web-flask/` | 保留作为 parity 参考的旧 Flask 后端/API 和 Flask 渲染前端回退。 |
| `legacy/python-renderer/` | 规范 Python MIDI 转 WAV parity 参考实现。 |
| `legacy/native/` | 已暂停/弃用的 macOS 和 Windows 原生应用。 |
| `assets/previews/` | 各应用共享的规范波形预览 WAV 文件。 |
| `docs/` | API 契约、仓库结构说明、许可证审计和评审报告。 |
| `deploy/production/` | Vue 生产路径的非 Docker 生产部署说明、辅助脚本和 Caddy 示例。 |
| `deploy/web-flask/` | 旧 Flask 回退路径的 Docker 部署文档和 Dockerfile。 |
| `scripts/` | 显式本地维护和 fixture 重新生成脚本。 |
| `compose.web.yml` | 旧 Flask 回退路径的 Docker Compose 入口。 |
| `global.json` | 保留的 Windows 解决方案使用的 .NET SDK 选择。 |
| `.dockerignore`, `.gitignore`, `.gitattributes` | 仓库打包、忽略和换行规则。 |
| `output/`, `tmp/` | 已跟踪的历史生成评审产物；两个路径都被忽略，用于未来生成输出。 |

`.venv/`、构建输出、`.codex/`、`.sisyphus/`、`.DS_Store`、`__pycache__/`、
`.xcodebuild/` 和各应用的 `build/` 目录等本地目录不属于维护中的源码结构。

### `backend/`

主 Go 后端运行时和迁移验证模块。它通过冻结的 Python baseline fixtures 满足 OpenAPI 契约，同时让常规运行时不再依赖 Python。

- `go.mod`：迁移后端的 Go module。
- `cmd/server/`：进程启动、环境配置、SQLite workspace store 打开、日志、graceful shutdown 和 HTTP server wiring。
- `internal/config/`：与当前 Flask 兼容的环境变量解析和默认值。
- `internal/httpapi/`：health、workspace state/upload/queue/config API、workspace-backed `/api/synthesis-jobs` JSON 和 multipart render/poll/download/delete flows、legacy `/synthesise` 以及 `/synthesise/jobs` render/poll/download/delete routes，以及 previews 的初始 OpenAPI 形状 HTTP routes、route tests、OpenAPI route-registration conformance，以及当前 frozen fixtures 覆盖路由的 Python-baseline replay tests。
- `internal/jobs/`：bounded render execution、legacy job file storage，以及与 workspace-backed job flow 共享 queue 语义的内存 render job lifecycle tests。
- `internal/midi/`：使用 `gitlab.com/gomidi/midi/v2/smf` 提取 Standard MIDI File 音符，并通过 PrettyMIDI-derived baseline fixture 对齐确认。
- `internal/renderer/`：与 Python baseline 对齐的渲染限制、声音层校验、频率曲线插值、曲线 hash、输出命名和 note-event PCM/WAV synthesis。
- `internal/storage/`：与 Flask storage model 对齐的 SQLite schema、connection pragmas、token hashing、路径 helpers 和 workspace cleanup/cascade 行为。
- `internal/workspace/`：与 Python baseline 对齐的工作区 token lifecycle、state payloads、upload queue 操作、limits、配置持久化、SQLite-backed synthesis jobs、WAV output cleanup 和 renderer form payload 转换。
- `testdata/python-baseline/`：规范化 API transcripts、代表性 MIDI 输入、期望 WAV
  输出、parsed note-event fixtures、渲染器命名/hash 预期，以及工作区配置规范化案例。重新生成必须显式运行
  `scripts/generate_python_parity_fixtures.py`；常规测试直接读取这些文件，不应要求 Python。

## 应用目标

### `frontend/`

公开浏览器体验的生产 Vue/Vite 前端。

- `index.html`：Vite 应用外壳。
- `src/App.vue`：顶层 Vue 工作流和状态编排。
- `src/api/`：后端 `/api/*` 路由的类型化客户端。
- `src/components/`：上传队列、声音层编辑器、输出控制、头部控制、已转换文件和曲线编辑器组件。
- `src/i18n/`：英文、法文和简体中文前端 catalog。
- `src/styles/app.css`：从 Flask UI 复用的当前 OctaBit 视觉系统。
- `vite.config.ts`：开发环境中把 `/api` 和 `/static/previews` 代理到
  `http://127.0.0.1:8000`。
- `package.json` 和 `package-lock.json`：Vue/Vite 依赖元数据。

生产 Caddy 提供 `frontend/dist`，并把 API、预览资源和兼容合成请求代理到 Go 后端。

### `legacy/web-flask/`

旧 Flask 后端 API、工作区/合成服务、预览路由提供者，以及保留作为 parity fixture 重新生成和回退参考的旧 Flask 渲染前端回退。

- `app.py`：Flask 入口、上传处理、合成/API 端点、预览路由和服务器端渲染任务端点。
- `synthesis_jobs.py`：基于文件系统的合成任务生命周期、清理和渲染线程编排。
- `templates/index.html`：浏览器 UI 外壳。
- `static/css/` 和 `static/js/`：Web 专用样式和浏览器行为。
- `i18n/`：英文、法文和简体中文 UI 文本的 JSON 目录。
- `tests/`：Flask 和渲染路径测试。
- `requirements.txt`：Web 运行时依赖；它包含共享渲染器依赖。
- `Launch_Synthesiser.command` 和 `Launch_Synthesiser.bat`：本地启动器。
- `README.md`、`README.zh-CN.md`、`User_Guide.txt`：Web 应用文档。

Flask 后端将合成交给 `legacy/python-renderer/midi_to_wave.py`，并从
`assets/previews/` 提供预览音频；常规生产运行时使用 Go 后端。

### `legacy/native/macos/`

已 deprecated/paused 的原生 SwiftUI macOS 应用和 Xcode 工程。该代码不是主要开发目标；在项目聚焦
Web 服务期间，它保留用于参考或未来可能的恢复。

- `MIDI8BitSynthesiser.xcodeproj/`：Xcode 工程和共享 scheme。
- `MIDI8BitSynthesiser/`：SwiftUI 应用源码。
- `MIDI8BitSynthesiserTests/`：用于模型和文件名逻辑的 XCTest 目标。
- `macos/build_desktop_resources.sh`：Xcode 构建阶段脚本，用于将 Python 渲染器冻结为辅助二进制文件，并把预览 WAV 资源复制进应用包。
- `requirements-build.txt`：辅助程序的 Python 构建依赖。
- `macos/README.md`、`macos/README.zh-CN.md`：macOS 构建和使用说明。

macOS 应用不运行 Flask 服务器。它会为每个队列中的 MIDI 文件启动随包附带的 Python
辅助程序。

### `legacy/native/windows/`

已 deprecated/paused 的原生 WinUI 3 Windows 应用、C# 渲染器、测试、安装程序和评审工具。该代码不是主要开发目标；在项目聚焦
Web 服务期间，它保留用于参考或未来可能的恢复。

- `Midi8BitSynthesiser.sln`：Windows 解决方案。
- `Directory.Packages.props`：集中管理的 NuGet 包版本。
- `src/Midi8BitSynthesiser.Core/`：C# 渲染引擎、波形模型和输出命名。
- `src/Midi8BitSynthesiser.App/`：WinUI 3 外壳、兼容性检查、文件对话框服务、预览播放、本地化资源和应用清单。
- `tests/Midi8BitSynthesiser.Tests/`：单元、工作流、兼容性和 Python 对齐测试。
- `installer/Midi8BitSynthesiser.iss`：Inno Setup 安装程序脚本。
- `installer/RuntimeNotice.txt`：安装前运行时提示。
- `scripts/create_review_bundle.sh`：准备 Windows 评审包的脚本。
- `README.md`、`README.zh-CN.md`、`REVIEWING.md`：Windows 构建和评审文档。

保留的 Windows 应用有自己的 C# 渲染器，并在对齐测试中用 Python 参考渲染器进行校验。应用工程会从规范
`assets/previews/` 目录链接预览 WAV 文件，用于构建和发布输出。
`src/Midi8BitSynthesiser.App/Assets/Previews/` 下也存在一份字节相同的已跟踪副本，但工程文件使用共享资源目录作为构建来源。

## 共享核心和资源

### `legacy/python-renderer/`

规范 Python MIDI 转 WAV 渲染器。

- `midi_to_wave.py`：渲染器模块和 CLI 入口。
- `requirements.txt`：仅包含渲染器/运行时依赖。
- `tests/`：渲染器测试。
- `README.md`：渲染器接口、层结构和依赖边界。

渲染器接收平台无关的文件路径和波形层设置，然后将 WAV 文件写入磁盘。旧 Flask 后端会直接调用它；保留的
macOS 应用也会直接调用它，保留的 Windows 应用将它作为原生 C# 渲染器的对齐参考。

### `assets/previews/`

Web 前端/后端路径和保留的原生应用路径使用的规范预览 WAV 资源。`assets/README.md`
记录了它们的预期用途和来源说明。

## 文档和生成产物

- `docs/api-contract.md`、`docs/api-contract.zh-CN.md` 和 `docs/openapi.yaml`：当前 Web API 契约、兼容路由说明、任务载荷和公开演示安全边界。
- `docs/repository-layout.md` 和 `docs/repository-layout.zh-CN.md`：当前仓库结构的英文和简体中文说明。
- `docs/licensing-audit.md`：面向仓库和发布规划的许可证与署名审计。
- `docs/reviews/windows-app-review.md`：Windows 评审记录。
- `output/pdf/repo-structure-evaluation.pdf`、
  `tmp/pdfs/repo-structure-evaluation.html` 和
  `tmp/pdfs/rendered/repo-structure-evaluation.png`：已跟踪的历史生成评审产物。它们不是当前结构的事实来源。

## 构建和开发流程

除非某个文档另有说明，否则从仓库根目录运行命令。

创建本地 Python 环境：

```bash
python3 -m venv .venv
```

只安装当前工作区域需要的依赖：

```bash
./.venv/bin/python3 -m pip install -r legacy/web-flask/requirements.txt
./.venv/bin/python3 -m pip install -r legacy/native/macos/requirements-build.txt
./.venv/bin/python3 -m pip install -r legacy/python-renderer/requirements.txt
```

常规检查：

```bash
cd backend && go test ./...
cd frontend && npm run build
```

旧 Python 检查用于回退路径或 parity 参考代码变更：

```bash
./.venv/bin/python3 -m unittest discover -s legacy/web-flask/tests
./.venv/bin/python3 -m unittest discover -s legacy/python-renderer/tests
```

已暂停的 Windows 应用仍可通过 .NET 8 和 Python 渲染器依赖进行检查：

```powershell
dotnet restore legacy/native/windows/Midi8BitSynthesiser.sln
dotnet build legacy/native/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64
dotnet test legacy/native/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build
```

仓库中已不再保留受维护的 Windows 发布工作流或 CI 发布流水线。若未来恢复原生
Windows 打包工作，应基于当前工程文件重新建立发布步骤，而不是依赖已经移除的工作流。

已暂停的 macOS 应用通过 Xcode 使用 `MIDI8BitSynthesiser` scheme 构建。Xcode 构建阶段会运行
`legacy/native/macos/macos/build_desktop_resources.sh`。

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
`frontend/dist`，并将 `/api/*`、`/static/previews/*` 和 `/synthesise*` 反向代理到该私有监听地址。工作区/任务目录、任务 TTL、最大上传大小和渲染 worker 设置应与当前合成任务行为保持一致。Caddy 生产和回滚示例见 `deploy/production/README.zh-CN.md`。

Docker 部署仍保留为旧 Flask 回退的另一种路径：

```bash
docker compose -f compose.web.yml up -d --build
```

Compose 文件将服务绑定到 `127.0.0.1:8000`，用于先通过隧道测试；镜像中只构建旧 Flask
后端/回退、共享渲染器、共享预览资源和项目许可证。

## 依赖和打包边界

- Python 渲染器依赖位于 `legacy/python-renderer/requirements.txt`。
- Go 后端依赖位于 `backend/go.mod` 和 `backend/go.sum`。
- 旧 Web 专用 Python 依赖位于 `legacy/web-flask/requirements.txt`。
- 生产前端 JavaScript 依赖位于 `frontend/package.json` 和
  `frontend/package-lock.json`。
- macOS 辅助构建依赖位于 `legacy/native/macos/requirements-build.txt`。
- Windows NuGet 版本位于 `legacy/native/windows/Directory.Packages.props`。
- Docker 部署文件仅限旧 Flask 回退路径。
- 保留的原生应用打包仍位于对应应用目录下。

## 归属边界

- 运行时渲染行为属于 `backend/internal/renderer/`。
- Python parity 参考行为属于 `legacy/python-renderer/`。
- 生产 Web UI 属于 `frontend/`。
- 旧 Flask 后端 API 和旧 Flask 渲染前端回退逻辑属于 `legacy/web-flask/`。
- 保留的原生 UI、启动和打包逻辑仍位于对应的 `legacy/native/` 目录下。
- 共享二进制/媒体资源属于 `assets/`。
- 仓库级文档、审计和评审记录属于 `docs/`。
- 部署专用文件属于 `deploy/` 和根目录部署入口，例如 `compose.web.yml`。
