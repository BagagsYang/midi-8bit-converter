# Web API 约定

Language/语言: [English](./api-contract.md) | 简体中文

本文档描述 Go OctaBit 后端面向浏览器的 API 边界。生产 Web 前端是
`frontend/` 中的 Vue 应用，由 Caddy 从 Vite `dist` 构建产物提供服务。Go 服务是私有后端
API、工作区/合成服务、预览提供方和运行时渲染器。机器可读契约位于
[`openapi.yaml`](./openapi.yaml)。

## 范围

- 生产前端代码应使用 `/api/*` 路由。
- 生产环境中，Caddy 应将 `/api/*` 和 `/static/previews/*`
  反向代理到 `127.0.0.1:8000` 上的 Go 后端。
- 匿名临时工作区使用名为 `octabit_workspace` 的随机 HttpOnly cookie。
- SQLite 存储工作区元数据。MIDI 和 WAV 二进制文件仍存储在文件系统中。
- 浏览器可见的 `file_id` 和 `job_id` 是随机、不透明的 UUID 十六进制标识。
  API 永不返回内部 SQLite 数字 id。
- 包含 `<file_id>` 或 `<job_id>` 的路由通过以下链路检查归属：
  cookie token -> token hash -> 活跃工作区行 -> 归属资源行。

## 配置

- `WEB_SYNTHESISE_JOB_ROOT`：工作区数据库、上传文件和 WAV 文件的存储根目录。默认值为系统临时目录加 `octabit-jobs`。
- `WEB_WORKSPACE_TTL_SECONDS`：匿名工作区在最后一次活动后的保留时间。默认值为 `86400`。
- `WEB_WORKSPACE_MAX_QUEUED_FILES`：每个工作区的 MIDI 队列文件数量上限。默认值为 `20`。
- `WEB_WORKSPACE_MAX_UPLOAD_BYTES`：每个工作区活跃 MIDI 上传总字节上限。默认值为 `104857600`。
- `WEB_WORKSPACE_MAX_CONVERTED_FILES`：每个工作区活跃已转换任务数量上限。默认值为 `20`。
- `WEB_MAX_UPLOAD_BYTES`：单请求上传大小上限。默认值为 `20971520`。
- 支持的上传扩展名：`.mid`、`.midi`。
- 支持的采样率：`44100`、`48000`、`96000`。

SQLite 每次请求或操作打开一个连接，启用 `PRAGMA foreign_keys=ON`，使用 WAL
journal mode，并设置 `PRAGMA busy_timeout=5000` 以减少多 worker 下的锁等待失败。

## 通用响应类型

`ApiError`：

```json
{
  "error": {
    "code": "invalid_layers",
    "message": "Layer 1 frequency_curve frequencies must be strictly increasing."
  }
}
```

`WorkspaceConfigV1`：

```json
{
  "schema": "octabit.workspace_config.v1",
  "sample_rate": 48000,
  "layers": [
    {
      "type": "pulse",
      "duty": 0.5,
      "volume": 1.0,
      "curve_enabled": false,
      "frequency_curve": [
        {"frequency_hz": 8.175798915643707, "gain_db": 0.0},
        {"frequency_hz": 12543.853951415975, "gain_db": 0.0}
      ]
    }
  ]
}
```

验证规则：

- `sample_rate`：`44100`、`48000` 或 `96000`
- `layers`：1 到 4 个层对象
- `type`：`pulse`、`sine`、`sawtooth` 或 `triangle`
- `duty`：`0.01` 到 `0.99`
- `volume`：`0.0` 到 `2.0`
- `curve_enabled`：布尔值
- `frequency_curve`：使用渲染器规则验证的频率/增益点；当 `curve_enabled` 为
  `false` 时，渲染会忽略它

`WorkspaceStateResponse`：

```json
{
  "workspace": {
    "expires_at": 1770000000.0
  },
  "limits": {
    "max_queued_files": 20,
    "max_upload_bytes": 104857600,
    "max_converted_files": 20
  },
  "config": {
    "schema": "octabit.workspace_config.v1",
    "sample_rate": 48000,
    "layers": [
      {"type": "pulse", "duty": 0.5, "volume": 1.0, "curve_enabled": false, "frequency_curve": []}
    ]
  },
  "uploads": [
    {
      "file_id": "0123456789abcdef0123456789abcdef",
      "name": "lead.mid",
      "size": 12345,
      "created_at": 1770000000.0
    }
  ],
  "converted_files": [
    {
      "job_id": "abcdef0123456789abcdef0123456789",
      "name": "lead_pulse.wav",
      "source_name": "lead.mid",
      "size": 123456,
      "download_url": "/api/synthesis-jobs/abcdef0123456789abcdef0123456789/download",
      "delete_url": "/api/synthesis-jobs/abcdef0123456789abcdef0123456789",
      "created_at": 1770000000.0,
      "updated_at": 1770000001.0,
      "expires_at": 1770086401.0
    }
  ]
}
```

`WorkspaceUploadResponse`：

```json
{
  "upload": {
    "file_id": "0123456789abcdef0123456789abcdef",
    "name": "lead.mid",
    "size": 12345,
    "created_at": 1770000000.0
  }
}
```

`WorkspaceQueueResponse`：

```json
{
  "uploads": [
    {
      "file_id": "0123456789abcdef0123456789abcdef",
      "name": "lead.mid",
      "size": 12345,
      "created_at": 1770000000.0
    }
  ]
}
```

`WorkspaceConfigResponse`：

```json
{
  "config": {
    "schema": "octabit.workspace_config.v1",
    "sample_rate": 48000,
    "layers": [
      {"type": "pulse", "duty": 0.5, "volume": 1.0, "curve_enabled": false, "frequency_curve": []}
    ]
  }
}
```

