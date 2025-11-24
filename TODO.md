# TODO

Planned surfacing of additional OpenAPI/Swagger schema aspects in Markdown. Items are grouped by priority and risk. Check off as implemented.

## High value, low risk
- [ ] Deprecation flags
  - v2: operation `deprecated`
  - v3: `deprecated` on operations, parameters, and schema properties
  - Render a clear "Deprecated" badge where present
- [ ] Read/Write visibility and nullability
  - v2: property `readOnly`, common vendor `x-nullable`
  - v3: `readOnly`, `writeOnly`, and `nullable` (or JSON Schema null in 3.1)
  - Show next to property types
- [ ] Validation constraints (beyond default/enum)
  - `minimum`/`maximum` (+ `exclusive*`), `minLength`/`maxLength`, `pattern`,
    `minItems`/`maxItems`, `uniqueItems`, `minProperties`/`maxProperties`, `additionalProperties`
  - Render succinctly alongside property
- [ ] Response headers
  - v2/v3: `responses[status].headers`
  - Show header name, type/schema, and description under each response
- [ ] Media types and content negotiation summary
  - v2: `consumes`/`produces` (global/operation)
  - v3: request/response `content` keys
  - Present a compact matrix per operation
- [ ] Security overview and per-operation requirements
  - v2: `securityDefinitions` + `security` (global/op)
  - v3: `components.securitySchemes` + `security` (global/op)
  - Top-level catalog + per-operation listing of required schemes/scopes

## Medium value, medium risk
- [ ] Parameter serialization details
  - v2: `collectionFormat`, `allowEmptyValue`
  - v3: `style`, `explode`, `allowReserved`, parameter `content`
  - Explain array/object encoding briefly; optional example strings
- [ ] Composition and discriminator hints
  - v2: `allOf`, legacy `discriminator`
  - v3: `oneOf`/`anyOf`/`allOf`, `discriminator`
  - Summarize alternatives and show discriminator mapping (no deep merge)
- [ ] Links (v3)
  - `responses[status].links`, `components.links`
  - Show link name, target `operationId`/`operationRef`, and parameter mapping
- [ ] Multipart encoding (v3)
  - `requestBody.content["multipart/form-data"].encoding[*]`
  - Show per-property `contentType`, `headers`, `style`
- [ ] Tags metadata and external docs
  - v2/v3: tag `description`, `externalDocs`; also on operations/components
  - Render descriptions and external links

## Lower value or higher risk
- [ ] Servers and variables enrichment
  - v2: derive from `host`/`basePath`/`schemes`
  - v3: `servers` with `variables`
  - Optionally render resolved example URLs and variable defaults
- [ ] Webhooks (v3.1)
  - `webhooks` map of inbound operations
  - Render like paths section
- [ ] XML metadata
  - v2/v3: property/schema `xml` hints (`name`, `namespace`, `prefix`, `attribute`, `wrapped`)
  - List when present (important for XML-focused APIs)
- [ ] Callbacks (v3)
  - Operation `callbacks`
  - Render callback paths similarly to normal endpoints

## Examples-specific follow-ups
- [ ] Vendor examples parity in v3 (if applicable)
  - Consider honoring common vendor `x-example`/`x-examples` on media/content where seen in the wild
- [ ] Example visibility controls
  - CLI/library options to suppress or collapse examples when generating minimal output

## Testing
- [ ] Add fixtures covering each new surfaced aspect for both v2 and v3 where applicable
- [ ] Extend assertions in `pkg/markdown/markdown_test.go` for each feature
