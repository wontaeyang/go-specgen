package generator

import (
	"maps"
	"slices"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
	"go.yaml.in/yaml/v4"
)

// Generator generates OpenAPI specifications using libopenapi's v3high models.
//
// OpenAPI 3.1 is the baseline. 3.2 is accepted and differs only in the version
// string, so there is no version-conditional code anywhere in this package —
// see newSchema, markNullable, and exclusiveBound for the 3.1 spellings that
// used to be branches.
type Generator struct {
	version string // "3.1" or "3.2"
}

// NewGenerator creates a new generator
func NewGenerator(version string) *Generator {
	return &Generator{version: version}
}

// newSchema returns a schema of the given type. OpenAPI models Type as an array
// because 3.1 uses it to express nullability; see markNullable. Called with no
// type it returns the empty schema, which renders as {} and accepts any value.
func newSchema(types ...string) *base.Schema {
	return &base.Schema{Type: types}
}

// markNullable widens a schema to accept null, which OpenAPI 3.1 spells by
// adding "null" to the type array rather than with 3.0's nullable: true.
//
// A schema with no type is left alone: it already accepts anything, so there is
// nothing to widen.
func markNullable(schema *base.Schema) {
	if len(schema.Type) > 0 {
		schema.Type = append(schema.Type, "null")
	}
}

// exclusiveBound builds an exclusiveMinimum or exclusiveMaximum value. In 3.1
// the keyword carries the bound itself; 3.0 spelled it as a boolean sitting
// next to minimum/maximum, which is why libopenapi models it as a DynamicValue.
func exclusiveBound(value float64) *base.DynamicValue[bool, float64] {
	return &base.DynamicValue[bool, float64]{N: 1, B: value}
}

// Generate generates an OpenAPI spec from a resolved package
func (g *Generator) Generate(pkg *resolver.Package) (*v3.Document, error) {
	doc := &v3.Document{
		Version: g.getOpenAPIVersion(),
		Info:    g.generateInfo(pkg.API),
	}

	if len(pkg.API.Servers) > 0 {
		doc.Servers = g.generateServers(pkg.API.Servers)
	}

	if len(pkg.API.Tags) > 0 {
		doc.Tags = g.generateTags(pkg.API.Tags)
	}

	doc.Paths = g.generatePaths(pkg.Endpoints)
	doc.Components = g.generateComponents(pkg)

	if len(pkg.API.Security) > 0 {
		doc.Security = g.generateSecurity(pkg.API.Security)
	}

	return doc, nil
}

// getOpenAPIVersion returns the version string written into the document.
// Anything other than 3.2 is 3.1; the CLI rejects the rest before we get here.
func (g *Generator) getOpenAPIVersion() string {
	if g.version == "3.2" {
		return "3.2.0"
	}
	return "3.1.0"
}

// RenderYAML renders the document as YAML.
func (g *Generator) RenderYAML(doc *v3.Document) ([]byte, error) {
	return doc.Render()
}

// RenderJSON renders the document as JSON, indented two spaces.
func (g *Generator) RenderJSON(doc *v3.Document) ([]byte, error) {
	return doc.RenderJSON("  ")
}

// generateInfo generates the info section
func (g *Generator) generateInfo(api *resolver.API) *base.Info {
	info := &base.Info{
		Title:   api.Title,
		Version: api.Version,
	}

	if api.Description != "" {
		info.Description = api.Description
	}

	if api.TermsOfService != "" {
		info.TermsOfService = api.TermsOfService
	}

	if api.Contact != nil {
		info.Contact = &base.Contact{
			Name:  api.Contact.Name,
			URL:   api.Contact.URL,
			Email: api.Contact.Email,
		}
	}

	if api.License != nil {
		info.License = &base.License{
			Name: api.License.Name,
			URL:  api.License.URL,
		}
	}

	return info
}

// generateServers generates the servers section
func (g *Generator) generateServers(servers []*resolver.Server) []*v3.Server {
	result := make([]*v3.Server, len(servers))
	for i, server := range servers {
		result[i] = &v3.Server{
			URL:         server.URL,
			Description: server.Description,
		}
	}
	return result
}

