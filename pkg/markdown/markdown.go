package markdown

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/spec"
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

// ---------------------------
// Swagger 2.0 (OpenAPI 2.0)
// ---------------------------

func swagger2ToMarkdown(data []byte) (md string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("swagger2 conversion panic: %v", r)
			md = ""
		}
	}()

	var s spec.Swagger
	if err := json.Unmarshal(data, &s); err != nil {
		return "", fmt.Errorf("parse swagger 2.0: %w", err)
	}

	var b bytes.Buffer

	// Overview
	title := "-"
	version := "-"
	if s.Info != nil {
		if s.Info.Title != "" {
			title = s.Info.Title
		}
		if s.Info.Version != "" {
			version = s.Info.Version
		}
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "## Overview\n")
	fmt.Fprintf(&b, "- Version: %s\n", version)
	if s.Info != nil && s.Info.Description != "" {
		fmt.Fprintf(&b, "- Description: %s\n", strings.TrimSpace(s.Info.Description))
	}
	if s.Info != nil && s.Info.Contact != nil {
		if s.Info.Contact.Name != "" {
			fmt.Fprintf(&b, "- Contact: %s\n", s.Info.Contact.Name)
		}
		if s.Info.Contact.Email != "" {
			fmt.Fprintf(&b, "- Contact Email: %s\n", s.Info.Contact.Email)
		}
	}
	if s.Info != nil && s.Info.License != nil && s.Info.License.Name != "" {
		fmt.Fprintf(&b, "- License: %s\n", s.Info.License.Name)
	}

	// Authentication
	fmt.Fprintf(&b, "\n## Authentication\n")
	if len(s.SecurityDefinitions) == 0 {
		fmt.Fprintf(&b, "- None defined\n")
	} else {
		for name, sec := range s.SecurityDefinitions {
			line := fmt.Sprintf("- %s — type=%s", name, sec.Type)
			if sec.Name != "" {
				line += fmt.Sprintf(", name=%s", sec.Name)
			}
			if sec.In != "" {
				line += fmt.Sprintf(", in=%s", sec.In)
			}
			if sec.AuthorizationURL != "" {
				line += fmt.Sprintf(", authUrl=%s", sec.AuthorizationURL)
			}
			if sec.TokenURL != "" {
				line += fmt.Sprintf(", tokenUrl=%s", sec.TokenURL)
			}
			if len(sec.Scopes) > 0 {
				var scopes []string
				for k, v := range sec.Scopes {
					if v != "" {
						scopes = append(scopes, fmt.Sprintf("%s (%s)", k, v))
					} else {
						scopes = append(scopes, k)
					}
				}
				sort.Strings(scopes)
				line += fmt.Sprintf(", scopes=[%s]", strings.Join(scopes, ", "))
			}
			fmt.Fprintln(&b, line)
		}
	}

	// Servers
	fmt.Fprintf(&b, "\n## Servers\n")
	hostLine := hostURL(s.Schemes, s.Host, s.BasePath)
	if hostLine != "" {
		fmt.Fprintf(&b, "- %s\n", hostLine)
	} else {
		fmt.Fprintf(&b, "- None defined\n")
	}

	// Tags
	fmt.Fprintf(&b, "\n## Tags\n")
	if len(s.Tags) == 0 {
		fmt.Fprintf(&b, "- None defined\n")
	} else {
		for _, t := range s.Tags {
			if t.Description != "" {
				fmt.Fprintf(&b, "- %s — %s\n", t.Name, t.Description)
			} else {
				fmt.Fprintf(&b, "- %s\n", t.Name)
			}
		}
	}

	// Endpoints by Tag
	fmt.Fprintf(&b, "\n## Endpoints by Tag\n")
	// Build tag -> operations map, plus untagged.
	type opRef struct {
		Method string
		Path   string
		Op     *spec.Operation
	}
	tagged := map[string][]opRef{}
	untagged := []opRef{}

	paths := make([]string, 0, len(s.Paths.Paths))
	for p := range s.Paths.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		pi := s.Paths.Paths[p]
		ops := []struct {
			method string
			op     *spec.Operation
		}{
			{"GET", pi.Get}, {"POST", pi.Post}, {"PUT", pi.Put}, {"DELETE", pi.Delete},
			{"PATCH", pi.Patch}, {"OPTIONS", pi.Options}, {"HEAD", pi.Head},
		}
		for _, it := range ops {
			if it.op == nil {
				continue
			}
			ref := opRef{Method: it.method, Path: p, Op: it.op}
			if len(it.op.Tags) == 0 {
				untagged = append(untagged, ref)
				continue
			}
			for _, tag := range it.op.Tags {
				tagged[tag] = append(tagged[tag], ref)
			}
		}
	}

	// Print tagged endpoints in tag name order.
	tagNames := make([]string, 0, len(tagged))
	for name := range tagged {
		tagNames = append(tagNames, name)
	}
	sort.Strings(tagNames)
	for _, name := range tagNames {
		fmt.Fprintf(&b, "\n### %s\n", name)
		for _, ref := range tagged[name] {
			writeSwagger2Operation(&b, ref.Method, ref.Path, ref.Op, s.Produces, s.Consumes)
		}
	}

	// Untagged endpoints.
	if len(untagged) > 0 {
		fmt.Fprintf(&b, "\n### Untagged\n")
		for _, ref := range untagged {
			writeSwagger2Operation(&b, ref.Method, ref.Path, ref.Op, s.Produces, s.Consumes)
		}
	}

	// Schemas (Definitions)
	if len(s.Definitions) > 0 {
		fmt.Fprintf(&b, "\n## Schemas\n")
		names := make([]string, 0, len(s.Definitions))
		for name := range s.Definitions {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			sch := s.Definitions[name]
			fmt.Fprintf(&b, "\n### %s\n", name)
			if sch.Description != "" {
				fmt.Fprintf(&b, "%s\n\n", sch.Description)
			}
			if len(sch.Properties) > 0 {
				fmt.Fprintf(&b, "**Properties**\n")
				propNames := make([]string, 0, len(sch.Properties))
				for pn := range sch.Properties {
					propNames = append(propNames, pn)
				}
				sort.Strings(propNames)
				for _, pn := range propNames {
					ps := sch.Properties[pn]
					typ := nonEmpty(schemaSummarySwagger2(&ps), "-")
					desc := strings.TrimSpace(ps.Description)
					req := ""
					if contains(sch.Required, pn) {
						req = " (required)"
					}
					def := defaultAsString(ps.Default)
					enum := enumAsString(ps.Enum)
					line := fmt.Sprintf("- `%s` (%s)%s", pn, typ, req)
					if desc != "" {
						line += fmt.Sprintf(" — %s", desc)
					}
					if def != "" {
						line += fmt.Sprintf(" [default: %s]", def)
					}
					if enum != "" {
						line += fmt.Sprintf(" [enum: %s]", enum)
					}
					fmt.Fprintln(&b, line)
				}
			}
		}
	}

	// Examples (basic):
	fmt.Fprintf(&b, "\n## Examples\n")
	// For v1, we just note where response examples exist.
	for _, p := range paths {
		pi := s.Paths.Paths[p]
		ops := []struct {
			method string
			op     *spec.Operation
		}{
			{"GET", pi.Get}, {"POST", pi.Post}, {"PUT", pi.Put}, {"DELETE", pi.Delete},
			{"PATCH", pi.Patch}, {"OPTIONS", pi.Options}, {"HEAD", pi.Head},
		}
		for _, it := range ops {
			if it.op == nil || it.op.Responses == nil {
				continue
			}
			for code, r := range it.op.Responses.StatusCodeResponses {
				if len(r.Headers) == 0 && r.Schema == nil && len(r.Examples) == 0 {
					continue
				}
				if len(r.Examples) > 0 {
					fmt.Fprintf(&b, "- %s %s %d — has inline examples\n", it.method, p, code)
				}
			}
		}
	}

	return b.String(), nil
}

