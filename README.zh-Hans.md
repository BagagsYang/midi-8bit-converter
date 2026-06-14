# OctaBit

Language/语言: [English](./README.md) | 简体中文

OctaBit 是一个基于浏览器的工具，用于将 MIDI 文件转换为 8-bit 风格 WAV 音频。公开服务地址是 <https://octabit.cc>。

OctaBit 的公开 OSS 代码库由私有上游 monorepo 镜像到 `bagags/octabit`。公开镜像用于让任何人阅读、审计、运行和自行部署 AGPL 授权的 OSS 代码，但它不是开放贡献目标，也不接受未经邀请的 pull request。提交 issue 或事先约定贡献工作前，请先阅读 [CONTRIBUTING.zh-Hans.md](./CONTRIBUTING.zh-Hans.md)。

## 归档说明

此公开仓库将作为 OctaBit 的历史 OSS 镜像归档，不再是主要开发仓库。

OctaBit 和 OctaBit Pro 的后续开发将在私有的专有 `bagags/octabit-pro`
代码库中继续。已经以 GNU AGPL-3.0-or-later 发布的历史公开版本和源码快照，
仍按发布时附带的 AGPL 条款提供。此归档说明不会撤销历史公开版本已经授予的权利。

第三方依赖、图标、字体、资源和 fixtures 仍受其各自许可证条款约束。当前通知清单见
[THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)。

生产 Web 前端是 `frontend/` 中的 Vue 3 应用。主后端是 `backend/` 中的 Go 服务，它实现稳定的 `/api/*` 合约、工作区存储、合成任务和 Go MIDI-to-WAV 渲染器。

## 当前活跃内容

| 路径 | 作用 |
| --- | --- |
| `frontend/` | 从 Vite `dist` 构建产物提供服务的生产 Vue 3 前端 |
| `backend/` | 主 Go 后端 API、工作区/合成服务、Go 渲染器和冻结 Python 对齐 fixtures |
| `assets/previews/` | 通过后端提供的共享波形预览 WAV 文件 |
| `deploy/production/` | Vue 生产路径的非 Docker 生产部署说明、辅助脚本和 Caddy 示例 |
| `docs/api-contract.md`、`docs/openapi.yaml` | Web API 请求和响应契约 |

## 运行 Web 应用

在仓库根目录执行：

```bash
cd backend
PORT=8000 go run ./cmd/server
```

在另一个终端运行 Vue 前端：

```bash
cd frontend
npm ci
npm run dev
```

打开 `http://127.0.0.1:5173/`。Vite 会把 `/api/*` 和 `/static/previews/*` 代理到
`127.0.0.1:8000` 上的 Go 后端。

运行常规检查：

```bash
cd backend && go test ./...
cd frontend && npm run build
```

## 用户限制

以下是当前 Web 应用和渲染器的默认限制。部署者可以通过环境变量调整 Web 服务限制，渲染器安全限制由 Go 渲染器强制执行。

| 限制 | 默认值 | 来源 |
| --- | ---: | --- |
| 单次请求上传大小 | 20 MiB | `WEB_MAX_UPLOAD_BYTES` |
| 工作区最后活动后的保留时间 | 86400 秒 | `WEB_WORKSPACE_TTL_SECONDS` |
| 每个工作区排队 MIDI 文件数 | 20 个文件 | `WEB_WORKSPACE_MAX_QUEUED_FILES` |
| 每个工作区排队上传总存储 | 100 MiB | `WEB_WORKSPACE_MAX_UPLOAD_BYTES` |
| 每个工作区已转换 WAV 文件数 | 20 个文件 | `WEB_WORKSPACE_MAX_CONVERTED_FILES` |
| 每个容器活跃渲染工作线程 | 2 个线程 | `WEB_RENDER_WORKERS` |
| 每个容器等待渲染队列 | 8 个任务 | `WEB_RENDER_QUEUE_SIZE` |
| MIDI 时长 | 1800 秒 | 渲染器限制 |
| 渲染样本数 | 172800000 个样本 | 渲染器限制 |
| WAV 样本数据大小 | 345600000 字节，约 329.6 MiB | 渲染器限制 |
| MIDI 音符数 | 20000 个音符 | 渲染器限制 |
| 声音层数 | 4 层 | 渲染器限制和 Web 配置 |
| 每层频率曲线点数 | 8 个点 | 渲染器限制 |
| 采样率 | 44100、48000 或 96000 Hz | Web 校验 |
| Pulse 占空比 | 0.01 到 0.99 | 渲染器校验 |
| Web 层音量 | 0.0 到 2.0 | 工作区配置校验 |
| 频率曲线增益 | -36 dB 到 12 dB | 渲染器校验 |
| 频率曲线范围 | MIDI 音符 0 到 127 对应频率 | 渲染器校验 |

