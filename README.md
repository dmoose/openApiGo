# openApiGo

`openApiGo` is a small Go module that converts OpenAPI / Swagger specs (v2 or v3, JSON or YAML) into Markdown. It provides:

- A CLI (`cmd/openapi-go-md`) for converting specs from files, stdin, or URLs.
- A Go library (`pkg/markdown`) that you can call directly from your own code.

## Supported specifications and behavior

- **Specifications**: Swagger 2.0 (documents with top-level `swagger: "2.0"`) and OpenAPI 3.x (documents with top-level `openapi: "3.x"`).
- **Input formats**: JSON or YAML, read from a local file, stdin, or an HTTP(S) URL.
- **Behavior**: conversion is best-effort and never panics on user input; malformed specs may return an error or, when partially interpretable, produce incomplete Markdown.

Some features are intentionally minimal for now (for example, no per-operation security breakdown and limited expansion of deeply nested schemas). These may evolve in future versions.

## Installation

### Using `go install`

```bash
go install github.com/dmoose/ApiGo-go-md/cmd/openApiGo@latest
```

This installs the `openapi-go-md` binary into your `GOBIN` (usually `$(go env GOPATH)/bin`).

### Building from source

From the repo root:

```bash
go build ./cmd/openapi-go-md
```

This produces a local `openapi-go-md` binary in `cmd/openapi-go-md/`.

## CLI Usage

The CLI reads an OpenAPI/Swagger document (JSON or YAML), normalizes it, detects whether it is Swagger 2.0 or OpenAPI 3.x, and prints a Markdown description to stdout or an output file.

### Flags

- `--file`   — Path to spec file, or `-` to read from stdin.
- `--url`    — HTTP(S) URL to fetch the spec from.
- `--out`    — Optional output file path (defaults to stdout).
- `--format` — `auto` (default), `json`, or `yaml` to control input parsing.

Exactly one of `--file` or `--url` is required.

### Examples

#### From a local file

```bash
openapi-go-md --file openapi.yaml --format auto > api.md
```

#### From stdin

```bash
cat openapi.yaml | openapi-go-md --file - --format auto > api.md
```

#### From a URL

```bash
openapi-go-md \
  --url https://example.com/openapi.json \
  --format json \
  --out api.md
```

If input parsing fails (e.g., invalid JSON/YAML), or the spec cannot be interpreted as Swagger 2.0 / OpenAPI 3.x, the CLI prints an error to stderr and exits with a non-zero status.

## Library Usage

The `pkg/markdown` package exposes a single high-level function:

- `ToMarkdown(data []byte, opts Options) (string, error)`

`Options` controls how the input is interpreted:

- `Format` — One of `FormatAuto`, `FormatJSON`, or `FormatYAML`.
- `SkipValidation` — When `true`, skips extra validation for OpenAPI 3 documents.

The generated Markdown includes:

- Overview, authentication, servers, tags.
- Endpoints grouped by tag, with parameters, responses, operation IDs, and media types.
- Schemas with property types, required flags, default values, and enums where available.

See `pkg/markdown/testdata` for example Swagger 2.0 and OpenAPI 3.x documents used in tests.

### Minimal example

```go
package main

import (
    "fmt"
    "os"

    "github.com/dmoose/openApiGo/pkg/markdown"
)

func main() {
    // Read an OpenAPI / Swagger spec from disk.
    data, err := os.ReadFile("openapi.yaml")
    if err != nil {
        panic(err)
    }

    // Convert to Markdown, auto-detecting JSON vs YAML.
    md, err := markdown.ToMarkdown(data, markdown.Options{Format: markdown.FormatAuto})
    if err != nil {
        panic(err)
    }

    fmt.Println(md)
}
```

### Skipping validation for problematic specs

If you need to generate Markdown from a spec that fails strict validation, you can skip validation explicitly:

```go
md, err := markdown.ToMarkdown(data, markdown.Options{
    Format:         markdown.FormatAuto,
    SkipValidation: true,
})
```

The converter still attempts to parse the document and generate Markdown; it just avoids the additional validation step.

## Stability & versioning

This module follows semantic versioning. Until `v1.0.0` the API may change in minor ways as it evolves; once `v1.0.0` is tagged, breaking changes will only occur in new major versions.

For most users it is sufficient to depend on the latest tagged version, for example:

- `go get github.com/dmoose/openApiGo@latest`

## Development

From the repo root:

- Run all tests:

  ```bash
  go test ./...
  ```

- Run only the markdown package tests:

  ```bash
  go test ./pkg/markdown
  ```

- Run only the CLI tests:

  ```bash
  go test ./cmd/openapi-go-md
  ```

- Basic static analysis:

  ```bash
  go vet ./...
  ```
