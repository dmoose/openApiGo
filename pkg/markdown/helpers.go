package markdown

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/spec"
)

// Shared helpers across Swagger 2.0 and OpenAPI 3.x markdown generation.

// nonEmpty returns s if it is non-empty, otherwise fallback.
func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func defaultAsString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func enumAsString(list []any) string {
	if len(list) == 0 {
		return ""
	}
	var parts []string
	for _, v := range list {
		parts = append(parts, fmt.Sprintf("%v", v))
	}
	return strings.Join(parts, ", ")
}

// schemaSummarySwagger2 returns a concise description of a Swagger 2.0 schema
// suitable for inline use in response summaries.
func schemaSummarySwagger2(s *spec.Schema) string {
	if s == nil {
		return ""
	}
	// Prefer $ref if present.
	if ref := s.Ref.String(); ref != "" {
		if name := refName(ref); name != "" {
			return name
		}
	}
	// Handle arrays with item refs or simple types.
	if len(s.Type) == 1 && s.Type[0] == "array" && s.Items != nil {
		if s.Items.Schema != nil {
			if ref := s.Items.Schema.Ref.String(); ref != "" {
				if name := refName(ref); name != "" {
					return name + "[]"
				}
			}
			if len(s.Items.Schema.Type) > 0 {
				return fmt.Sprintf("array<%s>", strings.Join(s.Items.Schema.Type, ","))
			}
		}
		return "array"
	}
	if len(s.Type) > 0 {
		if s.Format != "" {
			return fmt.Sprintf("%s (%s)", strings.Join(s.Type, ","), s.Format)
		}
		return strings.Join(s.Type, ",")
	}
	return ""
}

func refName(ref string) string {
	if ref == "" {
		return ""
	}
	i := strings.LastIndex(ref, "/")
	if i >= 0 && i+1 < len(ref) {
		return ref[i+1:]
	}
	return ref
}

func hostURL(schemes []string, host, basePath string) string {
	s := "http"
	if len(schemes) > 0 {
		s = schemes[0]
	}
	if host == "" {
		return ""
	}
	if basePath == "" {
		return fmt.Sprintf("%s://%s", s, host)
	}
	return fmt.Sprintf("%s://%s%s", s, host, basePath)
}

func typeOfSchemaRef(ref *openapi3.SchemaRef) string {
	if ref == nil || ref.Value == nil {
		return "-"
	}
	// Prefer $ref if present.
	if ref.Ref != "" {
		i := strings.LastIndex(ref.Ref, "/")
		if i >= 0 && i+1 < len(ref.Ref) {
			return fmt.Sprintf("$ref:%s", ref.Ref[i+1:])
		}
		return fmt.Sprintf("$ref:%s", ref.Ref)
	}
	// Handle arrays with item refs or simple types.
	if ref.Value.Type != nil && len(*ref.Value.Type) == 1 && (*ref.Value.Type)[0] == "array" && ref.Value.Items != nil {
		if ref.Value.Items.Ref != "" {
			name := refName(ref.Value.Items.Ref)
			if name != "" {
				return fmt.Sprintf("%s[]", name)
			}
		}
		if ref.Value.Items.Value != nil && ref.Value.Items.Value.Type != nil && len(*ref.Value.Items.Value.Type) > 0 {
			return fmt.Sprintf("array<%s>", strings.Join(*ref.Value.Items.Value.Type, ","))
		}
		return "array"
	}
	// Fall back to the declared types if available.
	if ref.Value.Type != nil && len(*ref.Value.Type) > 0 {
		return strings.Join(*ref.Value.Type, ",")
	}
	return "object"
}
