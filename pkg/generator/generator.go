package generator

import (
	"maps"
	"slices"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
)

// Generate builds the OpenAPI document from a resolved package. The IR is
// pre-ordered by the resolver (schemas sorted, parameters and responses in
// emission order), so assembly here is a straight traversal.
func (g *Generator) Generate(pkg *resolver.Package) *v3.Document {
	doc := &v3.Document{
		Version: g.openAPIVersion(),
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

	return doc
}

// generateInfo generates the info section.
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

// generateServers generates the servers section.
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

// generateTags generates the document-level tags.
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

// generateComponents generates the components section. The components object
// itself is always emitted, even when empty (legacy behavior).
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

// generateSchemas generates component schemas. The list arrives sorted by
// name; generic templates are skipped.
func (g *Generator) generateSchemas(schemas []*resolver.Schema) *orderedmap.Map[string, *base.SchemaProxy] {
	result := orderedmap.New[string, *base.SchemaProxy]()

	for _, schema := range schemas {
		if schema.IsGeneric {
			continue
		}
		result.Set(schema.Name, g.generateSchema(schema))
	}

	return result
}

// generateSchema generates a single named object schema.
func (g *Generator) generateSchema(schema *resolver.Schema) *base.SchemaProxy {
	s := g.schemaBuilder.NewSchema()
	g.schemaBuilder.SetType(s, "object")

	if schema.Description != "" {
		s.Description = schema.Description
	}

	if len(schema.Fields) > 0 {
		props := orderedmap.New[string, *base.SchemaProxy]()
		var required []string

		for _, field := range schema.Fields {
			props.Set(field.Name, g.fieldSchema(field))
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

// generateSecuritySchemes generates security schemes sorted by name.
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

// generateSecurity generates global security requirements.
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

// generatePaths groups endpoints by path and emits paths sorted. Duplicate
// method+path pairs are last-wins (legacy behavior; the validator gains a
// duplicate check in a later phase).
func (g *Generator) generatePaths(endpoints []*resolver.Endpoint) *v3.Paths {
	paths := &v3.Paths{
		PathItems: orderedmap.New[string, *v3.PathItem](),
	}

	pathMap := make(map[string]*v3.PathItem)

	for _, endpoint := range endpoints {
		item, ok := pathMap[endpoint.Path]
		if !ok {
			item = &v3.PathItem{}
			pathMap[endpoint.Path] = item
		}

		operation := g.generateOperation(endpoint)

		switch strings.ToLower(endpoint.Method) {
		case "get":
			item.Get = operation
		case "post":
			item.Post = operation
		case "put":
			item.Put = operation
		case "delete":
			item.Delete = operation
		case "patch":
			item.Patch = operation
		case "head":
			item.Head = operation
		case "options":
			item.Options = operation
		case "trace":
			item.Trace = operation
		}
	}

	for _, path := range slices.Sorted(maps.Keys(pathMap)) {
		paths.PathItems.Set(path, pathMap[path])
	}

	return paths
}

// generateOperation generates one operation.
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

	if len(endpoint.Parameters) > 0 {
		params := make([]*v3.Parameter, 0, len(endpoint.Parameters))
		for _, p := range endpoint.Parameters {
			params = append(params, g.generateParameter(p))
		}
		op.Parameters = params
	}

	if endpoint.Request != nil {
		op.RequestBody = g.generateRequestBody(endpoint.Request)
	}

	op.Responses = g.generateResponses(endpoint.Responses)

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

// generateParameter generates a single operation parameter.
func (g *Generator) generateParameter(p *resolver.Param) *v3.Parameter {
	field := p.Field
	required := field.Required

	param := &v3.Parameter{
		Name:        field.Name,
		In:          p.In,
		Description: field.Description,
		Required:    &required,
		Schema:      g.parameterFieldSchema(field),
	}

	if field.Constraints.Deprecated {
		param.Deprecated = true
	}

	return param
}

// generateRequestBody generates a request body. A request without content
// renders as an empty requestBody (legacy behavior for bodyless @request).
func (g *Generator) generateRequestBody(request *resolver.Request) *v3.RequestBody {
	if request.Content == nil {
		return &v3.RequestBody{}
	}

	content := orderedmap.New[string, *v3.MediaType]()
	content.Set(request.ContentType, &v3.MediaType{
		Schema: g.bodySchema(request.Content),
	})

	required := request.Required
	return &v3.RequestBody{
		Content:  content,
		Required: &required,
	}
}

// generateResponses generates the responses in their pre-merged order.
func (g *Generator) generateResponses(responses []*resolver.Response) *v3.Responses {
	result := &v3.Responses{
		Codes: orderedmap.New[string, *v3.Response](),
	}

	for _, response := range responses {
		resp := &v3.Response{
			Description: response.Description,
		}

		if len(response.Headers) > 0 {
			resp.Headers = g.responseHeaders(response.Headers)
		}

		if response.Content != nil {
			content := orderedmap.New[string, *v3.MediaType]()
			content.Set(response.ContentType, &v3.MediaType{
				Schema: g.bodySchema(response.Content),
			})
			resp.Content = content
		}

		result.Codes.Set(response.Status, resp)
	}

	return result
}

// responseHeaders generates response headers. Header schemas carry only the
// type and format; the description lives on the header itself.
func (g *Generator) responseHeaders(fields []*resolver.Field) *orderedmap.Map[string, *v3.Header] {
	headers := orderedmap.New[string, *v3.Header]()

	for _, field := range fields {
		headerSchema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(headerSchema, field.Type.OpenAPI)
		if field.Format != "" {
			headerSchema.Format = field.Format
		}

		headers.Set(field.Name, &v3.Header{
			Schema:      base.CreateSchemaProxy(headerSchema),
			Description: field.Description,
		})
	}

	return headers
}
