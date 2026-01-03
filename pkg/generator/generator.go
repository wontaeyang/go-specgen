package generator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
	"go.yaml.in/yaml/v4"
)

// Generator generates OpenAPI specifications using libopenapi's v3high models
type Generator struct {
	version       string // "3.0", "3.1", "3.2"
	schemaBuilder *SchemaBuilder
}

// OutputFormat represents the output format
type OutputFormat string

const (
	FormatJSON OutputFormat = "json"
	FormatYAML OutputFormat = "yaml"
)

// NewGenerator creates a new generator
func NewGenerator(version string) *Generator {
	return &Generator{
		version:       version,
		schemaBuilder: NewSchemaBuilder(version),
	}
}

// Generate generates an OpenAPI spec from a resolved package
func (g *Generator) Generate(pkg *resolver.ResolvedPackage) (*v3.Document, error) {
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

	doc.Paths = g.generatePaths(pkg.Endpoints, pkg.Parameters, pkg.Schemas)
	doc.Components = g.generateComponents(pkg)

	if len(pkg.API.Security) > 0 {
		doc.Security = g.generateSecurity(pkg.API.Security)
	}

	return doc, nil
}

// getOpenAPIVersion returns the OpenAPI version string
func (g *Generator) getOpenAPIVersion() string {
	switch g.version {
	case "3.0":
		return "3.0.3"
	case "3.1":
		return "3.1.0"
	case "3.2":
		return "3.2.0"
	default:
		return "3.0.3"
	}
}

