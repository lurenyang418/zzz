# zzz

A self-hosted GitHub file downloader for NAS. Enter a GitHub repository, file, or directory URL, filter the contents, and download them in the browser or save them directly to a Docker host directory.

[中文（默认）](README.md) · English

## Features

- Supports GitHub repository, file, and directory URLs
- Reads directories through the Git Trees API and asks for a narrower path when GitHub truncates an oversized tree
- Ignore patterns, include-only patterns, and directory-tree selection
- Directory search, select all, expand/collapse all, and indeterminate states
- File-count and estimated-size preview before downloading
- Browser ZIP downloads or host-side folder/ZIP exports
- Folder exports support original and smart structures; smart mode removes shared parent directories while preserving necessary branches
- Request a GitHub Token when needed and keep it in the current browser
- Chinese/English UI and light/dark themes
- Frontend assets embedded into the final Go binary during the build
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

The “Save to host” option uses a path relative to `DOWNLOAD_ROOT`, such as `ebooks/2025`. Folder exports can use the original or smart structure: selecting only `a/b/c` starts the output at `c`; selecting both `a/b/c` and `a/d` removes the shared `a` while preserving the necessary `b/c` and `d` branches. ZIP mode creates a ZIP file inside the mounted directory. Submitted paths cannot escape `DOWNLOAD_ROOT`.

Compose is recommended after creating the host directory:

```bash
mkdir -p downloads
docker compose -f docker/docker-compose.yml up -d
```

### Docker Compose

The repository includes [docker/docker-compose.yml](docker/docker-compose.yml):

```bash
mkdir -p downloads
docker compose -f docker/docker-compose.yml up -d
```

It mounts `downloads/` in the repository root and enables host exports by default. `DOWNLOAD_ROOT` is a container-side path, and the destination entered in the page must be relative to it rather than a host absolute path. Compose sets the GitHub request timeout to 180 seconds. If the server cannot reach GitHub, check Docker networking, DNS, and proxy settings first.

## Images and releases

| Purpose | Address |
| --- | --- |
| Docker Hub | `lurenyang/zzz` |
| GHCR | `ghcr.io/lurenyang418/zzz` |

Images are published for `linux/amd64` and `linux/arm64`; Docker selects the matching architecture automatically.
Images include OCI metadata such as project/source URLs, license, version, revision, and build time.

When a `v*.*.*` tag is pushed, GitHub Actions creates a GitHub Release containing:

- `zzz-linux-amd64`
- `zzz-linux-arm64`
- `SHA256SUMS`

The Go backend is built as a static `CGO_ENABLED=0` Linux binary with frontend assets embedded. The runtime does not depend on system OpenSSL, libgcc, or libstdc++.

## Configuration

Create a `.env` file if you need to provide configuration values:

| Variable | Required | Description |
| --- | --- | --- |
| `ACCESS_TOKEN` | No | zzz service access token; when set, all APIs except health and capabilities require it |
| `GITHUB_TOKEN` | No | Default server-side GitHub Token |
| `DOWNLOAD_ROOT` | No | Container-side root for host exports; host export is disabled when unset |
| `GITHUB_TIMEOUT_SECS` | No | Per-request GitHub timeout, 180 seconds by default |
| `WRITE_TIMEOUT_SECS` | No | HTTP response write timeout, 1800 seconds by default |
| `MAX_FILES` | No | Maximum files per request, 10000 by default |
| `MAX_FILE_SIZE_BYTES` | No | Maximum size of one file, 100 MiB by default |
| `MAX_TOTAL_SIZE_BYTES` | No | Maximum total content size, 512 MiB by default |
| `MAX_ZIP_SIZE_BYTES` | No | Maximum generated ZIP size, 512 MiB by default |
| `MAX_TREE_REQUESTS` | No | Maximum GitHub metadata requests per tree browse, 500 by default |
| `MAX_CONCURRENT_JOBS` | No | Number of concurrent tree/download/export jobs, 2 by default |
| `RATE_LIMIT_PER_MINUTE` | No | Maximum jobs per IP per minute, 30 by default |
| `LISTEN_ADDR` | No | Listen address, `0.0.0.0:8080` by default |

### GitHub Token

zzz can start without a Token. For private repositories or API rate limits, create and paste a [Fine-grained Token](https://github.com/settings/personal-access-tokens/new) in the page. Grant only `Contents: read-only` access to the target repository when possible.

The page stores the Token only in the current browser's `localStorage` and sends it in the `X-GitHub-Token` header, never in the URL. Use HTTPS when accessing your zzz service with a Token; clearing the field removes the browser copy.

When `ACCESS_TOKEN` is configured, the page asks for the service access token and sends it in the `X-ZZZ-Access-Token` header. HTTPS is recommended in production.

### Network errors

GitHub requests are made by the zzz backend. If the server or Docker container cannot reach `api.github.com`, the API returns 502; request timeouts return 504. A Token can solve permissions and rate limits, but cannot replace network connectivity.

### Logs

The service writes startup, HTTP request, and error logs to stdout/stderr. With Docker Compose:

```bash
docker compose -f docker/docker-compose.yml logs -f zzz
```

Request logs include a request ID, source address, method, path, status code, response size, and duration. The request ID is also returned in the `X-Request-ID` response header.

## API

- `GET /api/health`: health check
- `GET /api/capabilities`: query host-export capabilities
- `GET /api/tree?url=...`: load a selectable file tree
- `GET /api/download?url=...`: generate a browser-downloadable ZIP
- `POST /api/export`: save a folder or ZIP below `DOWNLOAD_ROOT`

`/api/tree`, `/api/download`, and `/api/export` accept `X-GitHub-Token` for a request-scoped Token. It takes precedence over the server-side `GITHUB_TOKEN`.

Directory reading and directory file collection use the Git Trees API; direct file URLs use the Contents API for file metadata. If GitHub truncates an oversized recursive tree, use a more specific directory URL.

## Local development

Requirements: Go 1.26+, Node.js 24+, and pnpm 11.21.0.

Install and start the frontend:

```bash
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend dev
```

Start the Go backend in another terminal:

```bash
(cd backend && go run .)
```

Build a binary with embedded frontend assets:

```bash
pnpm --dir frontend build
cp -R frontend/dist/. backend/static/
mkdir -p dist
(cd backend && go build -trimpath -ldflags='-s -w' -o ../dist/zzz .)
```

Production only needs `dist/zzz`; frontend assets are embedded in the binary.

Useful checks:

```bash
pnpm --dir frontend build
(cd backend && gofmt -w *.go && go test ./...)
```

## Contributing and license

- Read [CONTRIBUTING.md](CONTRIBUTING.md) before contributing
- See [SECURITY.md](SECURITY.md) for security reports
- Follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Licensed under the [MIT License](LICENSE)
