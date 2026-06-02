# Vue 生产部署

这是 `octabit.cc` 预期使用的非 Docker 生产路径。

- Caddy 从 `frontend/dist` 提供构建后的 Vue 3 前端。
- Caddy 将 `/api/*`、`/static/previews/*` 和 `/synthesise*` 反向代理到
  `127.0.0.1:8000` 上的 Go 后端。
- Go 后端保持私有，负责工作区、上传、合成、下载、预览资源和旧路由兼容。
- 旧 Flask 栈保留在仓库中，用于回退参考和 fixture 重新生成，不作为常规生产路径。

`deploy/web-flask/` 中的 Docker 文件是旧 Flask 回退的另一条路径。除非生产计划改变，不要把 Docker 引入当前生产切换流程。

## 一次性服务器形态

建议使用 `/home/deploy/octabit` 这样的仓库检出路径、用于 Go 后端的
`octabit-web` systemd 服务，以及作为公开服务器的 Caddy。

Go 后端应保持为私有监听：

```bash
cd /home/deploy/octabit/backend
go build -o octabit-server ./cmd/server
PORT=8000 WEB_SYNTHESISE_JOB_ROOT=/var/lib/octabit ./octabit-server
```

Vue 切换前，先用服务器常规软件源安装 Node.js 和 npm。Vue 依赖安装应使用 lockfile：

```bash
cd /home/deploy/octabit/frontend
npm ci
npm run build
```

## Caddy 路由

生产模型使用 `Caddyfile.vue-production`：

```caddyfile
octabit.cc {
	encode zstd gzip

	handle /api/* {
		reverse_proxy 127.0.0.1:8000
	}

	handle /static/previews/* {
		reverse_proxy 127.0.0.1:8000
	}

	handle /synthesise* {
		reverse_proxy 127.0.0.1:8000
	}

	handle {
		root * /home/deploy/octabit/frontend/dist
		try_files {path} /index.html
		file_server
	}
}
```

这会让 Vue 应用成为公开前端，同时保留 API、预览音频路由和旧合成路由。`try_files`
回退用于 Vue/Vite 浏览器路由；由于 API 路由先被处理，它不会截获 API 请求。

## 部署流程

在生产 VM 上执行：

```bash
cd /home/deploy/octabit
git fetch --prune origin
git checkout main
git pull --ff-only origin main
cd backend
go build -o octabit-server ./cmd/server
cd /home/deploy/octabit
cd frontend
npm ci
npm run build
cd /home/deploy/octabit
sudo systemctl restart octabit-web
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

辅助脚本默认目标是 `main`：

```bash
deploy/production/deploy-vue-production.sh
```

如果生产检出路径不同，可设置 `APP_DIR=/path/to/octabit`。

辅助脚本会打印已部署的 commit，从 `frontend/src/i18n/*.json` 推导预期 UI locale，
确认本地 Vue bundle 包含每个 `toolbar.language_option.<locale>` 标记，然后在 Caddy
reload 后抓取 `PUBLIC_URL` 并确认公开 JavaScript bundle 也包含这些标记。`PUBLIC_URL`
默认是 `https://octabit.cc`；私有 dry run 可设置 `PUBLIC_URL=` 跳过公开检查。

如果 Caddy 服务的是单独静态 root，请设置 `WEB_ROOT`，让辅助脚本在 reload Caddy
之前把 `frontend/dist/` 发布到该目录。例如当前 VM 可使用：

```bash
WEB_ROOT=/var/www/octabit deploy/production/deploy-vue-production.sh
```

`WEB_ROOT` 必须是专用静态 Web root。辅助脚本使用 `rsync --delete`，不要指向包含无关文件的目录。

## Smoke 检查

在 VM 本机运行：

```bash
curl -fsS http://127.0.0.1:8000/api/health
test -f /home/deploy/octabit/frontend/dist/index.html
```

Caddy reload 后运行公开检查：

```bash
curl -fsS https://octabit.cc/
curl -fsS https://octabit.cc/api/health
curl -fsSI https://octabit.cc/static/previews/pulse_50.wav
```

然后在浏览器中上传一个小 MIDI 文件，确认刷新后工作区仍可恢复，运行合成并下载 WAV，切换主题/语言，并清空排队和已转换文件。

## 回滚

如果新的 Vue 构建在部署后失败，保持 Go 后端运行，并将检出回滚到上一个已知可用版本，然后重建
`frontend/dist` 并重载 Caddy：

```bash
cd /home/deploy/octabit
git checkout <previous-good-revision>
cd frontend
npm ci
npm run build
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

如果 Go 后端部署失败，先恢复前一个 systemd unit 或二进制版本：

```bash
sudo systemctl restart octabit-web
curl -fsS http://127.0.0.1:8000/api/health
```