// Render renders the spec to the specified format
func (g *Generator) Render(doc *v3.Document, format OutputFormat) ([]byte, error) {
	switch format {
	case FormatJSON:
		return doc.RenderJSON("  ")
	case FormatYAML:
		return doc.Render()
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// generateInfo generates the info section
func (g *Generator) generateInfo(api *resolver.ResolvedAPI) *base.Info {
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
func (g *Generator) generateComponents(pkg *resolver.ResolvedPackage) *v3.Components {
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
func (g *Generator) generateSchemas(schemas map[string]*resolver.ResolvedSchema) *orderedmap.Map[string, *base.SchemaProxy] {
	result := orderedmap.New[string, *base.SchemaProxy]()

	for name, schema := range schemas {
		// Skip generic schemas - they are templates, not concrete types
		if schema.IsGeneric {
			continue
		}

		result.Set(name, g.generateSchema(schema, schemas))
	}

	return result
}

// generateSchema generates a single schema
func (g *Generator) generateSchema(schema *resolver.ResolvedSchema, allSchemas map[string]*resolver.ResolvedSchema) *base.SchemaProxy {
	s := g.schemaBuilder.NewSchema()
	g.schemaBuilder.SetType(s, "object")

	if schema.Description != "" {
		s.Description = schema.Description
	}

	if len(schema.Fields) > 0 {
		props := orderedmap.New[string, *base.SchemaProxy]()
		var required []string

		for _, field := range schema.Fields {
			props.Set(field.Name, g.generateFieldSchemaWithRefs(field, allSchemas))
			if field.Required {
				required = append(required, field.Name)
			}
		}

		s.Properties = props
		if len(required) > 0 {
			s.Required = required
		}
	}

	if schema.Deprecated {
		t := true
		s.Deprecated = &t
	}

	return base.CreateSchemaProxy(s)
}

// generateFieldSchemaWithRefs generates a schema for a field, using $ref for schema types
func (g *Generator) generateFieldSchemaWithRefs(field *resolver.ResolvedField, schemas map[string]*resolver.ResolvedSchema) *base.SchemaProxy {
	// Handle anonymous structs - inline their fields
	if len(field.InlineFields) > 0 {
		return g.buildInlineObjectSchema(field.InlineFields, schemas)
	}

	// Handle arrays of anonymous structs
	if len(field.ItemsInlineFields) > 0 {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "array")
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: g.buildInlineObjectSchema(field.ItemsInlineFields, schemas),
		}
		g.addFieldConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	// Handle maps of anonymous structs
	if len(field.MapValueInlineFields) > 0 {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "object")
		schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: g.buildInlineObjectSchema(field.MapValueInlineFields, schemas),
		}
		g.addFieldConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	goType := field.GoType

	// Handle arrays: []User or []string
	if strings.HasPrefix(goType, "[]") {
		elemType := strings.TrimPrefix(goType, "[]")
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "array")

		if isSchemaReference(elemType, schemas) {
			schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
				A: base.CreateSchemaProxyRef(fmt.Sprintf("#/components/schemas/%s", extractTypeName(elemType))),
			}
		} else if isPrimitive(extractTypeName(elemType)) {
			itemSchema := g.schemaBuilder.NewSchema()
			g.schemaBuilder.SetType(itemSchema, goTypeToPrimitive(extractTypeName(elemType)))
			schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
				A: base.CreateSchemaProxy(itemSchema),
			}
		} else {
			itemSchema := g.schemaBuilder.NewSchema()
			g.schemaBuilder.SetType(itemSchema, field.ItemsType)
			schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
				A: base.CreateSchemaProxy(itemSchema),
			}
		}
		g.addFieldConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	// Handle maps: map[string]User
	if strings.HasPrefix(goType, "map[") {
		if idx := strings.LastIndex(goType, "]"); idx > 0 && idx < len(goType)-1 {
			valueType := goType[idx+1:]
			schema := g.schemaBuilder.NewSchema()
			g.schemaBuilder.SetType(schema, "object")

			if isSchemaReference(valueType, schemas) {
				schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{
					A: base.CreateSchemaProxyRef(fmt.Sprintf("#/components/schemas/%s", extractTypeName(valueType))),
				}
			} else if isPrimitive(extractTypeName(valueType)) {
				propSchema := g.schemaBuilder.NewSchema()
				g.schemaBuilder.SetType(propSchema, goTypeToPrimitive(extractTypeName(valueType)))
				schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{
					A: base.CreateSchemaProxy(propSchema),
				}
			} else {
				propSchema := g.schemaBuilder.NewSchema()
				g.schemaBuilder.SetType(propSchema, "string")
				schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{
					A: base.CreateSchemaProxy(propSchema),
				}
			}
			g.addFieldConstraints(schema, field)
			return base.CreateSchemaProxy(schema)
		}
	}

	// Handle schema references: User (named type that is a schema)
	if isSchemaReference(goType, schemas) {
		return base.CreateSchemaProxyRef(fmt.Sprintf("#/components/schemas/%s", extractTypeName(goType)))
	}

	// Handle any value (empty schema)
	if field.IsAnyValue {
		schema := g.schemaBuilder.NewSchema()
		g.addFieldConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	// Handle primitives and other types
	schema := g.schemaBuilder.NewSchema()
	if field.IsArray {
		g.schemaBuilder.SetType(schema, "array")
		itemSchema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(itemSchema, field.ItemsType)
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: base.CreateSchemaProxy(itemSchema),
		}
	} else {
		g.schemaBuilder.SetType(schema, field.OpenAPIType)
	}

	g.addFieldConstraints(schema, field)
	return base.CreateSchemaProxy(schema)
}

