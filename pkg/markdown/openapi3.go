package markdown

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// OpenAPI 3.x markdown generation.

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
	if len(doc.Components.SecuritySchemes) == 0 {
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

	if doc.Paths == nil {
		fmt.Fprintf(&b, "- None defined\n")
	} else {
		pathMap := doc.Paths.Map()
		pathKeys := make([]string, 0, len(pathMap))
		for p := range pathMap {
			pathKeys = append(pathKeys, p)
		}
		sort.Strings(pathKeys)

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
			ops := []struct {
				method string
				op     *openapi3.Operation
			}{
				{"GET", pi.Get}, {"POST", pi.Post}, {"PUT", pi.Put}, {"DELETE", pi.Delete},
				{"PATCH", pi.Patch}, {"OPTIONS", pi.Options}, {"HEAD", pi.Head}, {"TRACE", pi.Trace},
			}
			for _, it := range ops {
				if it.op == nil {
					continue
				}
				ref := opRef{Method: it.method, Path: p, PathItem: pi, Op: it.op}
				if len(it.op.Tags) == 0 {
					untagged = append(untagged, ref)
					continue
				}
				for _, tag := range it.op.Tags {
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
	if len(doc.Components.Schemas) > 0 {
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
				// Schema example
				if ref.Value.Example != nil {
					writeExampleFence(&b, "Example", "application/json", ref.Value.Example)
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

		for _, p := range pathKeys {
			pi := pathMap[p]
			ops := []struct {
				method string
				op     *openapi3.Operation
			}{
				{"GET", pi.Get}, {"POST", pi.Post}, {"PUT", pi.Put}, {"DELETE", pi.Delete},
				{"PATCH", pi.Patch}, {"OPTIONS", pi.Options}, {"HEAD", pi.Head}, {"TRACE", pi.Trace},
			}
			for _, it := range ops {
				op := it.op
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
						fmt.Fprintf(&b, "- %s %s %s — has inline examples\n", it.method, p, code)
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
		// Stable order of media types
		var mts []string
		for mt := range op.RequestBody.Value.Content {
			mts = append(mts, mt)
		}
		sort.Strings(mts)
		for _, mt := range mts {
			media := op.RequestBody.Value.Content[mt]
			typ := "-"
			if media.Schema != nil && media.Schema.Value != nil {
				typ = typeOfSchemaRef(media.Schema)
			}
			fmt.Fprintf(b, "- %s — schema: %s\n", mt, typ)
			// Examples: inline example or named examples
			if media.Example != nil {
				writeExampleFence(b, "Request example ("+mt+")", mt, media.Example)
			}
			if len(media.Examples) > 0 {
				var exNames []string
				for name := range media.Examples {
					exNames = append(exNames, name)
				}
				sort.Strings(exNames)
				for _, name := range exNames {
					exRef := media.Examples[name]
					if exRef != nil && exRef.Value != nil && exRef.Value.Value != nil {
						writeExampleFence(b, fmt.Sprintf("Request example (%s, %s)", name, mt), mt, exRef.Value.Value)
					}
				}
			}
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
					// Stable order of media types
					var mts []string
					for mt := range r.Value.Content {
						mts = append(mts, mt)
					}
					sort.Strings(mts)
					for _, mt := range mts {
						media := r.Value.Content[mt]
						typ := "-"
						if media.Schema != nil && media.Schema.Value != nil {
							typ = typeOfSchemaRef(media.Schema)
						}
						fmt.Fprintf(b, "  - %s — schema: %s\n", mt, typ)
						// Examples per media type
						if media.Example != nil {
							writeExampleFence(b, fmt.Sprintf("Response example (%s, %s)", code, mt), mt, media.Example)
						}
						if len(media.Examples) > 0 {
							var exNames []string
							for name := range media.Examples {
								exNames = append(exNames, name)
							}
							sort.Strings(exNames)
							for _, name := range exNames {
								exRef := media.Examples[name]
								if exRef != nil && exRef.Value != nil && exRef.Value.Value != nil {
									writeExampleFence(b, fmt.Sprintf("Response example (%s, %s, %s)", name, code, mt), mt, exRef.Value.Value)
								}
							}
						}
					}
				}
			}
		}
	}
}
