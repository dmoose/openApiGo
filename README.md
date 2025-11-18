# openapi-go-md

`openapi-go-md` is a small Go module that converts OpenAPI / Swagger specs (v2 or v3, JSON or YAML) into Markdown. It provides:

- A CLI (`cmd/openapi-go-md`) for converting specs from files, stdin, or URLs.
- A Go library (`pkg/markdown`) that you can call directly from your own code.

## Installation

### Using `go install`

```bash
go install github.com/dmoose/openapi-go-md/cmd/openapi-go-md@latest
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

### Minimal example

```go
package main

import (
    "fmt"
    "os"

    "github.com/dmoose/openapi-go-md/pkg/markdown"
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
