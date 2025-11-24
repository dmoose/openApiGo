package markdown

import (
	"os"
	"strings"
	"testing"
)

const minimalSwagger2JSON = `{
  "swagger": "2.0",
  "info": {
    "title": "Minimal API",
    "version": "1.0.0"
  },
  "paths": {
    "/ping": {
      "get": {
        "summary": "Ping",
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  }
}`

// swagger2NoInfoJSON is a minimal Swagger 2.0 document without an info section.
// It previously triggered a nil-pointer panic; this is a regression test to ensure
// we handle missing info gracefully.
const swagger2NoInfoJSON = `{
  "swagger": "2.0",
  "paths": {
    "/ping": {
      "get": {
        "responses": {
          "200": { "description": "ok" }
        }
      }
    }
  }
}`

func TestToMarkdown_MinimalSwagger2(t *testing.T) {
	md, err := ToMarkdown([]byte(minimalSwagger2JSON), Options{Format: FormatJSON})
	if err != nil {
		t.Fatalf("ToMarkdown returned error: %v", err)
	}
	if md == "" {
		t.Fatalf("expected non-empty markdown output")
	}
	if got, want := md[:1], "#"; got != want {
		t.Fatalf("expected output to start with %q, got %q", want, got)
	}
}

func TestSwagger2_Examples_Rendering(t *testing.T) {
	data, err := os.ReadFile("testdata/v2.examples.json")
	if err != nil {
		t.Fatalf("failed to read v2.examples.json: %v", err)
	}
	md, err := ToMarkdown(data, Options{Format: FormatJSON})
	if err != nil {
		t.Fatalf("ToMarkdown(v2.examples.json) returned error: %v", err)
	}
	if !strings.Contains(md, "Request example (application/json)") {
		t.Fatalf("expected markdown to include a labeled Request example for application/json")
	}
	if !strings.Contains(md, "Response example (200, application/json)") {
		t.Fatalf("expected markdown to include a labeled Response example for 200 application/json")
	}
	if !strings.Contains(md, "## Schemas") || !strings.Contains(md, "Example") {
		t.Fatalf("expected markdown to include schema example section")
	}
}

func TestOpenAPI3_Examples_Rendering(t *testing.T) {
	data, err := os.ReadFile("testdata/v3.examples.json")
	if err != nil {
		t.Fatalf("failed to read v3.examples.json: %v", err)
	}
	md, err := ToMarkdown(data, Options{Format: FormatJSON})
	if err != nil {
		t.Fatalf("ToMarkdown(v3.examples.json) returned error: %v", err)
	}
	t.Logf("\n--- v3.examples.md ---\n%s\n--- end ---\n", md)
	if !strings.Contains(md, "Request example (application/json)") {
		t.Fatalf("expected markdown to include a labeled Request example for application/json")
	}
	if !strings.Contains(md, "Request example (alt, application/json)") {
		t.Fatalf("expected markdown to include a named Request example 'alt'")
	}
	if !strings.Contains(md, "Response example (200, application/json)") {
		t.Fatalf("expected markdown to include a labeled Response example for 200 application/json")
	}
	if !strings.Contains(md, "Response example (alt, 200, application/json)") {
		t.Fatalf("expected markdown to include a named Response example 'alt'")
	}
	if !strings.Contains(md, "## Schemas") || !strings.Contains(md, "Example") {
		t.Fatalf("expected markdown to include schema example section")
	}
}

func TestToMarkdown_Swagger2_NoInfo(t *testing.T) {
	md, err := ToMarkdown([]byte(swagger2NoInfoJSON), Options{Format: FormatJSON})
	if err != nil {
		t.Fatalf("ToMarkdown(swagger2NoInfoJSON) returned error: %v", err)
	}
	if md == "" {
		t.Fatalf("expected non-empty markdown output for swagger2NoInfoJSON")
	}
	if got, want := md[:1], "#"; got != want {
		t.Fatalf("expected output to start with %q, got %q", want, got)
	}
}

func TestToMarkdown_V2Fixture_JSON(t *testing.T) {
	data, err := os.ReadFile("testdata/v2.json")
	if err != nil {
		t.Fatalf("failed to read v2.json: %v", err)
	}

	md, err := ToMarkdown(data, Options{Format: FormatJSON})
	if err != nil {
		t.Fatalf("ToMarkdown(v2.json) returned error: %v", err)
	}
	if md == "" {
		t.Fatalf("expected non-empty markdown output for v2.json")
	}
	if !strings.Contains(md, "Mini Store API (v2)") {
		t.Fatalf("expected markdown to mention v2 title, got: %s", md[:min(80, len(md))])
	}
	// Verify that at least one response line shows a named schema from a $ref.
	if !strings.Contains(md, "schema: PetList") {
		t.Fatalf("expected markdown to mention PetList schema in a response, got: %s", md[:min(200, len(md))])
	}
	// Verify that produces section is rendered for at least one operation.
	if !strings.Contains(md, "**Produces**") {
		t.Fatalf("expected markdown to include a Produces section")
	}
	// Verify that schema properties include enums (status enum in Pet schema).
	if !strings.Contains(md, "status` (string") || !strings.Contains(md, "[enum:") {
		t.Fatalf("expected markdown schemas to include enum details for properties, got: %s", md[:min(300, len(md))])
	}
}

func TestToMarkdown_V3Fixture_JSON(t *testing.T) {
	data, err := os.ReadFile("testdata/v3.json")
	if err != nil {
		t.Fatalf("failed to read v3.json: %v", err)
	}

	md, err := ToMarkdown(data, Options{Format: FormatJSON})
	if err != nil {
		t.Fatalf("ToMarkdown(v3.json) returned error: %v", err)
	}
	if md == "" {
		t.Fatalf("expected non-empty markdown output for v3.json")
	}
	if !strings.Contains(md, "Mini Store API (v3)") {
		t.Fatalf("expected markdown to mention v3 title, got: %s", md[:min(80, len(md))])
	}
}

func TestToMarkdown_V3Fixture_YAML(t *testing.T) {
	data, err := os.ReadFile("testdata/v3.yaml")
	if err != nil {
		t.Fatalf("failed to read v3.yaml: %v", err)
	}

	md, err := ToMarkdown(data, Options{Format: FormatAuto})
	if err != nil {
		t.Fatalf("ToMarkdown(v3.yaml) returned error: %v", err)
	}
	if md == "" {
		t.Fatalf("expected non-empty markdown output for v3.yaml")
	}
	if !strings.Contains(md, "Mini Store API (v3)") {
		t.Fatalf("expected markdown to mention v3 title, got: %s", md[:min(80, len(md))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
