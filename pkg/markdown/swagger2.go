package markdown

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/go-openapi/spec"
)

// Swagger 2.0 (OpenAPI 2.0) markdown generation.

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
			// Schema example (standard or vendor)
			if sch.Example != nil {
				writeExampleFence(&b, "Example", "application/json", sch.Example)
			} else if v, ok := sch.VendorExtensible.Extensions["x-example"]; ok {
				writeExampleFence(&b, "Example", "application/json", v)
			}
		}
	}

	// Examples (basic)
	fmt.Fprintf(&b, "\n## Examples\n")
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

	// Request example (Swagger 2.0: body parameter schema.example)
	var bodySchema *spec.Schema
	for _, prm := range op.Parameters {
		if prm.In == "body" && prm.Schema != nil {
			bodySchema = prm.Schema
			break
		}
	}
	if bodySchema != nil {
		// Prefer schema-level Example if present, else look at items when array.
		var ex any
		if bodySchema.Example != nil {
			ex = bodySchema.Example
		} else if bodySchema.Items != nil && bodySchema.Items.Schema != nil && bodySchema.Items.Schema.Example != nil {
			ex = bodySchema.Items.Schema.Example
		}
		// Vendor extensions: x-example or x-examples
		if ex == nil {
			if v, ok := bodySchema.VendorExtensible.Extensions["x-example"]; ok {
				ex = v
			}
		}
		if ex != nil {
			if len(consumes) > 0 {
				for _, mt := range consumes {
					writeExampleFence(b, "Request example ("+mt+")", mt, ex)
				}
			} else {
				writeExampleFence(b, "Request example", "", ex)
			}
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

			// Render response examples by media type if present.
			if len(r.Examples) > 0 {
				var mts []string
				for mt := range r.Examples {
					mts = append(mts, mt)
				}
				sort.Strings(mts)
				for _, mt := range mts {
					writeExampleFence(b, fmt.Sprintf("Response example (%d, %s)", code, mt), mt, r.Examples[mt])
				}
			}
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
