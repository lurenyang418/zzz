# zzz

A self-hosted GitHub file downloader for NAS. Enter a GitHub repository, file, or directory URL, filter the contents, and download them in the browser or save them directly to a Docker host directory.

[中文（默认）](README.md) · English

## Features

- Supports GitHub repository, file, and directory URLs
- Ignore patterns, include-only patterns, and directory-tree selection
- Directory search, select all, expand/collapse all, and indeterminate states
- File-count and estimated-size preview before downloading
- Browser ZIP downloads or host-side folder/ZIP exports
- Request a GitHub Token when needed and keep it in the current browser
- Chinese/English UI and light/dark themes
- Frontend assets embedded into the final Rust binary during the build
- Docker images for `linux/amd64` and `linux/arm64`

## Quick start

### Run with Docker

```bash
docker pull lurenyang/zzz:latest
docker run --rm -p 8080:8080 lurenyang/zzz:latest
```

Open <http://localhost:8080>.

### Save to the Docker host

Mount a host directory at `/downloads` and set `DOWNLOAD_ROOT`:

```bash
mkdir -p downloads
docker run --rm -p 8080:8080 \
  -v "$PWD/downloads:/downloads" \
  -e DOWNLOAD_ROOT=/downloads \
  lurenyang/zzz:latest
```

The “Save to host” option uses a path relative to `DOWNLOAD_ROOT`, such as `ebooks/2025`. Folder mode preserves the GitHub directory structure; ZIP mode creates a ZIP file inside the mounted directory. For safety, submitted paths must be relative and cannot escape `DOWNLOAD_ROOT`.

The container runs as a non-root user. Adjust the host directory permissions or ACL if it is not writable.

### Docker Compose

The repository includes [docker/docker-compose.yml](docker/docker-compose.yml):

```bash
docker compose -f docker/docker-compose.yml up -d
```

It mounts `downloads/` in the repository root and enables host exports by default.

## Images and releases

| Purpose | Address |
| --- | --- |
| Docker Hub | `lurenyang/zzz` |
| GHCR | `ghcr.io/lurenyang418/zzz` |

Images are published for `linux/amd64` and `linux/arm64`; Docker selects the matching architecture automatically.

When a `v*.*.*` tag is pushed, GitHub Actions creates a GitHub Release containing:

- `zzz-linux-amd64`
- `zzz-linux-arm64`
- `SHA256SUMS`

The binaries are built with Alpine/musl for Linux NAS systems and require the corresponding musl runtime and CA certificates. Network requests use Rustls and do not depend on system OpenSSL. Docker images are recommended for most deployments.

## Configuration

Copy [.env.example](.env.example) as a configuration reference:

| Variable | Required | Description |
| --- | --- | --- |
| `GITHUB_TOKEN` | No | Default server-side GitHub Token |
| `DOWNLOAD_ROOT` | No | Container-side root for host exports; the host-export option is disabled when unset |
| `GITHUB_TIMEOUT_SECS` | No | GitHub request timeout, 60 seconds by default |
| `MAX_FILES` | No | Maximum files processed per request |
| `MAX_FILE_SIZE_BYTES` | No | Maximum size of one file |
| `MAX_TOTAL_SIZE_BYTES` | No | Maximum total content size per request |
| `MAX_ZIP_SIZE_BYTES` | No | Maximum generated ZIP size |

### GitHub Token

zzz can start without a Token. For private repositories or API rate limits, create and paste a [Fine-grained Token](https://github.com/settings/personal-access-tokens/new) in the page. Grant only `Contents: read-only` access to the target repository when possible.

The page stores the Token only in the current browser's `localStorage` and sends it in the `X-GitHub-Token` header, never in the URL. Use HTTPS when accessing your zzz service with a Token; clearing the field removes the browser copy.

## API

- `GET /api/health`: health check
- `GET /api/capabilities`: query host-export capabilities
- `GET /api/tree?url=...`: load a selectable file tree
- `GET /api/download?url=...`: generate a browser-downloadable ZIP
- `POST /api/export`: save a folder or ZIP below `DOWNLOAD_ROOT`

`/api/tree`, `/api/download`, and `/api/export` accept `X-GitHub-Token` for a request-scoped Token. It takes precedence over the server-side `GITHUB_TOKEN`.

Directory reading and file collection use the Git Trees API. Very large recursive trees may be truncated by GitHub; use a more specific directory URL in that case.

## Local development

Requirements: Rust 1.94+, Node.js 24+, and pnpm 11.21.0.

```bash
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend dev
```

Start the backend in another terminal:

```bash
cargo run --manifest-path backend/Cargo.toml
```

Build a fully embedded binary:

```bash
pnpm --dir frontend build
rm -rf backend/static
mkdir -p backend/static
cp -R frontend/dist/. backend/static/
cargo build --manifest-path backend/Cargo.toml --release --locked
```

The production runtime only needs `backend/target/release/zzz-backend`; frontend assets are embedded with `rust-embed`. The project uses Rustls and does not depend on system OpenSSL.

Useful checks:

```bash
pnpm --dir frontend build
cargo fmt --manifest-path backend/Cargo.toml --check
cargo clippy --manifest-path backend/Cargo.toml --locked --all-targets -- -D warnings
cargo test --manifest-path backend/Cargo.toml --locked
```

## Contributing and license

- Read [CONTRIBUTING.md](CONTRIBUTING.md) before contributing
- See [SECURITY.md](SECURITY.md) for security reports
- Follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Licensed under the [MIT License](LICENSE)
