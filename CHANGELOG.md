# Changelog

All notable changes to zzz will be documented here.

The project follows a lightweight changelog format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.1.1] - 2026-08-10

### Changed

- Added writable-directory detection for Docker host exports so the UI disables an unavailable destination before downloading.
- Added request IDs and access logs with status, response size, client address, and duration.
- Synchronized Docker host export instructions and clarified that `DOWNLOAD_ROOT` is a container-side path.
- Added OCI image metadata and release build arguments for version, revision, and build time.

## [0.1.0] - 2026-08-10

### Changed

- Replaced the Rust backend with a pure Go backend using only the standard library.
- Embedded frontend assets into the Go binary and removed the runtime's Rust/OpenSSL toolchain dependencies.
- Directory browsing now uses the GitHub Git Trees API and reports upstream network failures as 502/504.
- Directory browsing separates tree discovery limits from download limits and caps metadata requests per operation.
- Added optional service authentication, request concurrency limits, per-IP rate limits, and sanitized internal errors.
- Browser ZIP downloads now use a temporary file instead of retaining the complete archive in memory.
- Added Dependabot configuration and guarded automatic merge for patch/minor dependency updates.

### Added

- GitHub repository, directory-tree selection, filtering, browser ZIP download, and host-side export.
- Chinese and English UI with light and dark themes.
- Multi-architecture Docker images and Linux binary release artifacts.
