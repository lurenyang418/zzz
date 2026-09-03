FROM node:26-bookworm-slim AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN npm install -g pnpm@11.21.0 && pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM golang:1.27-alpine AS backend-builder
ARG TARGETARCH
WORKDIR /app
COPY backend/go.mod ./backend/
COPY backend/*.go ./backend/
COPY backend/static ./backend/static
COPY --from=frontend-builder /app/dist ./backend/static
WORKDIR /app/backend
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -trimpath -ldflags='-s -w' -o /out/zzz .

FROM scratch AS binary-export
COPY --from=backend-builder /out/zzz /zzz

FROM alpine:3.24.1 AS runtime
ARG VERSION=dev
ARG VCS_REF=unknown
ARG BUILD_DATE
RUN apk add --no-cache ca-certificates && update-ca-certificates
WORKDIR /app
COPY --from=backend-builder /out/zzz /app/zzz
LABEL org.opencontainers.image.title="zzz" \
      org.opencontainers.image.description="Self-hosted GitHub file downloader with browser and Docker host exports" \
      org.opencontainers.image.url="https://github.com/lurenyang418/zzz" \
      org.opencontainers.image.documentation="https://github.com/lurenyang418/zzz/blob/main/README.md" \
      org.opencontainers.image.source="https://github.com/lurenyang418/zzz" \
      org.opencontainers.image.vendor="lurenyang418" \
      org.opencontainers.image.authors="lurenyang418" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.base.name="docker.io/library/alpine:3.24.1"
EXPOSE 8080
ENTRYPOINT ["/app/zzz"]
