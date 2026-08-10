# Contributing to zzz

Thanks for your interest in improving zzz.

## Before you start

- Search existing issues and pull requests before opening a new one.
- For security vulnerabilities, follow [SECURITY.md](SECURITY.md) instead of opening a public issue.
- Keep changes focused and explain the user-facing impact.

## Development

Install the frontend dependencies and run the frontend and backend separately:

```bash
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend dev
cargo run --manifest-path backend/Cargo.toml
```

Before submitting changes, run:

```bash
pnpm --dir frontend build
cargo fmt --manifest-path backend/Cargo.toml --check
cargo clippy --manifest-path backend/Cargo.toml --locked --all-targets -- -D warnings
cargo test --manifest-path backend/Cargo.toml --locked
```

If Docker is available, also validate the Compose configuration and build the image. Changes to frontend source must be included in the embedded release build.

## Pull requests

- Use a focused branch and a clear commit message.
- Describe what changed, why it changed, and how it was tested.
- Update both `README.md` and `README.en.md` when user-facing behavior or configuration changes.
- Do not include Tokens, credentials, generated dependency directories, or local downloads.

By contributing, you agree that your contribution is provided under the project's [MIT License](LICENSE).
