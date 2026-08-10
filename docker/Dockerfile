FROM node:24-bookworm-slim AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN npm install -g pnpm@11.21.0 && pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM rust:1.94-alpine AS backend-builder
RUN apk add --no-cache \
    build-base \
    cmake \
    nasm \
    openssl-dev \
    perl \
    pkgconf
WORKDIR /app
COPY backend/Cargo.toml backend/Cargo.lock ./backend/
COPY backend/src ./backend/src
COPY --from=frontend-builder /app/dist ./backend/static
WORKDIR /app/backend
RUN cargo build --release --locked

FROM alpine:3.24.1
RUN apk add --no-cache \
    ca-certificates \
    libgcc \
    libstdc++ \
    libssl3 \
    && update-ca-certificates
WORKDIR /app
COPY --from=backend-builder /app/backend/target/release/zzz-backend /app/zzz
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app/zzz"]

FROM scratch AS binary-export
COPY --from=backend-builder /app/backend/target/release/zzz-backend /zzz
