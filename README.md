# zzz

面向 NAS 的自托管 GitHub 文件下载器。输入 GitHub 仓库、文件或目录 URL，按规则筛选内容，并通过浏览器下载或直接保存到 Docker 宿主机。

[English](README.en.md) · 中文

## 功能

- 支持 GitHub 仓库、文件和目录 URL
- 使用 Git Trees API 读取递归目录，超大树被 GitHub 截断时提示缩小范围
- 支持忽略规则、仅包含规则和目录树勾选
- 支持目录搜索、全选、展开/折叠和半选状态
- 下载前显示文件数量和预计大小
- 支持浏览器下载 ZIP，或保存为宿主机文件夹/ZIP
- 文件夹导出支持原始结构和智能结构；智能结构会去除选中内容共同的父目录，同时保留必要的分支
- GitHub Token 可按需申请、填写，并保存在当前浏览器
- 支持中英文界面和亮暗主题
- 前端资源在 Go 构建阶段嵌入最终二进制
- Docker 镜像支持 `linux/amd64` 和 `linux/arm64`

## 快速开始

### 直接使用 Docker

```bash
docker pull lurenyang/zzz:latest
docker run --rm -p 8080:8080 lurenyang/zzz:latest
```

打开 <http://localhost:8080>。

### 保存到 Docker 宿主机

将宿主机目录挂载到容器的 `/downloads`，并设置 `DOWNLOAD_ROOT`：

```bash
mkdir -p downloads
docker run --rm -p 8080:8080 \
  -v "$PWD/downloads:/downloads" \
  -e DOWNLOAD_ROOT=/downloads \
  lurenyang/zzz:latest
```

页面中的“保存到主机”使用相对于 `DOWNLOAD_ROOT` 的路径，例如 `ebooks/2025`。文件夹模式可以选择原始结构或智能结构：例如只选择 `a/b/c` 时，智能结构输出从 `c` 开始；同时选择 `a/b/c` 和 `a/d` 时，会去掉共同的 `a`，保留 `b/c` 与 `d` 两个必要分支。ZIP 模式在挂载目录中生成 ZIP。页面只能提交相对路径，不能跳出 `DOWNLOAD_ROOT`。

Compose 推荐先创建目录后启动：

```bash
mkdir -p downloads
docker compose -f docker/docker-compose.yml up -d
```

### Docker Compose

仓库提供了 [docker/docker-compose.yml](docker/docker-compose.yml)：

```bash
mkdir -p downloads
docker compose -f docker/docker-compose.yml up -d
```

默认挂载仓库根目录下的 `downloads/`，并启用宿主机导出。`DOWNLOAD_ROOT` 是容器内路径，页面中的目标路径也必须是相对于它的路径，不能填写宿主机绝对路径。Compose 默认将 GitHub 请求超时设置为 180 秒；如服务器无法访问 GitHub，需先检查 Docker 网络、DNS 和代理配置。

## 镜像与 Release

| 用途 | 地址 |
| --- | --- |
| Docker Hub | `lurenyang/zzz` |
| GHCR | `ghcr.io/lurenyang418/zzz` |

镜像同时提供 `linux/amd64` 和 `linux/arm64`，Docker 会根据宿主机自动选择架构。
镜像包含 OCI 标准元数据，包括项目地址、源码地址、许可证、版本、提交 SHA 和构建时间。

推送 `v*.*.*` 版本 tag 时，GitHub Actions 会创建 GitHub Release，并上传：

- `zzz-linux-amd64`
- `zzz-linux-arm64`
- `SHA256SUMS`

Go 后端使用 `CGO_ENABLED=0` 构建静态 Linux 二进制，前端资源会嵌入其中。运行时不依赖系统 OpenSSL、libgcc 或 libstdc++。

## 配置

可以根据需要创建 `.env` 作为配置文件：