// generateTags generates the tags array
func (g *Generator) generateTags(tags []*resolver.Tag) []*base.Tag {
	result := make([]*base.Tag, len(tags))
	for i, tag := range tags {
		result[i] = &base.Tag{
			Name:        tag.Name,
			Description: tag.Description,
		}
	}
	return result
}

// generateComponents generates the components section
func (g *Generator) generateComponents(pkg *resolver.Package) *v3.Components {
	components := &v3.Components{}

	if len(pkg.Schemas) > 0 {
		components.Schemas = g.generateSchemas(pkg.Schemas)
	}

	if len(pkg.API.SecuritySchemes) > 0 {
		components.SecuritySchemes = g.generateSecuritySchemes(pkg.API.SecuritySchemes)
	}

	return components
}

// generateSchemas generates component schemas
func (g *Generator) generateSchemas(schemas map[string]*resolver.Schema) *orderedmap.Map[string, *base.SchemaProxy] {
	result := orderedmap.New[string, *base.SchemaProxy]()

	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		schema := schemas[name]
		// Skip generic schemas - they are templates, not concrete types
		if schema.IsGeneric {
			continue
		}

		result.Set(name, g.generateSchema(schema))
	}

	return result
}

// generateSchema generates a single schema
func (g *Generator) generateSchema(schema *resolver.Schema) *base.SchemaProxy {
	s := newSchema("object")

	if schema.Description != "" {
		s.Description = schema.Description
	}

	setObjectFields(s, schema.Fields, g.generateFieldSchema)

	if schema.Deprecated {
		t := true
		s.Deprecated = &t
	}

	return base.CreateSchemaProxy(s)
}

// generateFieldSchema renders a field.
//
// The field's TypeRef says what shape to emit; there is nothing to discover
// here, and in particular no Go type strings to re-parse. That is the point of
// the shape. The generator used to have two versions of this — one that could
// recognize a schema reference and one that could not — so the same field
// rendered differently depending on which one reached it.
func (g *Generator) generateFieldSchema(field *resolver.Field) *base.SchemaProxy {
	// A $ref is the one shape that cannot simply carry the field's keywords,
	// so it has its own builder.
	if field.Type != nil && field.Type.Shape == resolver.ShapeRef {
		return g.generateRefSchema(refPath(field.Type.Ref), field)
	}

	schema := g.buildTypeSchema(field.Type)
	g.addFieldConstraints(schema, field)
	return base.CreateSchemaProxy(schema)
}

// buildTypeSchema renders a type shape, without any field-level annotations.
//
// It recurses, so every level of a nested container keeps its own type and
// format: map[string][]time.Time is an object whose additionalProperties is an
// array whose items are date-time strings.
func (g *Generator) buildTypeSchema(t *resolver.TypeRef) *base.Schema {
	if t == nil {
		return newSchema()
	}

	switch t.Shape {
	case resolver.ShapeArray:
		schema := newSchema("array")
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{A: g.buildTypeProxy(t.Elem)}
		return schema

	case resolver.ShapeMap:
		schema := newSchema("object")
		schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{A: g.buildTypeProxy(t.Elem)}
		return schema

	case resolver.ShapeObject:
		schema := newSchema("object")
		setObjectFields(schema, t.Fields, g.generateFieldSchema)
		return schema

	case resolver.ShapeScalar:
		schema := newSchema(t.Type)
		schema.Format = t.Format
		return schema
	}

	// ShapeAny is the empty schema, which accepts any JSON value. ShapeRef is
	// handled by the callers, ShapeTypeParam belongs to a generic template that
	// never reaches components, and ShapeUnsupported is rejected before here.
	return newSchema()
}

// buildTypeProxy is buildTypeSchema for a nested position, where a reference
// has to become an actual $ref rather than a schema carrying keywords.
func (g *Generator) buildTypeProxy(t *resolver.TypeRef) *base.SchemaProxy {
	if t != nil && t.Shape == resolver.ShapeRef {
		return base.CreateSchemaProxyRef(refPath(t.Ref))
	}
	return base.CreateSchemaProxy(g.buildTypeSchema(t))
}

