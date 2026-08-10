FROM node:24-bookworm-slim AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN npm install -g pnpm@11.21.0 && pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM golang:1.26-alpine AS backend-builder
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
RUN apk add --no-cache ca-certificates && update-ca-certificates
WORKDIR /app
COPY --from=backend-builder /out/zzz /app/zzz
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app/zzz"]
