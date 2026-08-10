# AGENTS.md

## Project overview

`zzz` is a self-hosted GitHub file downloader. The backend is a pure Go HTTP service and the frontend is a Svelte/Vite application. The final Go binary embeds the built frontend assets.

## Repository layout

- `backend/`: Go server, GitHub API client, filtering, ZIP creation, host export, and tests.
- `frontend/`: Svelte UI, API client, bilingual messages, theme handling, and Vite build configuration.
- `docker/`: multi-stage Docker build and Docker Compose deployment configuration.
- `.github/`: container/release workflows, Dependabot configuration, and repository templates.
- `README.md` / `README.en.md`: Chinese and English user documentation; keep both synchronized for user-facing changes.

## Development commands

```bash
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend build
(cd backend && gofmt -w *.go && go test ./...)
(cd backend && go vet ./...)
docker compose -f docker/docker-compose.yml config
```

For race detection:

```bash
(cd backend && go test -race ./...)
```

## Build rules

- Frontend assets must be built and embedded into the Go binary; do not ship a runtime image that copies frontend files separately.
- The Dockerfile is the authoritative release build: it builds the frontend, copies `dist/` into `backend/static/`, and compiles the Go binary with `CGO_ENABLED=0`.
- `backend/static/.gitkeep` is intentional. It keeps the explicit `go:embed` pattern valid before a frontend build.
- `frontend/svelte.config.js` is intentionally absent; Vite uses the default Svelte configuration.
- Keep the runtime image free of Rust, OpenSSL, libgcc, and libstdc++ dependencies unless a verified requirement is introduced.
- Preserve `linux/amd64` and `linux/arm64` Docker and binary release support.
- Keep runtime image OCI labels synchronized with repository metadata; release builds inject version, commit, and build time.

## Security and operations

- Keep `ACCESS_TOKEN` optional but supported; never log GitHub or service tokens.
- Preserve path traversal and symlink protections for `DOWNLOAD_ROOT` exports.
- Keep download size, file count, concurrency, and per-IP rate limits enforced.
- Host exports require a writable bind mount. Compose users should set `ZZZ_UID` and `ZZZ_GID` to the host user's IDs.
- Logs go to stdout/stderr. Access logs include a generated request ID; avoid logging query strings because they may contain large GitHub URLs.

## Documentation and releases

- Update both README files, `CHANGELOG.md`, and relevant contribution/security documentation when behavior or configuration changes.
- Dependabot monitors GitHub Actions, npm, Go modules, and Docker images. Patch/minor updates may auto-merge after checks; major updates require review.
- Version tags matching `v*.*.*` publish multi-architecture images and embedded Linux binaries to the configured registries and GitHub Release.
- Do not commit generated `frontend/dist/`, `frontend/node_modules/`, local downloads, credentials, or tokens.
