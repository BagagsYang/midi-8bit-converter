# 仓库结构

Language/语言: [English](./repository-layout.md) | 简体中文

该仓库是 OctaBit 的单体仓库。OctaBit 是一个用于将 MIDI 文件转换为 8-bit 风格音乐的简单 Web 工具。当前生产
Web 前端是 `apps/web-vue/` 中的 Vue 应用，并从 Vite `dist` 构建产物为 `octabit.cc`
提供服务。`apps/web-flask/` 中的 Flask/Gunicorn 继续作为私有后端 API 和工作区/合成服务；其服务器端渲染前端保留为
legacy fallback。原生 macOS 和 Windows 应用已 deprecated/paused，不再作为活跃开发目标；代码保留用于参考或未来可能的恢复。仓库还包含规范
Python 渲染器、共享预览资源、API 契约文档、部署文件和参考文档。

## 顶层结构

| 路径 | 用途 |
| --- | --- |
| `AGENTS.md` | 面向编码代理和本地工作流的仓库说明。 |
| `README.md`, `README.zh-CN.md` | 根项目概览、设置说明、应用入口和仓库许可证摘要。 |
| `LICENSE.md` | 仓库 AGPL 许可证文本。 |
| `apps/` | 生产 Vue 前端、Flask 后端/回退、保留的原生应用代码和桌面占位目录。 |
| `core/python-renderer/` | 规范 Python MIDI 转 WAV 渲染器和对齐参考实现。 |
| `assets/previews/` | 各应用共享的规范波形预览 WAV 文件。 |
| `docs/` | API 契约、仓库结构说明、许可证审计和评审报告。 |
| `deploy/production/` | Vue 生产路径的非 Docker 生产部署说明、辅助脚本和 Caddy 示例。 |
| `deploy/web-flask/` | Flask 后端或旧前端回退路径的 Docker 部署文档和 Dockerfile。 |
| `compose.web.yml` | Flask 后端或旧前端回退路径的 Docker Compose 入口。 |
| `global.json` | 保留的 Windows 解决方案使用的 .NET SDK 选择。 |
| `.dockerignore`, `.gitignore`, `.gitattributes` | 仓库打包、忽略和换行规则。 |
| `output/`, `tmp/` | 已跟踪的历史生成评审产物；两个路径都被忽略，用于未来生成输出。 |

`.venv/`、构建输出、`.codex/`、`.sisyphus/`、`.DS_Store`、`__pycache__/`、
`.xcodebuild/` 和各应用的 `build/` 目录等本地目录不属于维护中的源码结构。

## 应用目标

### `apps/web-vue/`

公开浏览器体验的生产 Vue/Vite 前端。

- `index.html`：Vite 应用外壳。
- `src/App.vue`：顶层 Vue 工作流和状态编排。
- `src/api/`：Flask `/api/*` 路由的类型化客户端。
- `src/components/`：上传队列、声音层编辑器、输出控制、头部控制、已转换文件和曲线编辑器组件。
- `src/i18n/`：英文、法文和简体中文前端 catalog。
- `src/styles/app.css`：从 Flask UI 复用的当前 OctaBit 视觉系统。
- `vite.config.ts`：开发环境中把 `/api` 和 `/static/previews` 代理到
  `http://127.0.0.1:8000`。
- `package.json` 和 `package-lock.json`：Vue/Vite 依赖元数据。

生产 Caddy 提供 `apps/web-vue/dist`，并把 API 和预览资源请求代理到 Flask/Gunicorn。

### `apps/web-flask/`

Flask 后端 API、工作区/合成服务、预览路由提供者，以及旧 Flask 渲染前端回退。

- `app.py`：Flask 入口、上传处理、合成/API 端点、预览路由和服务器端渲染任务端点。
- `synthesis_jobs.py`：基于文件系统的合成任务生命周期、清理和渲染线程编排。
- `templates/index.html`：浏览器 UI 外壳。
- `static/css/` 和 `static/js/`：Web 专用样式和浏览器行为。
- `i18n/`：英文、法文和简体中文 UI 文本的 JSON 目录。
- `tests/`：Flask 和渲染路径测试。
- `requirements.txt`：Web 运行时依赖；它包含共享渲染器依赖。
- `Launch_Synthesiser.command` 和 `Launch_Synthesiser.bat`：本地启动器。
- `README.md`、`README.zh-CN.md`、`User_Guide.txt`：Web 应用文档。

Flask 后端将合成交给 `core/python-renderer/midi_to_wave.py`，并从
`assets/previews/` 提供预览音频。

### `apps/macos/`

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

### `apps/windows/`

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

### `apps/desktop/`

