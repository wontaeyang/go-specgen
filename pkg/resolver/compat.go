package resolver

// ConvertLegacy bridges the legacy ResolvedPackage IR to the emission-ready
// Package IR during the rewrite transition. The Go-type string parsing here
// is lifted verbatim from the legacy generator (generateFieldSchemaWithRefs,
// generateSchemaRef) so the produced TypeInfo reproduces its emission
// decisions exactly. This file is deleted once the resolver builds the new
// IR natively.

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ConvertLegacy converts the legacy resolved package into the new IR.
func ConvertLegacy(pkg *ResolvedPackage) *Package {
	out := &Package{
		Name: pkg.PackageName,
		API:  pkg.API,
	}

	for _, name := range slices.Sorted(maps.Keys(pkg.Schemas)) {
		out.Schemas = append(out.Schemas, convertSchema(pkg.Schemas[name], pkg.Schemas))
	}

	for _, e := range pkg.Endpoints {
		out.Endpoints = append(out.Endpoints, convertEndpoint(e, pkg.Parameters, pkg.Schemas))
	}

	return out
}

func convertSchema(s *ResolvedSchema, schemas map[string]*ResolvedSchema) *Schema {
	out := &Schema{
		Name:        s.Name,
		Description: s.Description,
		Deprecated:  s.Deprecated,
		IsGeneric:   s.IsGeneric,
	}
	for _, f := range s.Fields {
		out.Fields = append(out.Fields, convertField(f, schemas))
	}
	return out
}

func convertField(f *ResolvedField, schemas map[string]*ResolvedSchema) *Field {
	out := &Field{
		Name:        f.Name,
		GoName:      f.GoName,
		Description: f.Description,
		Required:    f.Required,
		Nullable:    f.Nullable,
		Format:      f.Format,
		Type:        convertTypeInfo(f, schemas),
		Constraints: Constraints{
			Enum:             f.Enum,
			Default:          f.Default,
			Example:          f.Example,
			Pattern:          f.Pattern,
			MinLength:        f.MinLength,
			MaxLength:        f.MaxLength,
			MinItems:         f.MinItems,
			MaxItems:         f.MaxItems,
			UniqueItems:      f.UniqueItems,
			Minimum:          f.Minimum,
			Maximum:          f.Maximum,
			ExclusiveMinimum: f.ExclusiveMinimum,
			ExclusiveMaximum: f.ExclusiveMaximum,
			Deprecated:       f.Deprecated,
			ReadOnly:         f.ReadOnly,
			WriteOnly:        f.WriteOnly,
		},
	}
	for _, inner := range f.InlineFields {
		out.Inline = append(out.Inline, convertField(inner, schemas))
	}
	for _, inner := range f.ItemsInlineFields {
		out.ItemsInline = append(out.ItemsInline, convertField(inner, schemas))
	}
	for _, inner := range f.MapValueInlineFields {
		out.MapValueInline = append(out.MapValueInline, convertField(inner, schemas))
	}
	return out
}

// convertTypeInfo mirrors the branch order of the legacy
// generateFieldSchemaWithRefs: []-prefix, map[-prefix, whole-type schema ref,
// any, IsArray fallback, scalar.
func convertTypeInfo(f *ResolvedField, schemas map[string]*ResolvedSchema) TypeInfo {
	goType := f.GoType

	if strings.HasPrefix(goType, "[]") {
		elem := strings.TrimPrefix(goType, "[]")
		ti := TypeInfo{IsArray: true, Items: f.ItemsType}
		if isLegacySchemaRef(elem, schemas) {
			ti.ItemsRef = extractTypeName(elem)
		} else if isLegacyPrimitive(extractTypeName(elem)) {
			ti.Items = legacyPrimitiveType(extractTypeName(elem))
		}
		return ti
	}

	if strings.HasPrefix(goType, "map[") {
		if idx := strings.LastIndex(goType, "]"); idx > 0 && idx < len(goType)-1 {
			val := goType[idx+1:]
			ti := TypeInfo{IsMap: true, MapValue: "string"}
			if isLegacySchemaRef(val, schemas) {
				ti.MapValueRef = extractTypeName(val)
				ti.MapValue = ""
			} else if isLegacyPrimitive(extractTypeName(val)) {
				ti.MapValue = legacyPrimitiveType(extractTypeName(val))
			}
			return ti
		}
	}

	if isLegacySchemaRef(goType, schemas) {
		return TypeInfo{Ref: extractTypeName(goType)}
	}

	if f.IsAnyValue {
		return TypeInfo{IsAny: true}
	}

	if f.IsArray {
		return TypeInfo{IsArray: true, Items: f.ItemsType}
	}

	return TypeInfo{OpenAPI: f.OpenAPIType}
}