func writeSwagger2Operation(b *bytes.Buffer, method, path string, op *spec.Operation, globalProduces, globalConsumes []string) {
	fmt.Fprintf(b, "\n#### %s %s\n", method, path)
	if op.Summary != "" {
		fmt.Fprintf(b, "%s\n\n", op.Summary)
	}
	if op.Description != "" {
		fmt.Fprintf(b, "%s\n\n", op.Description)
	}

	// Operation ID
	if op.ID != "" {
		fmt.Fprintf(b, "_Operation ID_: `%s`\n\n", op.ID)
	}

	// Media types
	produces := op.Produces
	if len(produces) == 0 {
		produces = globalProduces
	}
	consumes := op.Consumes
	if len(consumes) == 0 {
		consumes = globalConsumes
	}
	if len(produces) > 0 {
		fmt.Fprintf(b, "**Produces**\n")
		for _, mt := range produces {
			fmt.Fprintf(b, "- %s\n", mt)
		}
		fmt.Fprintln(b)
	}
	if len(consumes) > 0 {
		fmt.Fprintf(b, "**Consumes**\n")
		for _, mt := range consumes {
			fmt.Fprintf(b, "- %s\n", mt)
		}
		fmt.Fprintln(b)
	}

	// Parameters
	if len(op.Parameters) > 0 {
		fmt.Fprintf(b, "**Parameters**\n")
		for _, prm := range op.Parameters {
			loc, name := prm.In, prm.Name
			req := ""
			if prm.Required {
				req = " (required)"
			}
			typ := prm.Type
			if typ == "" && prm.Schema != nil && len(prm.Schema.Type) > 0 {
				typ = strings.Join(prm.Schema.Type, ",")
			}
			desc := strings.TrimSpace(prm.Description)
			def := defaultAsString(prm.Default)
			enum := enumAsString(prm.Enum)

			line := fmt.Sprintf("- %s `%s` (%s)%s", loc, name, nonEmpty(typ, "-"), req)
			if desc != "" {
				line += fmt.Sprintf(" — %s", desc)
			}
			if def != "" {
				line += fmt.Sprintf(" [default: %s]", def)
			}
			if enum != "" {
				line += fmt.Sprintf(" [enum: %s]", enum)
			}
			fmt.Fprintln(b, line)
		}
	}

	// Responses
	if op.Responses != nil && (len(op.Responses.StatusCodeResponses) > 0 || op.Responses.Default != nil) {
		fmt.Fprintf(b, "\n**Responses**\n")
		var codes []int
		for code := range op.Responses.StatusCodeResponses {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		for _, code := range codes {
			r := op.Responses.StatusCodeResponses[code]
			desc := strings.TrimSpace(r.Description)
			if desc == "" {
				desc = "No description"
			}
			line := fmt.Sprintf("- %d — %s", code, desc)
			if r.Schema != nil {
				if summary := schemaSummarySwagger2(r.Schema); summary != "" {
					line += fmt.Sprintf(" (schema: %s)", summary)
				}
			}
			fmt.Fprintln(b, line)
		}
		if op.Responses.Default != nil {
			desc := strings.TrimSpace(op.Responses.Default.Description)
			if desc == "" {
				desc = "No description"
			}
			line := fmt.Sprintf("- default — %s", desc)
			if op.Responses.Default.Schema != nil {
				if summary := schemaSummarySwagger2(op.Responses.Default.Schema); summary != "" {
					line += fmt.Sprintf(" (schema: %s)", summary)
				}
			}
			fmt.Fprintln(b, line)
		}
	}
}

// ---------------------------
// OpenAPI 3.x
// ---------------------------

func openAPI3ToMarkdown(data []byte, opts Options) (md string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("openapi3 conversion panic: %v", r)
			md = ""
		}
	}()

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return "", fmt.Errorf("parse openapi 3: %w", err)
	}
	if doc == nil {
		return "", fmt.Errorf("parse openapi 3: loader returned nil document")
	}
	if !opts.SkipValidation {
		// Validation is optional; do not fail hard on validation errors.
		_ = doc.Validate(context.Background())
	}

	var b bytes.Buffer

	// Overview
	title := "-"
	desc := ""
	version := "-"
	if doc.Info != nil {
		if doc.Info.Title != "" {
			title = doc.Info.Title
		}
		if doc.Info.Description != "" {
			desc = strings.TrimSpace(doc.Info.Description)
		}
	}
	if doc.Info != nil && doc.Info.Version != "" {
		version = doc.Info.Version
	} else if doc.OpenAPI != "" {
		version = doc.OpenAPI
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "## Overview\n")
	fmt.Fprintf(&b, "- Version: %s\n", version)
	if desc != "" {
		fmt.Fprintf(&b, "- Description: %s\n", desc)
	}
	if doc.Info != nil && doc.Info.Contact != nil {
		if doc.Info.Contact.Name != "" {
			fmt.Fprintf(&b, "- Contact: %s\n", doc.Info.Contact.Name)
		}
		if doc.Info.Contact.Email != "" {
			fmt.Fprintf(&b, "- Contact Email: %s\n", doc.Info.Contact.Email)
		}
	}
	if doc.Info != nil && doc.Info.License != nil && doc.Info.License.Name != "" {
		fmt.Fprintf(&b, "- License: %s\n", doc.Info.License.Name)
	}

	// Authentication (security schemes)
	fmt.Fprintf(&b, "\n## Authentication\n")
	if doc.Components.SecuritySchemes == nil || len(doc.Components.SecuritySchemes) == 0 {
		fmt.Fprintf(&b, "- None defined\n")
	} else {
		names := make([]string, 0, len(doc.Components.SecuritySchemes))
		for name := range doc.Components.SecuritySchemes {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			ref := doc.Components.SecuritySchemes[name]
			if ref == nil || ref.Value == nil {
				continue
			}
			ss := ref.Value
			line := fmt.Sprintf("- %s — type=%s", name, ss.Type)
			if ss.Scheme != "" {
				line += fmt.Sprintf(", scheme=%s", ss.Scheme)
			}
			if ss.Name != "" {
				line += fmt.Sprintf(", name=%s", ss.Name)
			}
			if ss.In != "" {
				line += fmt.Sprintf(", in=%s", ss.In)
			}
			fmt.Fprintln(&b, line)
		}
	}

	// Servers
	fmt.Fprintf(&b, "\n## Servers\n")
	if len(doc.Servers) == 0 {
		fmt.Fprintf(&b, "- None defined\n")
	} else {
		for _, s := range doc.Servers {
			u := s.URL
			if len(s.Variables) > 0 {
				u += " {vars}"
			}
			fmt.Fprintf(&b, "- %s\n", u)
		}
	}

	// Tags
	fmt.Fprintf(&b, "\n## Tags\n")
	if len(doc.Tags) == 0 {
		fmt.Fprintf(&b, "- None defined\n")
	} else {
		for _, t := range doc.Tags {
			if t.Description != "" {
				fmt.Fprintf(&b, "- %s — %s\n", t.Name, t.Description)
			} else {
				fmt.Fprintf(&b, "- %s\n", t.Name)
			}
		}
	}

	// Endpoints by Tag
	fmt.Fprintf(&b, "\n## Endpoints by Tag\n")

	var (
		pathMap  map[string]*openapi3.PathItem
		pathKeys []string
	)

	if doc.Paths == nil {
		fmt.Fprintf(&b, "- None defined\n")
	} else {
		pathMap = doc.Paths.Map()
		pathKeys = make([]string, 0, len(pathMap))
		for p := range pathMap {
			pathKeys = append(pathKeys, p)
		}
		sort.Strings(pathKeys)

		methodOrder := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD", "TRACE"}

		type opRef struct {
			Method   string
			Path     string
			PathItem *openapi3.PathItem
			Op       *openapi3.Operation
		}
		tagged := map[string][]opRef{}
		untagged := []opRef{}

		for _, p := range pathKeys {
			pi := pathMap[p]
			opsMap := pi.Operations()
			for _, m := range methodOrder {
				op := opsMap[strings.ToLower(m)]
				if op == nil {
					continue
				}
				ref := opRef{Method: m, Path: p, PathItem: pi, Op: op}
				if len(op.Tags) == 0 {
					untagged = append(untagged, ref)
					continue
				}
				for _, tag := range op.Tags {
					tagged[tag] = append(tagged[tag], ref)
				}
			}
		}

		tagNames := make([]string, 0, len(tagged))
		for name := range tagged {
			tagNames = append(tagNames, name)
		}
		sort.Strings(tagNames)
		for _, name := range tagNames {
			fmt.Fprintf(&b, "\n### %s\n", name)
			for _, ref := range tagged[name] {
				writeOpenAPI3Operation(&b, ref.Method, ref.Path, ref.PathItem, ref.Op)
			}
		}

		if len(untagged) > 0 {
			fmt.Fprintf(&b, "\n### Untagged\n")
			for _, ref := range untagged {
				writeOpenAPI3Operation(&b, ref.Method, ref.Path, ref.PathItem, ref.Op)
			}
		}
	}

	// Schemas
	if doc.Components.Schemas != nil && len(doc.Components.Schemas) > 0 {
		fmt.Fprintf(&b, "\n## Schemas\n")
		names := make([]string, 0, len(doc.Components.Schemas))
		for name := range doc.Components.Schemas {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			ref := doc.Components.Schemas[name]
			fmt.Fprintf(&b, "\n### %s\n", name)
			if ref.Value != nil {
				if ref.Value.Description != "" {
					fmt.Fprintf(&b, "%s\n\n", ref.Value.Description)
				}
				if len(ref.Value.Properties) > 0 {
					fmt.Fprintf(&b, "**Properties**\n")
					var propNames []string
					for pn := range ref.Value.Properties {
						propNames = append(propNames, pn)
					}
					sort.Strings(propNames)
					for _, pn := range propNames {
						ps := ref.Value.Properties[pn]
						typ := typeOfSchemaRef(ps)
						desc := ""
						def := ""
						enum := ""
						if ps.Value != nil {
							desc = strings.TrimSpace(ps.Value.Description)
							if ps.Value.Default != nil {
								def = fmt.Sprintf("%v", ps.Value.Default)
							}
							if len(ps.Value.Enum) > 0 {
								parts := make([]string, 0, len(ps.Value.Enum))
								for _, v := range ps.Value.Enum {
									parts = append(parts, fmt.Sprintf("%v", v))
								}
								enum = strings.Join(parts, ", ")
							}
						}
						req := ""
						if contains(ref.Value.Required, pn) {
							req = " (required)"
						}
						line := fmt.Sprintf("- `%s` (%s)%s", pn, typ, req)
						if desc != "" {
							line += fmt.Sprintf(" — %s", desc)
						}
						if def != "" {
							line += fmt.Sprintf(" [default: %s]", def)
						}
						if enum != "" {
							line += fmt.Sprintf(" [enum: %s]", enum)
						}
						fmt.Fprintln(&b, line)
					}
				}
			}
		}
	}

	// Examples (basic): note where response content examples exist.
	fmt.Fprintf(&b, "\n## Examples\n")
	if doc.Paths == nil {
		fmt.Fprintf(&b, "- None defined\n")
	} else {
		pathMap := doc.Paths.Map()
		pathKeys := make([]string, 0, len(pathMap))
		for p := range pathMap {
			pathKeys = append(pathKeys, p)
		}
		sort.Strings(pathKeys)

		methodOrder := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD", "TRACE"}

		for _, p := range pathKeys {
			pi := pathMap[p]
			opsMap := pi.Operations()
			for _, m := range methodOrder {
				op := opsMap[strings.ToLower(m)]
				if op == nil || op.Responses == nil {
					continue
				}
				respMap := op.Responses.Map()
				for code, r := range respMap {
					if r == nil || r.Value == nil {
						continue
					}
					if len(r.Value.Content) == 0 {
						continue
					}
					// If any media type has an example, mention it.
					hasExample := false
					for _, media := range r.Value.Content {
						if media.Example != nil || len(media.Examples) > 0 {
							hasExample = true
							break
						}
					}
					if hasExample {
						fmt.Fprintf(&b, "- %s %s %s — has inline examples\n", m, p, code)
					}
				}
			}
		}
	}

	return b.String(), nil
}

