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
(cd backend && go run .)
```

Before submitting changes, run:

```bash
pnpm --dir frontend build
(cd backend && gofmt -w *.go && go test ./...)
```

If Docker is available, also validate the Compose configuration and build the image. Changes to frontend source must be included in the embedded release build.

Dependabot monitors GitHub Actions, frontend npm packages, Go modules, and Docker base images. Patch and minor Dependabot updates may be merged automatically after the required checks pass; major updates remain manual.

## Pull requests

- Use a focused branch and a clear commit message.
- Describe what changed, why it changed, and how it was tested.
- Update both `README.md` and `README.en.md` when user-facing behavior or configuration changes.
- Do not include Tokens, credentials, generated dependency directories, or local downloads.

By contributing, you agree that your contribution is provided under the project's [MIT License](LICENSE).