// refPath is the components pointer for a schema name.
func refPath(name string) string {
	return "#/components/schemas/" + name
}

// generateRefSchema builds the schema for a field whose type is a named @schema.
// A bare $ref is emitted when the field has no annotation keywords and is not
// nullable; otherwise sibling keywords (description, deprecated, ...) are attached.
func (g *Generator) generateRefSchema(refPath string, field *resolver.Field) *base.SchemaProxy {
	// A $ref cannot also be typed "null", so nullability becomes a union. oneOf
	// rather than siblings: siblings would intersect the two, not unite them.
	if field.Nullable {
		wrapper := newSchema()
		wrapper.OneOf = []*base.SchemaProxy{
			base.CreateSchemaProxyRef(refPath),
			base.CreateSchemaProxy(newSchema("null")),
		}
		g.addFieldConstraints(wrapper, field)
		return base.CreateSchemaProxy(wrapper)
	}

	// JSON Schema 2020-12 lets $ref carry sibling keywords directly, so there is
	// nothing to wrap. An empty siblings schema renders as a bare $ref on its own,
	// which is why this needs no special case for unannotated fields.
	// addFieldConstraints is the single source of truth for which keywords exist.
	siblings := newSchema()
	g.addFieldConstraints(siblings, field)
	return base.CreateSchemaProxyRefWithSchema(refPath, siblings)
}

// addFieldConstraints applies every annotation keyword a field carries.
//
// It is the single source of truth for which keywords exist: generateRefSchema
// builds an empty schema, runs it through here, and emits a bare $ref when
// nothing came out, so a keyword added here works on refs without further
// changes.
func (g *Generator) addFieldConstraints(schema *base.Schema, field *resolver.Field) {
	if field.Description != "" {
		schema.Description = field.Description
	}
	g.addValueConstraints(schema, field)
}

// addValueConstraints applies everything addFieldConstraints does except the
// description.
//
// Parameters need the split: an OpenAPI parameter carries its own description
// field, and repeating it inside the parameter's schema would say the same
// thing twice in the rendered document.
func (g *Generator) addValueConstraints(schema *base.Schema, field *resolver.Field) {
	if field.Format != "" {
		schema.Format = field.Format
	}
	if len(field.Enum) > 0 {
		// An array's enum constrains its items, not the array itself, so it
		// goes inside items and is typed by the element rather than by "array".
		if field.Type.IsArray() {
			if schema.Items != nil && schema.Items.A != nil {
				if itemSchema, _ := schema.Items.A.BuildSchema(); itemSchema != nil {
					itemSchema.Enum = convertEnumToYAMLNodes(field.Enum, field.Type.Elem.ScalarName())
				}
			}
		} else {
			schema.Enum = convertEnumToYAMLNodes(field.Enum, field.Type.ScalarName())
		}
	}
	if field.Example != "" {
		schema.Example = &yaml.Node{Kind: yaml.ScalarNode, Value: field.Example}
	}
	if field.Default != "" {
		schema.Default = &yaml.Node{Kind: yaml.ScalarNode, Value: field.Default}
	}
	if field.Pattern != "" {
		schema.Pattern = field.Pattern
	}
	if field.MinLength != nil {
		val := int64(*field.MinLength)
		schema.MinLength = &val
	}
	if field.MaxLength != nil {
		val := int64(*field.MaxLength)
		schema.MaxLength = &val
	}
	if field.MinItems != nil {
		val := int64(*field.MinItems)
		schema.MinItems = &val
	}
	if field.MaxItems != nil {
		val := int64(*field.MaxItems)
		schema.MaxItems = &val
	}
	if field.UniqueItems {
		schema.UniqueItems = &field.UniqueItems
	}
	if field.Minimum != nil {
		schema.Minimum = field.Minimum
	}
	if field.Maximum != nil {
		schema.Maximum = field.Maximum
	}
	if field.ExclusiveMinimum != nil {
		schema.ExclusiveMinimum = exclusiveBound(*field.ExclusiveMinimum)
	}
	if field.ExclusiveMaximum != nil {
		schema.ExclusiveMaximum = exclusiveBound(*field.ExclusiveMaximum)
	}
	if field.Nullable {
		markNullable(schema)
	}
	if field.Deprecated {
		schema.Deprecated = &field.Deprecated
	}
	if field.ReadOnly {
		schema.ReadOnly = &field.ReadOnly
	}
	if field.WriteOnly {
		schema.WriteOnly = &field.WriteOnly
	}
}