// buildInlineObjectSchema builds an object schema from inline fields
func (g *Generator) buildInlineObjectSchema(fields []*resolver.ResolvedField, schemas map[string]*resolver.ResolvedSchema) *base.SchemaProxy {
	schema := g.schemaBuilder.NewSchema()
	g.schemaBuilder.SetType(schema, "object")

	props := orderedmap.New[string, *base.SchemaProxy]()
	var required []string

	for _, field := range fields {
		props.Set(field.Name, g.generateFieldSchemaWithRefs(field, schemas))
		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema.Properties = props
	if len(required) > 0 {
		schema.Required = required
	}

	return base.CreateSchemaProxy(schema)
}

// addFieldConstraints adds common field constraints to a schema
func (g *Generator) addFieldConstraints(schema *base.Schema, field *resolver.ResolvedField) {
	if field.Description != "" {
		schema.Description = field.Description
	}
	if field.Format != "" {
		schema.Format = field.Format
	}
	if len(field.Enum) > 0 {
		enumValues := convertEnumToYAMLNodes(field.Enum, field.OpenAPIType)
		if field.IsArray {
			// For arrays, enum goes inside items
			if schema.Items != nil && schema.Items.A != nil {
				itemSchema, _ := schema.Items.A.BuildSchema()
				if itemSchema != nil {
					itemSchema.Enum = enumValues
				}
			}
		} else {
			schema.Enum = enumValues
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
	if field.Nullable {
		g.schemaBuilder.SetNullable(schema, true)
	}
	if field.Deprecated {
		schema.Deprecated = &field.Deprecated
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

// generateFieldSchema generates a schema for a field (without schema reference awareness)
// Used for parameters and contexts where we don't have schema map
func (g *Generator) generateFieldSchema(field *resolver.ResolvedField) *base.SchemaProxy {
	// Handle any value (empty schema)
	if field.IsAnyValue {
		schema := g.schemaBuilder.NewSchema()
		if field.Description != "" {
			schema.Description = field.Description
		}
		return base.CreateSchemaProxy(schema)
	}

	// Handle anonymous structs - inline their fields
	if len(field.InlineFields) > 0 {
		return g.buildInlineObjectSchemaSimple(field.InlineFields)
	}

	// Handle arrays of anonymous structs
	if len(field.ItemsInlineFields) > 0 {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "array")
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: g.buildInlineObjectSchemaSimple(field.ItemsInlineFields),
		}
		g.addFieldConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	// Handle maps of anonymous structs
	if len(field.MapValueInlineFields) > 0 {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "object")
		schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: g.buildInlineObjectSchemaSimple(field.MapValueInlineFields),
		}
		g.addFieldConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	// Handle arrays
	schema := g.schemaBuilder.NewSchema()
	if field.IsArray {
		g.schemaBuilder.SetType(schema, "array")
		itemSchema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(itemSchema, field.ItemsType)
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: base.CreateSchemaProxy(itemSchema),
		}
	} else {
		g.schemaBuilder.SetType(schema, field.OpenAPIType)
	}

	g.addFieldConstraints(schema, field)
	return base.CreateSchemaProxy(schema)
}

// buildInlineObjectSchemaSimple builds an object schema from inline fields without schema refs
func (g *Generator) buildInlineObjectSchemaSimple(fields []*resolver.ResolvedField) *base.SchemaProxy {
	schema := g.schemaBuilder.NewSchema()
	g.schemaBuilder.SetType(schema, "object")

	props := orderedmap.New[string, *base.SchemaProxy]()
	var required []string

	for _, field := range fields {
		props.Set(field.Name, g.generateFieldSchema(field))
		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema.Properties = props
	if len(required) > 0 {
		schema.Required = required
	}

	return base.CreateSchemaProxy(schema)
}

// generateParameterFieldSchema generates a schema for a parameter field (excludes description)
func (g *Generator) generateParameterFieldSchema(field *resolver.ResolvedField) *base.SchemaProxy {
	schema := g.schemaBuilder.NewSchema()

	// Handle arrays
	if field.IsArray {
		g.schemaBuilder.SetType(schema, "array")
		itemSchema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(itemSchema, field.ItemsType)
		// Add enum to items if present
		if len(field.Enum) > 0 {
			itemSchema.Enum = convertEnumToYAMLNodes(field.Enum, field.ItemsType)
		}
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: base.CreateSchemaProxy(itemSchema),
		}
	} else {
		g.schemaBuilder.SetType(schema, field.OpenAPIType)
		// Add enum directly
		if len(field.Enum) > 0 {
			schema.Enum = convertEnumToYAMLNodes(field.Enum, field.OpenAPIType)
		}
	}

	if field.Format != "" {
		schema.Format = field.Format
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
	if field.Nullable {
		g.schemaBuilder.SetNullable(schema, true)
	}
	if field.Deprecated {
		schema.Deprecated = &field.Deprecated
	}

	return base.CreateSchemaProxy(schema)
}

// generateSecuritySchemes generates security schemes
func (g *Generator) generateSecuritySchemes(schemes map[string]*resolver.SecurityScheme) *orderedmap.Map[string, *v3.SecurityScheme] {
	result := orderedmap.New[string, *v3.SecurityScheme]()

	for name, scheme := range schemes {
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
func (g *Generator) generatePaths(endpoints []*resolver.ResolvedEndpoint, parameters map[string]*resolver.ResolvedParameter, schemas map[string]*resolver.ResolvedSchema) *v3.Paths {
	paths := &v3.Paths{
		PathItems: orderedmap.New[string, *v3.PathItem](),
	}

	// Group endpoints by path
	pathMap := make(map[string]*v3.PathItem)

	for _, endpoint := range endpoints {
		if _, ok := pathMap[endpoint.Path]; !ok {
			pathMap[endpoint.Path] = &v3.PathItem{}
		}

		operation := g.generateOperation(endpoint, parameters, schemas)

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

	// Add to paths
	for path, pathItem := range pathMap {
		paths.PathItems.Set(path, pathItem)
	}

	return paths
}

// generateOperation generates an operation
func (g *Generator) generateOperation(endpoint *resolver.ResolvedEndpoint, parameterMap map[string]*resolver.ResolvedParameter, schemas map[string]*resolver.ResolvedSchema) *v3.Operation {
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

	// Collect parameters
	var params []*v3.Parameter

	// Path parameters
	for _, paramRef := range endpoint.PathParams {
		if p, ok := parameterMap[paramRef.Name]; ok {
			params = append(params, g.generateParameters(p, "path")...)
		}
	}

	// Query parameters
	for _, paramRef := range endpoint.QueryParams {
		if p, ok := parameterMap[paramRef.Name]; ok {
			params = append(params, g.generateParameters(p, "query")...)
		}
	}

	// Header parameters
	for _, paramRef := range endpoint.HeaderParams {
		if p, ok := parameterMap[paramRef.Name]; ok {
			params = append(params, g.generateParameters(p, "header")...)
		}
	}

	// Cookie parameters
	for _, paramRef := range endpoint.CookieParams {
		if p, ok := parameterMap[paramRef.Name]; ok {
			params = append(params, g.generateParameters(p, "cookie")...)
		}
	}

	// Inline path parameters
	if endpoint.InlinePathParams != nil {
		params = append(params, g.generateInlineParameters(endpoint.InlinePathParams.Fields, "path")...)
	}

	// Inline query parameters
	if endpoint.InlineQueryParams != nil {
		params = append(params, g.generateInlineParameters(endpoint.InlineQueryParams.Fields, "query")...)
	}

	// Inline header parameters
	if endpoint.InlineHeaderParams != nil {
		params = append(params, g.generateInlineParameters(endpoint.InlineHeaderParams.Fields, "header")...)
	}

	// Inline cookie parameters
	if endpoint.InlineCookieParams != nil {
		params = append(params, g.generateInlineParameters(endpoint.InlineCookieParams.Fields, "cookie")...)
	}

	if len(params) > 0 {
		op.Parameters = params
	}

	// Add request body
	if endpoint.Request != nil {
		op.RequestBody = g.generateRequestBody(endpoint.Request, schemas)
	} else if endpoint.InlineRequest != nil {
		op.RequestBody = g.generateInlineRequestBody(endpoint.InlineRequest, schemas)
	}

	// Add responses
	op.Responses = g.generateResponsesWithInline(endpoint.Responses, endpoint.InlineResponses, schemas)

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

// generateParameters generates parameters from a parameter struct
func (g *Generator) generateParameters(param *resolver.ResolvedParameter, in string) []*v3.Parameter {
	params := make([]*v3.Parameter, 0, len(param.Fields))

	for _, field := range param.Fields {
		p := &v3.Parameter{
			Name:        field.Name,
			In:          in,
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

// generateInlineParameters generates parameters from inline struct fields
func (g *Generator) generateInlineParameters(fields []*resolver.ResolvedField, in string) []*v3.Parameter {
	params := make([]*v3.Parameter, 0, len(fields))

	for _, field := range fields {
		p := &v3.Parameter{
			Name:        field.Name,
			In:          in,
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

// generateRequestBody generates a request body
func (g *Generator) generateRequestBody(request *resolver.ResolvedRequestBody, schemas map[string]*resolver.ResolvedSchema) *v3.RequestBody {
	if request.Body == nil {
		return &v3.RequestBody{}
	}

	content := orderedmap.New[string, *v3.MediaType]()
	content.Set(request.ContentType, &v3.MediaType{
		Schema: g.generateBodySchema(request.Body, schemas),
	})

	return &v3.RequestBody{
		Content:  content,
		Required: &request.Required,
	}
}

// generateInlineRequestBody generates a request body from an inline struct
func (g *Generator) generateInlineRequestBody(inline *resolver.ResolvedInlineBody, schemas map[string]*resolver.ResolvedSchema) *v3.RequestBody {
	if inline == nil || len(inline.Fields) == 0 {
		return &v3.RequestBody{}
	}

	content := orderedmap.New[string, *v3.MediaType]()

	var schemaProxy *base.SchemaProxy
	if inline.Bind != nil {
		schemaProxy = g.generateInlineWrappedSchema(inline, schemas)
	} else {
		schemaProxy = g.generateInlineSchema(inline.Fields)
	}

	contentType := inline.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	content.Set(contentType, &v3.MediaType{Schema: schemaProxy})

	required := true
	return &v3.RequestBody{
		Content:  content,
		Required: &required,
	}
}

// generateResponsesWithInline generates responses merging explicit and inline definitions
func (g *Generator) generateResponsesWithInline(responses map[string]*resolver.ResolvedResponse, inlineResponses map[string]*resolver.ResolvedInlineBody, schemas map[string]*resolver.ResolvedSchema) *v3.Responses {
	result := &v3.Responses{
		Codes: orderedmap.New[string, *v3.Response](),
	}

	// Add explicit responses
	for statusCode, response := range responses {
		resp := &v3.Response{
			Description: response.Description,
		}

		// Add response headers if present
		if len(response.Headers) > 0 {
			headers := orderedmap.New[string, *v3.Header]()
			for _, headerParam := range response.Headers {
				for _, field := range headerParam.Fields {
					headerSchema := g.schemaBuilder.NewSchema()
					g.schemaBuilder.SetType(headerSchema, field.OpenAPIType)
					if field.Format != "" {
						headerSchema.Format = field.Format
					}

					header := &v3.Header{
						Schema:      base.CreateSchemaProxy(headerSchema),
						Description: field.Description,
					}
					headers.Set(field.Name, header)
				}
			}
			resp.Headers = headers
		}

		if response.Body != nil && response.Body.Schema != "" && response.ContentType != "" {
			content := orderedmap.New[string, *v3.MediaType]()
			content.Set(response.ContentType, &v3.MediaType{
				Schema: g.generateBodySchema(response.Body, schemas),
			})
			resp.Content = content
		}

		result.Codes.Set(statusCode, resp)
	}

	// Add inline responses (don't override explicit ones)
	for statusCode, inline := range inlineResponses {
		if result.Codes.GetOrZero(statusCode) != nil {
			continue // Skip if explicit response already exists
		}

		description := inline.Description
		if description == "" {
			description = fmt.Sprintf("Response for status %s", statusCode)
		}

		resp := &v3.Response{
			Description: description,
		}

		// Add inline response headers if present
		if len(inline.Headers) > 0 {
			headers := orderedmap.New[string, *v3.Header]()
			for _, headerParam := range inline.Headers {
				for _, field := range headerParam.Fields {
					headerSchema := g.schemaBuilder.NewSchema()
					g.schemaBuilder.SetType(headerSchema, field.OpenAPIType)
					if field.Format != "" {
						headerSchema.Format = field.Format
					}

					header := &v3.Header{
						Schema:      base.CreateSchemaProxy(headerSchema),
						Description: field.Description,
					}
					headers.Set(field.Name, header)
				}
			}
			resp.Headers = headers
		}

		if len(inline.Fields) > 0 {
			content := orderedmap.New[string, *v3.MediaType]()

			var schemaProxy *base.SchemaProxy
			if inline.Bind != nil {
				schemaProxy = g.generateInlineWrappedSchema(inline, schemas)
			} else {
				schemaProxy = g.generateInlineSchema(inline.Fields)
			}

			contentType := inline.ContentType
			if contentType == "" {
				contentType = "application/json"
			}
			content.Set(contentType, &v3.MediaType{Schema: schemaProxy})
			resp.Content = content
		}

		result.Codes.Set(statusCode, resp)
	}

	return result
}

// generateBodySchema generates schema for a body
func (g *Generator) generateBodySchema(body *resolver.ResolvedBody, schemas map[string]*resolver.ResolvedSchema) *base.SchemaProxy {
	if body.Bind != nil {
		return g.generateWrappedSchema(body, schemas)
	}
	return g.generateSchemaRef(body.Schema, body.IsArray, body.IsMap, body.ElementType)
}

// generateWrappedSchema generates a schema where the body is wrapped in an envelope
func (g *Generator) generateWrappedSchema(body *resolver.ResolvedBody, schemas map[string]*resolver.ResolvedSchema) *base.SchemaProxy {
	wrapperSchema := body.Bind.WrapperSchema
	if wrapperSchema == nil {
		return g.generateSchemaRef(body.Schema, body.IsArray, body.IsMap, body.ElementType)
	}

	schema := g.schemaBuilder.NewSchema()
	g.schemaBuilder.SetType(schema, "object")

	if wrapperSchema.Description != "" {
		schema.Description = wrapperSchema.Description
	}

	props := orderedmap.New[string, *base.SchemaProxy]()
	var required []string

	for _, field := range wrapperSchema.Fields {
		if field.GoName == body.Bind.Field {
			props.Set(field.Name, g.generateSchemaRef(body.Schema, body.IsArray, body.IsMap, body.ElementType))
		} else {
			props.Set(field.Name, g.generateFieldSchema(field))
		}

		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema.Properties = props
	if len(required) > 0 {
		schema.Required = required
	}

	return base.CreateSchemaProxy(schema)
}

// generateInlineWrappedSchema wraps inline struct fields in a wrapper schema
func (g *Generator) generateInlineWrappedSchema(inline *resolver.ResolvedInlineBody, schemas map[string]*resolver.ResolvedSchema) *base.SchemaProxy {
	if inline.Bind == nil || inline.Bind.WrapperSchema == nil {
		return g.generateInlineSchema(inline.Fields)
	}

	wrapperSchema := inline.Bind.WrapperSchema

	schema := g.schemaBuilder.NewSchema()
	g.schemaBuilder.SetType(schema, "object")

	if wrapperSchema.Description != "" {
		schema.Description = wrapperSchema.Description
	}

	props := orderedmap.New[string, *base.SchemaProxy]()
	var required []string

	for _, field := range wrapperSchema.Fields {
		if field.GoName == inline.Bind.Field {
			props.Set(field.Name, g.generateInlineSchema(inline.Fields))
		} else {
			props.Set(field.Name, g.generateFieldSchema(field))
		}

		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema.Properties = props
	if len(required) > 0 {
		schema.Required = required
	}

	return base.CreateSchemaProxy(schema)
}

// generateInlineSchema generates an object schema from inline struct fields
func (g *Generator) generateInlineSchema(fields []*resolver.ResolvedField) *base.SchemaProxy {
	schema := g.schemaBuilder.NewSchema()
	g.schemaBuilder.SetType(schema, "object")

	if len(fields) == 0 {
		return base.CreateSchemaProxy(schema)
	}

	props := orderedmap.New[string, *base.SchemaProxy]()
	var required []string

	for _, field := range fields {
		props.Set(field.Name, g.generateFieldSchema(field))
		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema.Properties = props
	if len(required) > 0 {
		schema.Required = required
	}

	return base.CreateSchemaProxy(schema)
}

// generateSchemaRef generates a schema reference (handles arrays, maps, and simple refs)
func (g *Generator) generateSchemaRef(schemaName string, isArray bool, isMap bool, elementType string) *base.SchemaProxy {
	if isArray {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "array")

		var itemsProxy *base.SchemaProxy
		if isPrimitive(elementType) {
			itemSchema := g.schemaBuilder.NewSchema()
			g.schemaBuilder.SetType(itemSchema, goTypeToPrimitive(elementType))
			itemsProxy = base.CreateSchemaProxy(itemSchema)
		} else {
			itemsProxy = base.CreateSchemaProxyRef(fmt.Sprintf("#/components/schemas/%s", elementType))
		}
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{A: itemsProxy}
		return base.CreateSchemaProxy(schema)
	}

	if isMap {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "object")

		var propsProxy *base.SchemaProxy
		if isPrimitive(elementType) {
			propSchema := g.schemaBuilder.NewSchema()
			g.schemaBuilder.SetType(propSchema, goTypeToPrimitive(elementType))
			propsProxy = base.CreateSchemaProxy(propSchema)
		} else {
			propsProxy = base.CreateSchemaProxyRef(fmt.Sprintf("#/components/schemas/%s", elementType))
		}
		schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{A: propsProxy}
		return base.CreateSchemaProxy(schema)
	}

	// Simple type or schema reference
	if isPrimitive(elementType) {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, goTypeToPrimitive(elementType))
		return base.CreateSchemaProxy(schema)
	}

	return base.CreateSchemaProxyRef(fmt.Sprintf("#/components/schemas/%s", elementType))
}

// extractTypeName extracts the simple type name from a Go type string
func extractTypeName(goType string) string {
	if idx := strings.LastIndex(goType, "."); idx >= 0 {
		return goType[idx+1:]
	}
	return goType
}

// isSchemaReference checks if a Go type corresponds to a known schema
func isSchemaReference(goType string, schemas map[string]*resolver.ResolvedSchema) bool {
	typeName := extractTypeName(goType)
	if schema, ok := schemas[typeName]; ok {
		return !schema.IsGeneric
	}
	return false
}

// isPrimitive checks if a type is a Go primitive
func isPrimitive(typeName string) bool {
	primitives := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "byte": true,
	}
	return primitives[typeName]
}

// goTypeToPrimitive converts Go type to OpenAPI primitive type
func goTypeToPrimitive(typeName string) string {
	switch typeName {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}

// convertEnumValues converts string enum values to the appropriate type based on OpenAPI type
func convertEnumValues(values []string, openAPIType string) []any {
	result := make([]any, len(values))
	if openAPIType == "integer" {
		for i, v := range values {
			if num, err := strconv.ParseInt(v, 10, 64); err == nil {
				result[i] = num
			} else {
				result[i] = v
			}
		}
		return result
	}
	for i, v := range values {
		result[i] = v
	}
	return result
}
