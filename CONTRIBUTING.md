# Contributing

Thanks for your interest in contributing to `openapi-go-md`!

This project is a small Go CLI and library for turning OpenAPI / Swagger specs into Markdown. Contributions that improve correctness, robustness, or usability are welcome.

## Development Setup

- Go 1.25 or newer.
- Clone the repository and ensure module dependencies are available:

  ```bash
  go mod tidy
  ```

## Common Tasks

From the repo root:

- Build the CLI binary into `bin/`:

  ```bash
  make build
  ```

- Run tests:

  ```bash
  make test
  ```

- Run basic static analysis:

  ```bash
  make lint
  ```

- Format Go code:

  ```bash
  make fmt
  ```

## Pull Requests

- Keep changes focused and as small as practical.
- Add or update tests for any behavior changes.
- Ensure `make test` and `make lint` pass before opening a PR.
- Include a brief description of the motivation and behavior changes in the PR description.