// convertEnumToYAMLNodes converts enum values to yaml.Node slice
func convertEnumToYAMLNodes(values []string, openAPIType string) []*yaml.Node {
	result := make([]*yaml.Node, len(values))
	if openAPIType == "integer" {
		for i, v := range values {
			result[i] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: v}
		}
	} else {
		for i, v := range values {
			result[i] = &yaml.Node{Kind: yaml.ScalarNode, Value: v}
		}
	}
	return result
}

// generateParameterFieldSchema generates a schema for a parameter field.
//
// Parameters are scalars or arrays of scalars — never objects, never refs — so
// this needs none of the shape handling generateFieldSchema does. The
// description is deliberately left off: it belongs on the parameter itself.
func (g *Generator) generateParameterFieldSchema(field *resolver.Field) *base.SchemaProxy {
	schema := g.buildTypeSchema(field.Type)
	g.addValueConstraints(schema, field)

	return base.CreateSchemaProxy(schema)
}

// generateSecuritySchemes generates security schemes
func (g *Generator) generateSecuritySchemes(schemes map[string]*resolver.SecurityScheme) *orderedmap.Map[string, *v3.SecurityScheme] {
	result := orderedmap.New[string, *v3.SecurityScheme]()

	for _, name := range slices.Sorted(maps.Keys(schemes)) {
		scheme := schemes[name]
		ss := &v3.SecurityScheme{
			Type:        scheme.Type,
			Description: scheme.Description,
		}
		if scheme.Scheme != "" {
			ss.Scheme = scheme.Scheme
		}
		if scheme.BearerFormat != "" {
			ss.BearerFormat = scheme.BearerFormat
		}
		if scheme.In != "" {
			ss.In = scheme.In
		}
		if scheme.ParameterName != "" {
			ss.Name = scheme.ParameterName
		}
		result.Set(name, ss)
	}

	return result
}

// generateSecurity generates global security requirements
func (g *Generator) generateSecurity(security [][]*resolver.SecurityRequirement) []*base.SecurityRequirement {
	result := make([]*base.SecurityRequirement, len(security))

	for i, reqs := range security {
		requirements := orderedmap.New[string, []string]()
		for _, r := range reqs {
			requirements.Set(r.SchemeName, r.Scopes)
		}
		result[i] = &base.SecurityRequirement{
			Requirements: requirements,
		}
	}

	return result
}

// generatePaths generates the paths section
func (g *Generator) generatePaths(endpoints []*resolver.Endpoint) *v3.Paths {
	paths := &v3.Paths{
		PathItems: orderedmap.New[string, *v3.PathItem](),
	}

	// Group endpoints by path
	pathMap := make(map[string]*v3.PathItem)

	for _, endpoint := range endpoints {
		if _, ok := pathMap[endpoint.Path]; !ok {
			pathMap[endpoint.Path] = &v3.PathItem{}
		}

		operation := g.generateOperation(endpoint)

		// Set operation on the appropriate method
		switch strings.ToLower(endpoint.Method) {
		case "get":
			pathMap[endpoint.Path].Get = operation
		case "post":
			pathMap[endpoint.Path].Post = operation
		case "put":
			pathMap[endpoint.Path].Put = operation
		case "delete":
			pathMap[endpoint.Path].Delete = operation
		case "patch":
			pathMap[endpoint.Path].Patch = operation
		case "head":
			pathMap[endpoint.Path].Head = operation
		case "options":
			pathMap[endpoint.Path].Options = operation
		case "trace":
			pathMap[endpoint.Path].Trace = operation
		}
	}

	// Add to paths in sorted order for deterministic output
	for _, path := range slices.Sorted(maps.Keys(pathMap)) {
		paths.PathItems.Set(path, pathMap[path])
	}

	return paths
}

