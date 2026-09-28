# PicFlow

PicFlow 是面向电商商品图的轻量处理平台。项目包含 Go 后端和 React 前端，本地开发时分别运行，部署时构建为一个同时包含前后端的镜像。

## 本地开发

本地开发不需要构建镜像。先启动后端：

```bash
cd backend
cp .env.example .env
go run . serve
```

再打开另一个终端启动前端：

```bash
cd frontend
npm install
npm run dev
```

访问 `http://localhost:5173`。Vite 会把 `/api` 请求代理到默认的 `http://localhost:8080`，并提供前端热更新。后端本地配置中的 `WEB_DIR` 保持为空。

## 单容器部署

在项目根目录执行：

```bash
docker compose up -d --build
```

访问 `http://localhost:8080`。最终镜像只有一个 Go 进程：它同时提供 API、健康检查、前端静态资源和 React Router fallback。

常用环境变量：

```bash
PICFLOW_PORT=8080
MAX_UPLOAD_SIZE_MB=20
WORKER_COUNT=2
GOPROXY=https://proxy.golang.org,direct
```

SQLite、原图和输出保存在 `picflow-data` 数据卷中，日志保存在 `picflow-logs` 数据卷中。

## 多平台镜像

macOS 上通过 Docker 构建的仍是 Linux 容器镜像。普通 `docker compose build` 默认生成当前 Docker 主机架构的镜像：Apple Silicon 通常为 `linux/arm64`，Intel Mac 和常见 x86 服务器通常为 `linux/amd64`。

要让同一个镜像标签同时支持 AMD64 和 ARM64，需要把多架构清单推送到镜像仓库：

```bash
docker login ghcr.io
PICFLOW_IMAGE=ghcr.io/<你的账号>/picflow:latest docker buildx bake --push
```

发布结果包含：

- `linux/amd64`：Intel/AMD 服务器和 x86_64 Linux。
- `linux/arm64`：Apple Silicon、ARM Linux 服务器。

部署机器会根据自身架构自动拉取同一标签下正确的镜像。服务器只运行预构建镜像时使用：

```bash
PICFLOW_IMAGE=ghcr.io/<你的账号>/picflow:latest docker compose pull
PICFLOW_IMAGE=ghcr.io/<你的账号>/picflow:latest docker compose up -d --no-build
```

如果只想在本机生成某个指定平台的镜像而不推送仓库，可以使用 `--load`，但一次只能加载一个平台：

```bash
docker buildx build --platform linux/amd64 --load -t picflow:amd64 .
docker buildx build --platform linux/arm64 --load -t picflow:arm64 .
```

## 在 GitHub 手动发布镜像

仓库内置 `.github/workflows/publish-image.yml`。把代码推送到 GitHub 默认分支后：

1. 打开仓库的 `Actions` 页面。
2. 选择“构建并发布容器镜像”。
3. 点击 `Run workflow`。
4. 输入镜像标签，例如 `latest` 或 `v1.0.0`，再次点击 `Run workflow`。

工作流会构建 `linux/amd64` 和 `linux/arm64`，并发布两个标签：

```text
ghcr.io/<仓库所有者>/<仓库名>:<输入的标签>
ghcr.io/<仓库所有者>/<仓库名>:sha-<提交短哈希>
```

工作流通过当前仓库自动提供的 `GITHUB_TOKEN` 登录 GHCR，不需要额外配置账号密码。它需要仓库允许 Actions 具有包写入权限；配置位置为 `Settings → Actions → General → Workflow permissions`。

GHCR 第一次发布的包默认是私有包。若希望其他人无需登录即可拉取，首次发布成功后进入 GitHub 个人或组织主页的 `Packages`，打开 PicFlow 包的 `Package settings`，在 `Danger Zone → Change visibility` 中改为 `Public`。公开后即可运行：

```bash
docker pull ghcr.io/<仓库所有者>/<仓库名>:latest
```

## 验证

```bash
cd backend
env GOCACHE=/tmp/picflow-go-cache go test ./...
env GOCACHE=/tmp/picflow-go-cache go vet ./...

cd ../frontend
npm run build
```