| 变量 | 必填 | 说明 |
| --- | --- | --- |
| `ACCESS_TOKEN` | 否 | zzz 服务访问密钥；配置后除健康检查和能力查询外的 API 都需要它 |
| `GITHUB_TOKEN` | 否 | 服务端默认 GitHub Token |
| `DOWNLOAD_ROOT` | 否 | 宿主机导出的容器内根目录；未配置时页面禁用“保存到主机” |
| `GITHUB_TIMEOUT_SECS` | 否 | 单次 GitHub 请求超时时间，默认 180 秒 |
| `WRITE_TIMEOUT_SECS` | 否 | HTTP 响应写入超时时间，默认 1800 秒 |
| `MAX_FILES` | 否 | 单次最多处理的文件数量，默认 10000 |
| `MAX_FILE_SIZE_BYTES` | 否 | 单个文件大小限制，默认 100 MiB |
| `MAX_TOTAL_SIZE_BYTES` | 否 | 单次内容总大小限制，默认 512 MiB |
| `MAX_ZIP_SIZE_BYTES` | 否 | 生成 ZIP 大小限制，默认 512 MiB |
| `MAX_TREE_REQUESTS` | 否 | 单次目录浏览最多请求 GitHub 元数据的次数，默认 500 |
| `MAX_CONCURRENT_JOBS` | 否 | 同时进行的目录读取、下载和导出任务数，默认 2 |
| `RATE_LIMIT_PER_MINUTE` | 否 | 单 IP 每分钟最多发起的任务数，默认 30 |
| `LISTEN_ADDR` | 否 | 监听地址，默认 `0.0.0.0:8080` |

### GitHub Token

不配置 Token 也可以启动 zzz。遇到私有仓库权限不足或 API 限流时，可以在页面中申请并填写 [Fine-grained Token](https://github.com/settings/personal-access-tokens/new)，建议只授予目标仓库的 `Contents: read-only` 权限。

页面 Token 只保存在当前浏览器的 `localStorage`，通过 `X-GitHub-Token` 请求头发送，不会进入 URL。使用 Token 时建议通过 HTTPS 访问 zzz；清除输入框即可删除浏览器中的 Token。

如果配置了 `ACCESS_TOKEN`，页面会要求填写服务访问密钥，并通过 `X-ZZZ-Access-Token` 请求头发送。生产环境建议同时使用 HTTPS。

### 网络错误

GitHub 请求由 zzz 后端发起。服务器或 Docker 容器无法访问 `api.github.com` 时，接口会返回 502；请求超时会返回 504。Token 只能解决权限和限流问题，不能替代服务器网络连接。

### 日志

服务将启动日志、HTTP 请求日志和错误日志输出到 stdout/stderr，Docker 可以直接查看：

```bash
docker compose -f docker/docker-compose.yml logs -f zzz
```

请求日志包含请求 ID、来源地址、方法、路径、状态码、响应大小和耗时；请求 ID 会通过 `X-Request-ID` 响应头返回，便于定位问题。

## API

- `GET /api/health`：健康检查
- `GET /api/capabilities`：查询宿主机导出能力
- `GET /api/tree?url=...`：读取可勾选的文件树
- `GET /api/download?url=...`：生成并返回浏览器下载 ZIP
- `POST /api/export`：保存到 `DOWNLOAD_ROOT` 下的文件夹或 ZIP

`/api/tree`、`/api/download` 和 `/api/export` 支持通过 `X-GitHub-Token` 传入本次请求使用的 Token，该 Token 优先于服务端的 `GITHUB_TOKEN`。

目录读取和目录文件收集使用 Git Trees API；直接文件 URL 通过 Contents API 获取文件元数据。对于过大的递归树，GitHub 可能返回截断结果，此时请使用更具体的目录 URL。

## 本地开发

要求：Go 1.26+、Node.js 24+、pnpm 11.21.0。

安装并启动前端：

```bash
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend dev
```

另一个终端启动 Go 后端：

```bash
(cd backend && go run .)
```

构建带嵌入前端资源的二进制：

```bash
pnpm --dir frontend build
cp -R frontend/dist/. backend/static/
mkdir -p dist
(cd backend && go build -trimpath -ldflags='-s -w' -o ../dist/zzz .)
```

生产运行时只需要 `dist/zzz`；前端资源已经嵌入二进制。

常用检查：

```bash
pnpm --dir frontend build
(cd backend && gofmt -w *.go && go test ./...)
```

## 贡献与协议

- 贡献代码请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)
- 安全问题请阅读 [SECURITY.md](SECURITY.md)
- 行为规范见 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- 本项目使用 [MIT License](LICENSE)