排队上传和已转换 WAV 文件都是临时文件。用户在浏览器中清空排队文件或已转换文件时，Web 应用会请求服务器立即删除对应的临时文件。

## Web API

Vue 前端通过 Go API 使用基于 cookie 的匿名临时工作区。`GET /api/workspace` 会创建或恢复工作区，资源路由要求携带当前工作区 cookie。可读 API 契约位于 `docs/api-contract.md`，机器可读 OpenAPI 契约位于 `docs/openapi.yaml`。

主要路由：

- `GET /api/health`
- `GET /api/workspace`
- `POST /api/workspace/uploads`
- `DELETE /api/workspace/uploads/<file_id>`
- `PATCH /api/workspace/queue`
- `PUT /api/workspace/config`
- `POST /api/synthesis-jobs`
- `GET /api/synthesis-jobs/<job_id>`
- `GET /api/synthesis-jobs/<job_id>/download`
- `DELETE /api/synthesis-jobs/<job_id>`

API 错误使用 `{"error":{"code":"...","message":"..."}}`。

## 声音配置

Web 应用会在临时工作区中保存采样率和声音层设置。合成支持 pulse、sine、sawtooth 和 triangle 层。频率-增益曲线由共享渲染器校验，并在合成时按层应用。

输出命名：

- 单个可听层且没有曲线：`<original>_<wave>.wav`
- 多个可听层且没有曲线：`<original>_mix.wav`
- 任一可听层带有非空频率曲线：`<original>_<base>_<hash>.wav`

哈希来自经过清理的层配置，因此不同曲线设置不会复用同一个导出名称。

## 本地化

生产 Vue UI 使用 `frontend/src/i18n/` 中的 JSON catalog 文件，覆盖英文、西班牙文、法文、日文、简体中文（`zh-Hans`）和繁体中文（`zh-Hant`）。修改生产前端 catalog 时，请保持 `en.json`、`es.json`、`fr.json`、`ja.json`、`zh-Hans.json` 和 `zh-Hant.json` 的键集合一致。英文是回退语言。仓库文档仍保持英文和简体中文两种语言。面向 agent 的标准流程见 [docs/localisation.md](./docs/localisation.md)。

面向用户的 Web 字符串应进入 catalog，不应硬编码在模板或 JavaScript 中。

## 部署

预期生产模型不使用 Docker：

```bash
cd backend && go build -o octabit-server ./cmd/server
PORT=8000 WEB_SYNTHESISE_JOB_ROOT=/var/lib/octabit ./octabit-server
cd frontend && npm ci && npm run build
```

公开部署时，应让 Go 后端私有监听 `127.0.0.1:8000`。Caddy 将 `frontend/dist` 作为公开前端，并把 `/api/*` 和 `/static/previews/*` 反向代理到 Go。生产部署说明、Caddy 示例、smoke 检查和回滚步骤位于 `deploy/production/README.zh-Hans.md`。

## 许可证

本项目的历史公开版本采用 GNU Affero General Public License v3.0 或更新版本（`AGPL-3.0-or-later`）授权。详情见
[LICENSE.md](./LICENSE.md)。未来私有开发可能对项目自有代码采用单独的专有许可证；第三方组件仍按其各自许可证条款授权。