为未来桌面打包层保留的占位目录。它只包含 README 文件，没有应用实现。

## 共享核心和资源

### `core/python-renderer/`

规范 Python MIDI 转 WAV 渲染器。

- `midi_to_wave.py`：渲染器模块和 CLI 入口。
- `requirements.txt`：仅包含渲染器/运行时依赖。
- `tests/`：渲染器测试。
- `README.md`：渲染器接口、层结构和依赖边界。

渲染器接收平台无关的文件路径和波形层设置，然后将 WAV 文件写入磁盘。Flask 后端会直接调用它；保留的
macOS 应用也会直接调用它，保留的 Windows 应用将它作为原生 C# 渲染器的对齐参考。

### `assets/previews/`

Web 前端/后端路径和保留的原生应用路径使用的规范预览 WAV 资源。`assets/README.md`
记录了它们的预期用途和来源说明。

## 文档和生成产物

- `docs/api-contract.md` 和 `docs/api-contract.zh-CN.md`：当前 Web API 契约、兼容路由说明、任务载荷和公开演示安全边界。
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
./.venv/bin/python3 -m pip install -r apps/web-flask/requirements.txt
./.venv/bin/python3 -m pip install -r apps/macos/requirements-build.txt
./.venv/bin/python3 -m pip install -r core/python-renderer/requirements.txt
```

常用检查：

```bash
./.venv/bin/python3 -m unittest discover -s apps/web-flask/tests
./.venv/bin/python3 -m unittest discover -s core/python-renderer/tests
```

已暂停的 Windows 应用仍可通过 .NET 8 和 Python 渲染器依赖进行检查：

```powershell
dotnet restore apps/windows/Midi8BitSynthesiser.sln
dotnet build apps/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64
dotnet test apps/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build
```

仓库中已不再保留受维护的 Windows 发布工作流或 CI 发布流水线。若未来恢复原生
Windows 打包工作，应基于当前工程文件重新建立发布步骤，而不是依赖已经移除的工作流。

已暂停的 macOS 应用通过 Xcode 使用 `MIDI8BitSynthesiser` scheme 构建。Xcode 构建阶段会运行
`apps/macos/macos/build_desktop_resources.sh`。

Vue 开发时，先在 8000 端口运行 Flask 后端，再启动 Vite dev server：

```bash
PORT=8000 WEB_FLASK_OPEN_BROWSER=0 ./.venv/bin/python3 apps/web-flask/app.py
cd apps/web-vue
npm ci
npm run dev
```

Vue 生产构建：

```bash
cd apps/web-vue
npm ci
npm run build
```

当前非 Docker 生产路径从 Python 虚拟环境运行 Flask/Gunicorn：Gunicorn 私有绑定到
`127.0.0.1:8000`，systemd 管理服务，Caddy 提供 `apps/web-vue/dist`，并将
`/api/*`、`/static/previews/*` 和 `/synthesise*` 反向代理到该私有 Gunicorn
监听地址。上传目录、任务 TTL、最大上传大小和 Gunicorn timeout 应与当前合成任务行为保持一致。Caddy
生产和回滚示例见 `deploy/production/README.zh-CN.md`。

Docker 部署仍保留为 Flask 后端或旧前端回退的另一种路径：

```bash
docker compose -f compose.web.yml up -d --build
```

Compose 文件将服务绑定到 `127.0.0.1:8000`，用于先通过隧道测试；镜像中只构建 Flask
后端/回退、共享渲染器、共享预览资源和项目许可证。

## 依赖和打包边界

- Python 渲染器依赖位于 `core/python-renderer/requirements.txt`。
- Web 专用 Python 依赖位于 `apps/web-flask/requirements.txt`。
- 生产前端 JavaScript 依赖位于 `apps/web-vue/package.json` 和
  `apps/web-vue/package-lock.json`。
- macOS 辅助构建依赖位于 `apps/macos/requirements-build.txt`。
- Windows NuGet 版本位于 `apps/windows/Directory.Packages.props`。
- Docker 部署文件仅限 Flask 后端/回退路径。
- 保留的原生应用打包仍位于对应应用目录下。

## 归属边界

- 共享渲染行为属于 `core/python-renderer/`。
- 生产 Web UI 属于 `apps/web-vue/`。
- Flask 后端 API 和旧 Flask 渲染前端回退逻辑属于 `apps/web-flask/`。
- 保留的原生 UI、启动和打包逻辑仍位于对应的 `apps/` 目录下。
- 共享二进制/媒体资源属于 `assets/`。
- 仓库级文档、审计和评审记录属于 `docs/`。
- 部署专用文件属于 `deploy/` 和根目录部署入口，例如 `compose.web.yml`。
