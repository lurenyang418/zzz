# zzz

面向 NAS 的自托管 GitHub 文件下载器。输入 GitHub 仓库、文件或目录 URL，按规则筛选内容，并通过浏览器下载或直接保存到 Docker 宿主机。

[English](README.en.md) · 中文（默认）

## 功能

- 支持 GitHub 仓库、文件和目录 URL
- 支持忽略规则、仅包含规则和目录树勾选
- 支持目录搜索、全选、展开/折叠和半选状态
- 下载前显示文件数量和预计大小
- 支持浏览器下载 ZIP，或保存为宿主机文件夹/ZIP
- GitHub Token 可按需申请、填写，并保存在当前浏览器
- 支持中英文界面和亮暗主题
- 前端资源在 Rust 构建阶段嵌入最终二进制
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

页面中的“保存到主机”使用相对于 `DOWNLOAD_ROOT` 的路径，例如 `ebooks/2025`。文件夹模式保留 GitHub 原目录结构，ZIP 模式在挂载目录中生成 ZIP。为保证安全，页面只能提交相对路径，不能跳出 `DOWNLOAD_ROOT`。

容器以非 root 用户运行。如果宿主机目录不可写，请先调整目录权限或 ACL。

### Docker Compose

仓库提供了 [docker/docker-compose.yml](docker/docker-compose.yml)：

```bash
docker compose -f docker/docker-compose.yml up -d
```

默认挂载仓库根目录下的 `downloads/`，并启用宿主机导出。

## 镜像与 Release

| 用途 | 地址 |
| --- | --- |
| Docker Hub | `lurenyang/zzz` |
| GHCR | `ghcr.io/lurenyang418/zzz` |

镜像同时提供 `linux/amd64` 和 `linux/arm64`，Docker 会根据宿主机自动选择架构。

推送 `v*.*.*` 版本 tag 时，GitHub Actions 会创建 GitHub Release，并上传：

- `zzz-linux-amd64`
- `zzz-linux-arm64`
- `SHA256SUMS`

二进制使用 Alpine/musl 构建，适合 Linux NAS；运行时需要对应系统的 musl、CA 和 OpenSSL 运行库。一般情况下推荐优先使用 Docker 镜像。

## 配置

可以复制 [.env.example](.env.example) 作为配置参考：

| 变量 | 必填 | 说明 |
| --- | --- | --- |
| `GITHUB_TOKEN` | 否 | 服务端默认 GitHub Token |
| `DOWNLOAD_ROOT` | 否 | 宿主机导出的容器内根目录；未配置时页面禁用“保存到主机” |
| `GITHUB_TIMEOUT_SECS` | 否 | GitHub 请求超时时间，默认 60 秒 |
| `MAX_FILES` | 否 | 单次最多处理的文件数量 |
| `MAX_FILE_SIZE_BYTES` | 否 | 单个文件大小限制 |
| `MAX_TOTAL_SIZE_BYTES` | 否 | 单次内容总大小限制 |
| `MAX_ZIP_SIZE_BYTES` | 否 | 生成 ZIP 大小限制 |

### GitHub Token

不配置 Token 也可以启动 zzz。遇到私有仓库权限不足或 API 限流时，可以在页面中申请并填写 [Fine-grained Token](https://github.com/settings/personal-access-tokens/new)，建议只授予目标仓库的 `Contents: read-only` 权限。

页面 Token 只保存在当前浏览器的 `localStorage`，通过 `X-GitHub-Token` 请求头发送，不会进入 URL。使用 Token 时建议通过 HTTPS 访问 zzz；清除输入框即可删除浏览器中的 Token。

## API

- `GET /api/health`：健康检查
- `GET /api/capabilities`：查询宿主机导出能力
- `GET /api/tree?url=...`：读取可勾选的文件树
- `GET /api/download?url=...`：生成并返回浏览器下载 ZIP
- `POST /api/export`：保存到 `DOWNLOAD_ROOT` 下的文件夹或 ZIP

`/api/tree`、`/api/download` 和 `/api/export` 支持通过 `X-GitHub-Token` 传入本次请求使用的 Token，该 Token 优先于服务端的 `GITHUB_TOKEN`。

目录读取和文件收集使用 Git Trees API。对于过大的递归树，GitHub 可能返回截断结果，此时请使用更具体的目录 URL。

## 本地开发

要求：Rust 1.94+、Node.js 24+、pnpm 11.21.0。

```bash
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend dev
```

另一个终端启动后端：

```bash
cargo run --manifest-path backend/Cargo.toml
```

完整嵌入式构建：

```bash
pnpm --dir frontend build
rm -rf backend/static
mkdir -p backend/static
cp -R frontend/dist/. backend/static/
cargo build --manifest-path backend/Cargo.toml --release --locked
```

生产运行时只需要 `backend/target/release/zzz-backend`，前端资源已经通过 `rust-embed` 嵌入二进制。

常用检查：

```bash
pnpm --dir frontend build
cargo fmt --manifest-path backend/Cargo.toml --check
cargo clippy --manifest-path backend/Cargo.toml --locked --all-targets -- -D warnings
cargo test --manifest-path backend/Cargo.toml --locked
```

## 贡献与协议

- 贡献代码请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)
- 安全问题请阅读 [SECURITY.md](SECURITY.md)
- 行为规范见 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- 本项目使用 [MIT License](LICENSE)
