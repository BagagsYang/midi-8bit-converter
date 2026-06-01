# Web Flask 应用

Language/语言: [English](./README.md) | 简体中文

此目录包含 OctaBit 的旧 Flask 后端和旧 Flask 渲染前端回退。公开生产前端现在是
`../../frontend/` 中的 Vue 应用，并从其 Vite `dist` 构建产物提供服务；常规生产 API 流量由 `../../backend/` 中的 Go 后端处理。

## 职责

- Flask 入口点与请求处理
- 工作区、上传、合成、下载和预览资源的后端 API 路由
- 供旧 Flask 渲染前端回退使用的 HTML 模板与 Web 专用静态资源
- 启动脚本
- 将合成工作委托给 `../python-renderer/` 中的 Python 渲染器

## 运行

在仓库根目录执行：

```bash
python3 -m venv .venv
./.venv/bin/python3 -m pip install -r legacy/web-flask/requirements.txt
PORT=8000 WEB_FLASK_OPEN_BROWSER=0 ./.venv/bin/python3 legacy/web-flask/app.py
```

或者运行同时启动 Vue 前端和 Flask 后端的辅助脚本：

```bash
legacy/web-flask/Launch_Synthesiser.command
```

在 Windows 上，请使用：

```bat
legacy\web-flask\Launch_Synthesiser.bat
```

辅助脚本要求先用 `cd frontend && npm ci` 安装一次 Vue 依赖，然后会打开
`http://127.0.0.1:5173` 上的 Vue dev server。

## 共享依赖

- 渲染器：`../python-renderer/midi_to_wave.py`
- 规范预览资源：`../../assets/previews/`

此应用从共享资源目录提供预览 WAV 文件，不应复制渲染器逻辑。

## 当前 API 契约

生产 Vue 前端和旧 Flask 渲染回退都使用基于 cookie 的匿名临时工作区。上传的 MIDI 文件、声音配置和已转换
WAV 链接会在刷新后通过 `/api/workspace` 恢复。完整契约位于
`../../docs/api-contract.zh-Hans.md`。

当前 API 路由：

- `GET /api/health`：轻量 JSON 健康检查。
- `GET /api/workspace`：创建或恢复匿名临时工作区。
- `POST /api/workspace/uploads`：存储一个排队的 `.mid` 或 `.midi` 上传文件。
- `DELETE /api/workspace/uploads/<file_id>`：删除归属当前工作区的排队上传文件。
- `PATCH /api/workspace/queue`：持久化队列顺序。
- `PUT /api/workspace/config`：持久化采样率和层控制。
- `POST /api/synthesis-jobs`：接受归属当前工作区的 `file_id` 和工作区配置，并返回任务 id；旧的 multipart API 形状仍保留用于兼容。
- `GET /api/synthesis-jobs/<job_id>`：报告归属任务的 queued、rendering、ready、failed 或 expired 状态。
- `GET /api/synthesis-jobs/<job_id>/download`：下载已准备好的 WAV 文件。
- `DELETE /api/synthesis-jobs/<job_id>`：当用户清空已转换文件列表时删除归属当前工作区的服务器临时文件。

兼容路由：

- `POST /synthesise`：单请求上传、渲染和下载路径。
- `POST /synthesise/jobs`：接受相同表单字段并返回任务 id。
- `GET /synthesise/jobs/<job_id>`：报告 queued、rendering、ready、failed 或 expired 状态。
- `GET /synthesise/jobs/<job_id>/download`：下载已准备好的 WAV 文件。
- `DELETE /synthesise/jobs/<job_id>`：当用户清空已转换文件列表时删除服务器上的临时文件。

工作区文件是临时文件，会在 `WEB_WORKSPACE_TTL_SECONDS` 后过期；默认值为
86400 秒。浏览器会在用户清空队列或已转换文件列表时立即删除对应的服务器文件。

API 错误使用 `{"error":{"code":"...","message":"..."}}`。兼容路由继续保留现有
`{"error":"..."}` 响应形状，除非后续明确迁移。

## 生产部署说明

此旧回退路径仍可不使用 Docker 运行：

- 将 `legacy/web-flask/requirements.txt` 安装到仓库本地虚拟环境。
- 从 `legacy/web-flask/` 运行指向 `app:app` 的 Gunicorn。
- 将 Gunicorn 私有绑定到 `127.0.0.1:8000`。
- 使用 systemd 管理 Gunicorn，例如通过 `octabit-web` 服务。
- 在 `../../frontend/` 中用 `npm ci && npm run build` 构建 Vue 前端。
- 使用 Caddy 公开提供 `../../frontend/dist`，并将 `/api/*`、`/static/previews/*`
  和 `/synthesise*` 反向代理到 `127.0.0.1:8000`。
- 让 `WEB_SYNTHESISE_JOB_ROOT`、`WEB_WORKSPACE_TTL_SECONDS`、
  `WEB_WORKSPACE_MAX_QUEUED_FILES`、`WEB_WORKSPACE_MAX_UPLOAD_BYTES`、
  `WEB_WORKSPACE_MAX_CONVERTED_FILES`、`WEB_DOWNLOAD_TTL_SECONDS`、
  `WEB_MAX_UPLOAD_BYTES` 和 Gunicorn timeout 与预期的上传、渲染、下载行为保持一致。

Gunicorn 命令形态示例：

```bash
./.venv/bin/python3 -m gunicorn --chdir legacy/web-flask --bind 127.0.0.1:8000 --workers 2 --timeout 600 app:app
```

`../../deploy/web-flask/` 下的 Docker 文件和 `../../compose.web.yml`
仍保留为旧 Flask 渲染前端回退的另一种部署路径。当前 Go 生产路径记录在
`../../deploy/production/README.zh-Hans.md`。

## 输出命名

- 单个可听层且无曲线：`<original>_<wave>.wav`
- 多个可听层且无曲线：`<original>_mix.wav`
- 任一可听层带有非空频率曲线：`<original>_<base>_<hash>.wav`

哈希值基于清理后的层载荷生成，因此不同曲线设置不会复用同一个导出名称。
