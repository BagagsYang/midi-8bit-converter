# Flask 后端 Docker 部署

Language/语言: [English](./README.md) | 简体中文

此 Docker 路径打包 Flask 后端和旧 Flask 渲染前端回退。预期的生产路径不使用 Docker：Caddy
从 `frontend/dist` 提供 Vue 构建产物，并将 API、预览和旧兼容路由反向代理到
`127.0.0.1:8000` 上的 Flask/Gunicorn。见 `../production/README.zh-Hans.md`。

镜像包含 `legacy/web-flask/`、`legacy/python-renderer/` 中的共享渲染器入口、`assets/previews/`
中的共享预览 WAV 文件，以及项目许可证。它不会打包 Vue 前端、macOS 应用或 Windows 桌面应用。

Compose 文件会把服务在服务器上绑定到 `127.0.0.1:8000`，便于在添加公开反向代理之前先通过 SSH 隧道测试。

## 构建并启动

在 Debian 服务器的仓库根目录执行：

```bash
docker compose -f compose.web.yml up -d --build
```

检查服务状态：

```bash
docker compose -f compose.web.yml ps
```

容器健康时，`docker compose ps` 应显示服务正在运行并处于 healthy 状态。

在服务器本机测试：

```bash
curl http://127.0.0.1:8000
```

查看日志：

```bash
docker compose -f compose.web.yml logs -f
```

## 通过 SSH 隧道测试

在你的 Mac 上打开隧道：

```bash
ssh -p 22080 -N -L 18080:127.0.0.1:8000 debian@42.121.121.121
```

然后在 Mac 上打开这个地址：

```text
http://127.0.0.1:18080
```

## 停止

在 Debian 服务器的仓库根目录执行：

```bash
docker compose -f compose.web.yml down
```

## 生产部署说明

- 容器会以非 root 用户运行 Gunicorn，并监听 `0.0.0.0:${PORT:-8000}`。
- Dockerfile 通过摘要固定 Python 基础镜像，并使用 `deploy/web-flask/build-requirements.lock` 和 `deploy/web-flask/requirements.lock` 通过 pip 哈希校验安装依赖。Python 依赖变化时，应有意重新生成这些 lock 文件。
- 后台合成使用有界渲染池。`WEB_RENDER_WORKERS` 默认每个容器最多 2 个活跃渲染，`WEB_RENDER_QUEUE_SIZE` 默认最多 8 个等待渲染。
- 镜像默认值和 `compose.web.yml` 都将 `GUNICORN_TIMEOUT` 设置为 600 秒。这样慢速 SSH 隧道下载有更长时间完成，但浏览器仍会在服务器完成渲染后再下载生成的 WAV。
- 容器包含一个使用 Python 标准库访问 `/` 的轻量健康检查，因此镜像里不需要额外安装 curl。
- 匿名工作区元数据、上传的 MIDI 文件和生成的 WAV 文件都位于 `WEB_SYNTHESISE_JOB_ROOT` 下，默认是 `/tmp/octabit-jobs`；Compose 文件把 `/tmp` 挂载为 1 GB 的内存 tmpfs，不会持久化上传数据。
- 工作区文件会在最后一次活动后保留 `WEB_WORKSPACE_TTL_SECONDS` 秒，默认 86400 秒。默认上限是每个工作区 20 个排队文件、100 MiB 活跃 MIDI 上传和 20 个已转换文件。
- 旧的 ready 渲染任务会保留 `WEB_DOWNLOAD_TTL_SECONDS` 秒，默认 1800 秒。当用户清空队列或已转换文件列表时，浏览器会请求服务器立即删除对应的临时文件。
- 主机端口有意绑定到 `127.0.0.1:8000`，用于仅通过隧道访问的测试阶段。
- 在 `octabit.cc` 的 Vue 生产部署中，应由 Caddy 提供 `frontend/dist`，并将
  `/api/*`、`/static/previews/*` 和 `/synthesise*` 反向代理到此服务。Flask/Gunicorn
  服务应保持为服务器本机或 Docker 网络内的私有服务。