// generateOperation generates an operation
func (g *Generator) generateOperation(endpoint *resolver.Endpoint) *v3.Operation {
	op := &v3.Operation{}

	if endpoint.Summary != "" {
		op.Summary = endpoint.Summary
	}

	if endpoint.Description != "" {
		op.Description = endpoint.Description
	}

	if endpoint.OperationID != "" {
		op.OperationId = endpoint.OperationID
	}

	if len(endpoint.Tags) > 0 {
		op.Tags = endpoint.Tags
	}

	// The resolver already merged and ordered these, so emission is a
	// straight walk.
	op.Parameters = g.generateParameters(endpoint.Parameters)

	if endpoint.Request != nil {
		op.RequestBody = g.generateRequestBody(endpoint.Request)
	}

	op.Responses = g.generateResponses(endpoint.Responses)

	// Add security
	if endpoint.Auth != "" {
		requirements := orderedmap.New[string, []string]()
		requirements.Set(endpoint.Auth, []string{})
		op.Security = []*base.SecurityRequirement{
			{Requirements: requirements},
		}
	}

	if endpoint.Deprecated {
		t := true
		op.Deprecated = &t
	}

	return op
}

// generateParameters renders the operation's parameters in the order the
// resolver put them in. Returns nil when there are none, so the key stays out
// of the document.
func (g *Generator) generateParameters(parameters []*resolver.Parameter) []*v3.Parameter {
	if len(parameters) == 0 {
		return nil
	}

	params := make([]*v3.Parameter, 0, len(parameters))
	for _, param := range parameters {
		field := param.Field

		p := &v3.Parameter{
			Name:        field.Name,
			In:          param.In,
			Description: field.Description,
			Required:    &field.Required,
			Schema:      g.generateParameterFieldSchema(field),
		}

		if field.Deprecated {
			p.Deprecated = true
		}

		params = append(params, p)
	}

	return params
}

// generateRequestBody generates a request body, from either @request form.
//
// Nil when there is no schema to carry, so the operation has no requestBody key
// at all. It used to return an empty &v3.RequestBody{}, which rendered as
// "requestBody: {}" -- a Request Body Object with no content, which the spec
// does not allow. The validator rejects the annotation that got here, so this is
// the belt to its braces rather than the only guard.
func (g *Generator) generateRequestBody(request *resolver.RequestBody) *v3.RequestBody {
	schema := g.bodySchema(request.Body, request.Inline)
	if schema == nil {
		return nil
	}

	return &v3.RequestBody{
		Content:  mediaContent(request.ContentType, schema),
		Required: &request.Required,
	}
}

// generateResponses renders the operation's responses in the order the resolver
// put them in.
func (g *Generator) generateResponses(responses []*resolver.Response) *v3.Responses {
	result := &v3.Responses{
		Codes: orderedmap.New[string, *v3.Response](),
	}

	for _, response := range responses {
		resp := &v3.Response{
			Description: response.Description,
			Headers:     generateResponseHeaders(response.Headers),
		}

		if schema := g.bodySchema(response.Body, response.Inline); schema != nil && response.ContentType != "" {
			resp.Content = mediaContent(response.ContentType, schema)
		}

		result.Codes.Set(response.StatusCode, resp)
	}

	return result
}

// bodySchema renders whichever of the two body forms is present, or nil when a
// message carries no body at all — a 204, or an @endpoint with no @request.
func (g *Generator) bodySchema(named *resolver.Body, inline *resolver.InlineBody) *base.SchemaProxy {
	switch {
	case named != nil && named.Schema != "":
		return g.generateBodySchema(named)
	case inline != nil && len(inline.Fields) > 0:
		return g.generateInlineBodySchema(inline)
	}
	return nil
}

// generateResponseHeaders renders a response's header parameters. Returns nil
// when there are none, so the caller can assign unconditionally and still leave
// the field absent from the output.
func generateResponseHeaders(params []*resolver.ParameterStruct) *orderedmap.Map[string, *v3.Header] {
	if len(params) == 0 {
		return nil
	}

	headers := orderedmap.New[string, *v3.Header]()
	for _, param := range params {
		for _, field := range param.Fields {
			schema := newSchema(field.Type.ScalarName())
			if field.Format != "" {
				schema.Format = field.Format
			}

			headers.Set(field.Name, &v3.Header{
				Schema:      base.CreateSchemaProxy(schema),
				Description: field.Description,
			})
		}
	}

	return headers
}