func convertEndpoint(e *ResolvedEndpoint, params map[string]*ResolvedParameter, schemas map[string]*ResolvedSchema) *Endpoint {
	out := &Endpoint{
		Method:      e.Method,
		Path:        e.Path,
		OperationID: e.OperationID,
		Summary:     e.Summary,
		Description: e.Description,
		Auth:        e.Auth,
		Tags:        e.Tags,
		Deprecated:  e.Deprecated,
	}

	// Parameters in legacy emission order: named groups by kind, then inline
	// fields by kind. The map lookup (and silent drop on miss) matches the
	// legacy generator.
	named := []struct {
		in   string
		refs []*ResolvedParameter
	}{
		{"path", e.PathParams},
		{"query", e.QueryParams},
		{"header", e.HeaderParams},
		{"cookie", e.CookieParams},
	}
	for _, group := range named {
		for _, ref := range group.refs {
			p, ok := params[ref.Name]
			if !ok {
				continue
			}
			for _, f := range p.Fields {
				out.Parameters = append(out.Parameters, &Param{In: group.in, Field: convertField(f, schemas)})
			}
		}
	}
	inline := []struct {
		in     string
		params *ResolvedInlineParams
	}{
		{"path", e.InlinePathParams},
		{"query", e.InlineQueryParams},
		{"header", e.InlineHeaderParams},
		{"cookie", e.InlineCookieParams},
	}
	for _, group := range inline {
		if group.params == nil {
			continue
		}
		for _, f := range group.params.Fields {
			out.Parameters = append(out.Parameters, &Param{In: group.in, Field: convertField(f, schemas)})
		}
	}

	// Request: named wins over inline, matching the legacy if/else-if.
	if e.Request != nil {
		out.Request = convertRequest(e.Request, schemas)
	} else if e.InlineRequest != nil {
		out.Request = convertInlineRequest(e.InlineRequest, schemas)
	}

	// Responses: sorted named statuses, then sorted inline statuses with
	// named-wins conflict resolution.
	for _, status := range slices.Sorted(maps.Keys(e.Responses)) {
		out.Responses = append(out.Responses, convertResponse(e.Responses[status], schemas))
	}
	for _, status := range slices.Sorted(maps.Keys(e.InlineResponses)) {
		if _, taken := e.Responses[status]; taken {
			continue
		}
		out.Responses = append(out.Responses, convertInlineResponse(status, e.InlineResponses[status], schemas))
	}

	return out
}

func convertRequest(r *ResolvedRequestBody, schemas map[string]*ResolvedSchema) *Request {
	out := &Request{
		ContentType: r.ContentType,
		Required:    r.Required,
	}
	if r.Body != nil {
		out.Content = &Content{
			Ref:  convertBodyRef(r.Body),
			Bind: convertBind(r.Body.Bind, schemas),
		}
	}
	return out
}

func convertInlineRequest(b *ResolvedInlineBody, schemas map[string]*ResolvedSchema) *Request {
	out := &Request{
		ContentType: defaultContentType(b.ContentType),
		Required:    true,
	}
	if len(b.Fields) > 0 {
		out.Content = &Content{
			Fields: convertFields(b.Fields, schemas),
			Bind:   convertBind(b.Bind, schemas),
		}
	}
	return out
}

func convertResponse(r *ResolvedResponse, schemas map[string]*ResolvedSchema) *Response {
	out := &Response{
		Status:      r.StatusCode,
		Description: r.Description,
		ContentType: r.ContentType,
		Headers:     convertHeaderFields(r.Headers, schemas),
	}
	// The legacy generator emits content only when a body, its schema, and a
	// content type are all present.
	if r.Body != nil && r.Body.Schema != "" && r.ContentType != "" {
		out.Content = &Content{
			Ref:  convertBodyRef(r.Body),
			Bind: convertBind(r.Body.Bind, schemas),
		}
	}
	return out
}

func convertInlineResponse(status string, b *ResolvedInlineBody, schemas map[string]*ResolvedSchema) *Response {
	description := b.Description
	if description == "" {
		description = fmt.Sprintf("Response for status %s", status)
	}
	out := &Response{
		Status:      status,
		Description: description,
		ContentType: defaultContentType(b.ContentType),
		Headers:     convertHeaderFields(b.Headers, schemas),
	}
	if len(b.Fields) > 0 {
		out.Content = &Content{
			Fields: convertFields(b.Fields, schemas),
			Bind:   convertBind(b.Bind, schemas),
		}
	}
	return out
}

// convertHeaderFields flattens response header parameter groups into fields,
// preserving group-then-field order.
func convertHeaderFields(headers []*ResolvedParameter, schemas map[string]*ResolvedSchema) []*Field {
	var out []*Field
	for _, h := range headers {
		for _, f := range h.Fields {
			out = append(out, convertField(f, schemas))
		}
	}
	return out
}

func convertFields(fields []*ResolvedField, schemas map[string]*ResolvedSchema) []*Field {
	out := make([]*Field, 0, len(fields))
	for _, f := range fields {
		out = append(out, convertField(f, schemas))
	}
	return out
}

// convertBodyRef mirrors the legacy generateSchemaRef input: the element type
// decides between a primitive schema and a components ref.
func convertBodyRef(b *ResolvedBody) *TypeRef {
	ref := &TypeRef{IsArray: b.IsArray, IsMap: b.IsMap}
	if isLegacyPrimitive(b.ElementType) {
		ref.Primitive = legacyPrimitiveType(b.ElementType)
	} else {
		ref.Schema = b.ElementType
	}
	return ref
}

func convertBind(b *ResolvedBindTarget, schemas map[string]*ResolvedSchema) *BindTarget {
	if b == nil {
		return nil
	}
	out := &BindTarget{Field: b.Field}
	if b.WrapperSchema != nil {
		out.Wrapper = convertSchema(b.WrapperSchema, schemas)
	}
	return out
}

func defaultContentType(ct string) string {
	if ct == "" {
		return "application/json"
	}
	return ct
}

// extractTypeName strips a package qualifier: "time.Time" -> "Time".
func extractTypeName(goType string) string {
	if idx := strings.LastIndex(goType, "."); idx >= 0 {
		return goType[idx+1:]
	}
	return goType
}

func isLegacySchemaRef(goType string, schemas map[string]*ResolvedSchema) bool {
	if schema, ok := schemas[extractTypeName(goType)]; ok {
		return !schema.IsGeneric
	}
	return false
}

func isLegacyPrimitive(typeName string) bool {
	switch typeName {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "bool", "byte":
		return true
	}
	return false
}

func legacyPrimitiveType(typeName string) string {
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