func writeOpenAPI3Operation(b *bytes.Buffer, method, path string, pi *openapi3.PathItem, op *openapi3.Operation) {
	fmt.Fprintf(b, "\n#### %s %s\n", method, path)
	if op.Summary != "" {
		fmt.Fprintf(b, "%s\n\n", op.Summary)
	}
	if op.Description != "" {
		fmt.Fprintf(b, "%s\n\n", op.Description)
	}

	// Parameters (PathItem + Operation)
	params := append([]*openapi3.ParameterRef{}, pi.Parameters...)
	params = append(params, op.Parameters...)
	if len(params) > 0 {
		fmt.Fprintf(b, "**Parameters**\n")
		for _, pr := range params {
			if pr == nil || pr.Value == nil {
				continue
			}
			par := pr.Value
			req := ""
			if par.Required {
				req = " (required)"
			}
			typ := "-"
			if par.Schema != nil && par.Schema.Value != nil {
				typ = typeOfSchemaRef(par.Schema)
			}
			desc := strings.TrimSpace(par.Description)
			def := ""
			if par.Schema != nil && par.Schema.Value != nil && par.Schema.Value.Default != nil {
				def = fmt.Sprintf("%v", par.Schema.Value.Default)
			}
			line := fmt.Sprintf("- %s `%s` (%s)%s", par.In, par.Name, typ, req)
			if desc != "" {
				line += fmt.Sprintf(" — %s", desc)
			}
			if def != "" {
				line += fmt.Sprintf(" [default: %s]", def)
			}
			fmt.Fprintln(b, line)
		}
	}

	// Request Body
	if op.RequestBody != nil && op.RequestBody.Value != nil && len(op.RequestBody.Value.Content) > 0 {
		fmt.Fprintf(b, "\n**Request Body**\n")
		for mt, media := range op.RequestBody.Value.Content {
			typ := "-"
			if media.Schema != nil && media.Schema.Value != nil {
				typ = typeOfSchemaRef(media.Schema)
			}
			fmt.Fprintf(b, "- %s — schema: %s\n", mt, typ)
		}
	}

	// Responses
	if op.Responses != nil {
		respMap := op.Responses.Map()
		if len(respMap) > 0 {
			fmt.Fprintf(b, "\n**Responses**\n")
			var codes []string
			for code := range respMap {
				codes = append(codes, code)
			}
			sort.Strings(codes)
				for _, code := range codes {
					r := respMap[code]
					if r == nil || r.Value == nil {
						continue
					}
					desc := "No description"
					if r.Value.Description != nil {
						desc = strings.TrimSpace(*r.Value.Description)
					}
					if desc == "" {
						desc = "No description"
					}
					fmt.Fprintf(b, "- %s — %s\n", code, desc)
				if len(r.Value.Content) > 0 {
					for mt, media := range r.Value.Content {
						typ := "-"
						if media.Schema != nil && media.Schema.Value != nil {
							typ = typeOfSchemaRef(media.Schema)
						}
						fmt.Fprintf(b, "  - %s — schema: %s\n", mt, typ)
					}
				}
			}
		}
	}
}

// ---------------------------
// Helpers
// ---------------------------

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

func defaultAsString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func enumAsString(list []interface{}) string {
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