// mediaContent wraps a single schema as a one-entry content map, which is the
// only shape specgen emits — there is no multi-content-type support.
func mediaContent(contentType string, schema *base.SchemaProxy) *orderedmap.Map[string, *v3.MediaType] {
	content := orderedmap.New[string, *v3.MediaType]()
	content.Set(contentType, &v3.MediaType{Schema: schema})
	return content
}

// generateInlineBodySchema renders an in-function struct as a body, wrapped in
// its @bind envelope when it has one.
func (g *Generator) generateInlineBodySchema(inline *resolver.InlineBody) *base.SchemaProxy {
	if inline.Bind != nil {
		return g.generateInlineWrappedSchema(inline)
	}
	return g.generateInlineSchema(inline.Fields)
}

// generateBodySchema generates schema for a body
func (g *Generator) generateBodySchema(body *resolver.Body) *base.SchemaProxy {
	if body.Bind != nil {
		return g.generateWrappedSchema(body)
	}
	return g.buildTypeProxy(body.Type)
}

// generateWrappedSchema generates a schema where the body is wrapped in an envelope
func (g *Generator) generateWrappedSchema(body *resolver.Body) *base.SchemaProxy {
	bodySchema := func() *base.SchemaProxy {
		return g.buildTypeProxy(body.Type)
	}

	if body.Bind.WrapperSchema == nil {
		return bodySchema()
	}

	return g.wrapInEnvelope(body.Bind.WrapperSchema, body.Bind.Field, bodySchema)
}

// generateInlineWrappedSchema wraps inline struct fields in a wrapper schema
func (g *Generator) generateInlineWrappedSchema(inline *resolver.InlineBody) *base.SchemaProxy {
	bodySchema := func() *base.SchemaProxy {
		return g.generateInlineSchema(inline.Fields)
	}

	if inline.Bind == nil || inline.Bind.WrapperSchema == nil {
		return bodySchema()
	}

	return g.wrapInEnvelope(inline.Bind.WrapperSchema, inline.Bind.Field, bodySchema)
}

// wrapInEnvelope renders a @bind wrapper: the wrapper schema inlined, with the
// bound field replaced by the body. Every other field renders normally.
//
// The two callers differ only in what the body is — a named schema reference or
// an inline struct — which is why it arrives as a function.
func (g *Generator) wrapInEnvelope(wrapper *resolver.Schema, boundField string, body func() *base.SchemaProxy) *base.SchemaProxy {
	schema := newSchema("object")

	if wrapper.Description != "" {
		schema.Description = wrapper.Description
	}

	setObjectFields(schema, wrapper.Fields, func(field *resolver.Field) *base.SchemaProxy {
		if field.GoName == boundField {
			return body()
		}
		return g.generateFieldSchema(field)
	})

	return base.CreateSchemaProxy(schema)
}

// generateInlineSchema generates an object schema from inline struct fields
func (g *Generator) generateInlineSchema(fields []*resolver.Field) *base.SchemaProxy {
	schema := newSchema("object")
	setObjectFields(schema, fields, g.generateFieldSchema)
	return base.CreateSchemaProxy(schema)
}

// setObjectFields fills in an object schema's properties and required list.
// fieldSchema renders one field, which is what lets a @bind wrapper substitute
// the body for its bound field while every other field renders normally.
//
// Property order is the field order it is given, which is Go declaration order
// all the way back to the resolver. Required is emitted only when non-empty, so
// an all-optional object has no required key rather than an empty list.
func setObjectFields(schema *base.Schema, fields []*resolver.Field, fieldSchema func(*resolver.Field) *base.SchemaProxy) {
	if len(fields) == 0 {
		return
	}

	props := orderedmap.New[string, *base.SchemaProxy]()
	var required []string

	for _, field := range fields {
		props.Set(field.Name, fieldSchema(field))
		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema.Properties = props
	if len(required) > 0 {
		schema.Required = required
	}
}