`SynthesisJobResponse`：

```json
{
  "job_id": "abcdef0123456789abcdef0123456789",
  "status": "ready",
  "source_name": "lead.mid",
  "created_at": 1770000000.0,
  "updated_at": 1770000001.0,
  "expires_at": 1770086401.0,
  "download_name": "lead_pulse.wav",
  "size_bytes": 123456,
  "download_url": "/api/synthesis-jobs/abcdef0123456789abcdef0123456789/download",
  "delete_url": "/api/synthesis-jobs/abcdef0123456789abcdef0123456789"
}
```

状态包括 `queued`、`rendering`、`ready`、`failed` 和 `expired`。

## 端点

### `GET /api/health`

接收：无请求体。

成功：

- 状态码：`200`
- 类型：JSON

```json
{
  "status": "ok",
  "service": "octabit-web"
}
```

### `GET /api/workspace`

接收：无请求体。

行为：

- cookie 缺失、无效、未知或已过期时，创建一个新的空工作区。
- 新工作区响应会设置 `octabit_workspace` cookie。

成功：

- 状态码：`200`
- 类型：`WorkspaceStateResponse`

### `POST /api/workspace/uploads`

接收：

- 类型：`multipart/form-data`
- `midi_file`：且仅一个 `.mid` 或 `.midi` 文件

成功：

- 状态码：`201`
- 类型：`WorkspaceUploadResponse`

错误：

- `400`，代码 `missing_midi_file`
- `400`，代码 `no_selected_file`
- `400`，代码 `empty_midi_file`
- `409`，代码 `workspace_queue_limit`
- `410`，代码 `workspace_expired`
- `413`，代码 `upload_too_large`
- `413`，代码 `workspace_upload_bytes_limit`
- `415`，代码 `unsupported_file_type`
- `500`，代码 `internal_error`

### `DELETE /api/workspace/uploads/<file_id>`

接收：路径参数 `file_id`，无请求体。

成功：

- 状态码：`204`
- 正文：空

错误：

- `400`，代码 `invalid_file_id`
- `404`，代码 `not_found`
- `410`，代码 `workspace_expired`

### `PATCH /api/workspace/queue`

接收：

```json
{
  "file_ids": ["0123456789abcdef0123456789abcdef"]
}
```

请求必须且仅包含所有活跃上传 id，每个 id 出现一次。

成功：

- 状态码：`200`
- 类型：`WorkspaceQueueResponse`

错误：

- `400`，代码 `invalid_file_id`
- `410`，代码 `workspace_expired`
- `422`，代码 `invalid_queue`

### `PUT /api/workspace/config`

接收：`WorkspaceConfigV1`。

成功：

- 状态码：`200`
- 类型：`WorkspaceConfigResponse`

错误：

- `410`，代码 `workspace_expired`
- `422`，代码 `invalid_workspace_config`

### `POST /api/synthesis-jobs`

创建渲染任务。

推荐的工作区请求：

```json
{
  "file_id": "0123456789abcdef0123456789abcdef",
  "config": {
    "schema": "octabit.workspace_config.v1",
    "sample_rate": 48000,
    "layers": [
      {"type": "pulse", "duty": 0.5, "volume": 1.0, "curve_enabled": false, "frequency_curve": []}
    ]
  }
}
```

兼容请求：

- 类型：`multipart/form-data`
- `midi_file`：`.mid` 或 `.midi` 文件
- `rate`：`44100`、`48000` 或 `96000`
- `layers_json`：渲染器层对象的 JSON 数组

成功：

- 状态码：`202`
- 类型：`SynthesisJobResponse`

错误：

- `400`，代码 `invalid_file_id`
- `400`，代码 `missing_midi_file`
- `400`，代码 `no_selected_file`
- `400`，代码 `empty_midi_file`
- `404`，代码 `not_found`
- `409`，代码 `workspace_converted_limit`
- `410`，代码 `workspace_expired`
- `413`，代码 `upload_too_large`
- `415`，代码 `unsupported_file_type`
- `429`，代码 `render_queue_full`
- `422`，代码 `invalid_sample_rate`
- `422`，代码 `invalid_layers`
- `422`，代码 `invalid_workspace_config`
- `500`，代码 `internal_error`

### `GET /api/synthesis-jobs/<job_id>`

接收：路径参数 `job_id`，无请求体。

成功：

- 状态码：`200`
- 类型：`SynthesisJobResponse`

错误和终止状态：

- `400`，代码 `invalid_job_id`
- `404`，代码 `not_found`
- `410`，代码 `workspace_expired`
- 归属任务已过期时返回 `410` JSON：`{"job_id": "...", "status": "expired"}`

### `GET /api/synthesis-jobs/<job_id>/download`

接收：路径参数 `job_id`，无请求体。

成功：

- 状态码：`200`
- 类型：WAV 文件附件

错误和终止状态：

- `400`，代码 `invalid_job_id`
- 归属任务失败时返回 `400` JSON `SynthesisJobResponse`
- `404`，代码 `not_found`
- 归属任务尚未 ready 时返回 `409` JSON `SynthesisJobResponse`
- `410`，代码 `workspace_expired`
- 归属任务已过期时返回 `410` JSON：`{"job_id": "...", "status": "expired"}`

### `DELETE /api/synthesis-jobs/<job_id>`

接收：路径参数 `job_id`，无请求体。

成功：

- 状态码：`204`
- 正文：空

错误：

- `400`，代码 `invalid_job_id`
- `404`，代码 `not_found`
- `410`，代码 `workspace_expired`
