package markdown

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type InputFormat string

const (
	FormatAuto InputFormat = "auto"
	FormatJSON InputFormat = "json"
	FormatYAML InputFormat = "yaml"
)

type Options struct {
	Format         InputFormat
	SkipValidation bool
}

type versionProbe struct {
	Swagger string `json:"swagger"`
	OpenAPI string `json:"openapi"`
}

// ToMarkdown converts an OpenAPI/Swagger JSON or YAML document to Markdown.
// - Detects version via top-level "swagger" (2.0) or "openapi" (3.x).
// - Supports auto-detection of JSON vs YAML, overridable via Options.Format.
func ToMarkdown(data []byte, opts Options) (string, error) {
	jsonData, err := normalizeToJSON(data, opts.Format)
	if err != nil {
		return "", err
	}

	var vp versionProbe
	if err := json.Unmarshal(jsonData, &vp); err != nil {
		return "", fmt.Errorf("failed to parse input as JSON: %w", err)
	}

	switch {
	case strings.HasPrefix(vp.Swagger, "2.0"):
		return swagger2ToMarkdown(jsonData)
	case strings.HasPrefix(vp.OpenAPI, "3."):
		return openAPI3ToMarkdown(jsonData, opts)
	default:
		// Try 2.0 first, then 3.x as a fallback.
		if md, err := swagger2ToMarkdown(jsonData); err == nil {
			return md, nil
		}
		if md, err := openAPI3ToMarkdown(jsonData, opts); err == nil {
			return md, nil
		}
		return "", fmt.Errorf("could not detect or parse OpenAPI version (swagger=%q, openapi=%q)", vp.Swagger, vp.OpenAPI)
	}
}

// normalizeToJSON ensures we always work with JSON for downstream parsing.
func normalizeToJSON(data []byte, format InputFormat) ([]byte, error) {
	// If the user specified a format, honor it.
	if format == FormatJSON {
		return data, nil
	}

	if format == FormatYAML {
		var v any
		if err := yaml.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("failed to parse input as YAML: %w", err)
		}
		jsonData, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
		}
		return jsonData, nil
	}

	// Auto-detect: try JSON, then YAML.
	var tmp any
	if err := json.Unmarshal(data, &tmp); err == nil {
		return data, nil
	}

	if err := yaml.Unmarshal(data, &tmp); err == nil {
		jsonData, err := json.Marshal(tmp)
		if err != nil {
			return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
		}
		return jsonData, nil
	}

	return nil, fmt.Errorf("input is neither valid JSON nor YAML")
}
