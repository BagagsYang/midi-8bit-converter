# 为 OctaBit 做贡献

Language/语言: [English](./CONTRIBUTING.md) | 简体中文

感谢你帮助改进 OctaBit。本文说明贡献应放在仓库的哪些区域、什么情况应先开 issue，以及 pull request 中应包含哪些信息。

## 先确认合适的区域

OctaBit 是一个单体仓库。当前活跃的贡献目标是：

- `frontend/`：生产 Vue 浏览器前端。
- `backend/`：主 Go 后端 API、工作区/合成服务和 Go 渲染器。
- `legacy/web-flask/`：保留作为 parity 参考的旧 Flask 后端/API 和 Flask 渲染前端回退。
- `legacy/python-renderer/`：保留作为 parity 参考的规范 Python MIDI 转 WAV 渲染器。
- `docs/`、`deploy/production/`、`deploy/web-flask/` 和 `assets/previews/`：配套文档、部署和共享资源区域。

`legacy/native/macos/` 和 `legacy/native/windows/` 下的原生 macOS 与 Windows 应用是暂停/参考区域。若要在这些区域做较大的工作，请先开 issue，让维护者确认范围。

更完整的仓库结构请参阅 [docs/repository-layout.zh-CN.md](./docs/repository-layout.zh-CN.md)。

## 开始之前

小型文档修正和范围很窄的 bug 修复可以直接提交 pull request。

以下较大变更请先开 issue：

- 公开 Web UI 或工作流变更；
- 渲染器行为、渲染器 schema 或输出命名变更；
- 部署、Docker 或服务器运行时变更；
- 架构、仓库结构或依赖变更；
- 涉及许可证的新工作、vendored 资源、生成媒体或新的第三方材料；
- macOS 或 Windows 应用变更。

在 issue 中说明问题、期望行为，以及你预计会改动的仓库区域。

## 分支工作流

OctaBit 使用简单的长期分支工作流：

- `main` 是稳定、可部署的分支，供线上服务器使用。请保持它处于可部署状态。
- `dev` 是活跃开发分支，用于持续开发和较大变更。
- 日常开发应在 `dev` 上进行，或使用基于 `dev` 的短期功能分支。
- 较大变更只有在 `dev` 或功能分支上经过 review 和测试后，才应合并到 `main`。

除非维护者明确同意某个具体变更需要额外流程，否则避免长期 release 分支或重量级流程。

## 开发环境

除非文档另有说明，请在仓库根目录运行命令。

创建本地 Python 环境：

```bash
python3 -m venv .venv
```

只安装你当前处理区域所需的依赖：

```bash
./.venv/bin/python3 -m pip install -r legacy/web-flask/requirements.txt
./.venv/bin/python3 -m pip install -r legacy/python-renderer/requirements.txt
```

各区域说明可先阅读：

- [frontend/README.md](./frontend/README.md)
- [backend/README.md](./backend/README.md)
- [legacy/web-flask/README.zh-CN.md](./legacy/web-flask/README.zh-CN.md)
- [legacy/python-renderer/README.zh-CN.md](./legacy/python-renderer/README.zh-CN.md)
- [deploy/production/README.zh-CN.md](./deploy/production/README.zh-CN.md)

## 修改代码或文档

- 将 Vue 应用视为生产公开前端。
- 将 Go 后端作为主要 API 和合成服务。
- 共享合成行为应保留在 `backend/internal/renderer/`，parity 参考保留在 `legacy/python-renderer/`。
- 不要为了本地化复制应用源码树。请使用被修改平台已有的本地化资源。
- 对 `frontend/`，优先使用 `src/i18n/*.json` 保存面向用户的 UI 字符串。
- 对 `legacy/web-flask/` 中的旧 Flask 渲染 UI，优先使用 `i18n/*.json` 加独立静态
  JS/CSS，避免在模板中加入大量内联脚本或硬编码面向用户的字符串。
- 修改成对文档时，请保持英文和简体中文版本一致。
- 在功能或 bug 修复 pull request 中避免无关重构。

## Pull request 检查清单

提交 pull request 前，请确认其中包含：

- 对问题和改动的清晰总结；
- 主要改动的仓库区域；
- 对用户可见行为、UI、部署或兼容性的影响；
- 可见 Web UI 变更的截图或简短说明；
- 已运行的检查，以及相关但无法运行的检查；
- 新依赖、vendored 资源、生成媒体或其他第三方材料的来源和许可证说明。

## 验证

请运行与你改动区域相关的检查，并在 pull request 中报告结果。

Go 后端：

```bash
cd backend && go test ./...
```

Vue 前端：

```bash
cd frontend && npm run build
```

旧 Python 代码：

```bash
./.venv/bin/python3 -m unittest discover -s legacy/web-flask/tests
./.venv/bin/python3 -m unittest discover -s legacy/python-renderer/tests
```

仅修改文档时，请校对受影响文件；如果同时存在英文和中文版本，请保持两者一致。

修改部署内容时，请包含对已改部署文件的静态检查，以及可运行的 Docker 或 Compose 验证。

修改原生应用时，如果你的机器具备所需工具，请运行相关应用 README 中记录的检查。若由于缺少 Xcode、.NET、Docker 或其他依赖而无法运行某项检查，请在 pull request 中说明。

## 许可证与来源

OctaBit 采用 GNU Affero General Public License v3.0 或更新版本（`AGPL-3.0-or-later`）授权。请参阅 [LICENSE.md](./LICENSE.md)。

提交贡献即表示你确认自己有权提交相应代码、文档、资源和其他材料，并且这些内容与仓库许可证兼容。对于新依赖、vendored 资源、生成媒体或其他许可证敏感文件，请在 pull request 中说明来源和许可证信息。
