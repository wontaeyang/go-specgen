package resolver

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/parser"
	"golang.org/x/tools/go/packages"
)

// specialTypeMapping defines how a special Go type maps to OpenAPI
type specialTypeMapping struct {
	openAPIType string
	format      string
}

// specialTypes maps package path + type name to OpenAPI type info
// These are Go standard library struct types that should be treated as primitives
var specialTypes = map[string]map[string]*specialTypeMapping{
	"time": {
		"Time": {openAPIType: "string", format: "date-time"},
	},
	"net/url": {
		"URL": {openAPIType: "string", format: "uri"},
	},
	"net/netip": {
		"Addr":     {openAPIType: "string", format: ""},
		"AddrPort": {openAPIType: "string", format: ""},
		"Prefix":   {openAPIType: "string", format: ""},
	},
	"math/big": {
		"Int":   {openAPIType: "string", format: ""},
		"Float": {openAPIType: "string", format: ""},
		"Rat":   {openAPIType: "string", format: ""},
	},
	"regexp": {
		"Regexp": {openAPIType: "string", format: ""},
	},
}

// resolveSpecialType checks if a type is a special standard library type
// and returns its OpenAPI mapping, or nil if not special
func resolveSpecialType(pkgPath, typeName string) *specialTypeMapping {
	if pkgTypes, ok := specialTypes[pkgPath]; ok {
		if mapping, ok := pkgTypes[typeName]; ok {
			return mapping
		}
	}
	return nil
}

// isSpecialType checks if a package path and type name represent a special type
func isSpecialType(pkgPath, typeName string) bool {
	return resolveSpecialType(pkgPath, typeName) != nil
}

// Resolver resolves Go types to OpenAPI types
type Resolver struct {
	parsed *parser.Package
	pkg    *packages.Package

	// schemaNames is the set of @schema type names. A named struct in that set
	// resolves to a $ref; one outside it has no representation. Held here
	// rather than threaded through every resolve function, since it is fixed
	// for the life of the resolver.
	schemaNames map[string]bool
}

// NewResolver creates a resolver for an already-parsed package.
//
// The Go package comes from the parser, which loaded it to read comments in the
// first place. Loading it again here would produce a second set of *types.Type
// values that compare unequal to the parser's, and would double the cost of the
// most expensive step in the pipeline.
func NewResolver(parsed *parser.Package) *Resolver {
	schemaNames := make(map[string]bool, len(parsed.Schemas))
	for name := range parsed.Schemas {
		schemaNames[name] = true
	}

	return &Resolver{
		parsed:      parsed,
		pkg:         parsed.Pkg,
		schemaNames: schemaNames,
	}
}

// Resolve resolves all types in the parsed package
func (r *Resolver) Resolve() (*Package, error) {
	parsed := r.parsed

	resolved := &Package{
		PackageName: parsed.PackageName,
		Schemas:     make(map[string]*Schema),
		Parameters:  make(map[string]*ParameterStruct),
		Endpoints:   make([]*Endpoint, 0),
	}

	// Resolve API info (no type resolution needed, just copy)
	if parsed.API != nil {
		resolved.API = r.resolveAPI(parsed.API)
	}

	// Resolve schemas
	for name, schema := range parsed.Schemas {
		resolvedSchema, err := r.resolveSchema(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve schema %s: %w", name, err)
		}
		resolved.Schemas[name] = resolvedSchema
	}

	// Resolve parameters
	for name, param := range parsed.Parameters {
		resolvedParam, err := r.resolveParameter(param)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve parameter %s: %w", name, err)
		}
		resolved.Parameters[name] = resolvedParam
	}

	// Resolve endpoints
	defaultContentType := ""
	if resolved.API != nil {
		defaultContentType = resolved.API.DefaultContentType
	}
	for _, endpoint := range parsed.Endpoints {
		resolvedEndpoint, err := r.resolveEndpoint(endpoint, resolved.Parameters, resolved.Schemas, defaultContentType)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve endpoint %s %s: %w", endpoint.Method, endpoint.Path, err)
		}
		resolved.Endpoints = append(resolved.Endpoints, resolvedEndpoint)
	}

	return resolved, nil
}

// resolveAPI copies API info (no type resolution needed)
func (r *Resolver) resolveAPI(api *parser.APIInfo) *API {
	resolved := &API{
		Title:           api.Title,
		Version:         api.Version,
		Description:     api.Description,
		TermsOfService:  api.TermsOfService,
		Servers:         make([]*Server, len(api.Servers)),
		SecuritySchemes: make(map[string]*SecurityScheme),
		Security:        make([][]*SecurityRequirement, len(api.Security)),
	}

	// Copy contact
	if api.Contact != nil {
		resolved.Contact = &Contact{
			Name:  api.Contact.Name,
			Email: api.Contact.Email,
			URL:   api.Contact.URL,
		}
	}

	// Copy license
	if api.License != nil {
		resolved.License = &License{
			Name: api.License.Name,
			URL:  api.License.URL,
		}
	}

	// Copy servers
	for i, server := range api.Servers {
		resolved.Servers[i] = &Server{
			URL:         server.URL,
			Description: server.Description,
		}
	}

	// Copy security schemes
	for name, scheme := range api.SecuritySchemes {
		resolved.SecuritySchemes[name] = &SecurityScheme{
			Name:          scheme.Name,
			Type:          scheme.Type,
			Scheme:        scheme.Scheme,
			BearerFormat:  scheme.BearerFormat,
			In:            scheme.In,
			ParameterName: scheme.ParameterName,
			Description:   scheme.Description,
		}
	}

	// Copy security requirements
	for i, reqs := range api.Security {
		resolved.Security[i] = make([]*SecurityRequirement, len(reqs))
		for j, req := range reqs {
			resolved.Security[i][j] = &SecurityRequirement{
				SchemeName: req.SchemeName,
				Scopes:     req.Scopes,
			}
		}
	}

	// Copy tags
	resolved.Tags = make([]*Tag, len(api.Tags))
	for i, tag := range api.Tags {
		resolved.Tags[i] = &Tag{
			Name:        tag.Name,
			Description: tag.Description,
		}
	}

	// Copy default content type
	resolved.DefaultContentType = api.DefaultContentType

	return resolved
}

// resolveSchema resolves a schema by looking up the Go struct and resolving its fields
// schemaNames contains all known @schema type names for detecting unresolved struct references
func (r *Resolver) resolveSchema(schema *parser.Schema) (*Schema, error) {
	resolved := &Schema{
		Name:        schema.Name,
		GoTypeName:  schema.GoTypeName,
		Description: schema.Description,
		Deprecated:  schema.Deprecated,
		Fields:      make([]*Field, 0),
		IsGeneric:   schema.IsGeneric,
		IsTypeAlias: schema.IsTypeAlias,
		AliasOf:     schema.AliasOf,
	}

	// Extract type argument for type aliases (e.g., "DataResponse[User]" -> "User")
	if schema.IsTypeAlias && schema.AliasOf != "" {
		resolved.TypeArg = extractTypeArg(schema.AliasOf)
	}

	// Find the Go struct
	obj := r.pkg.Types.Scope().Lookup(schema.GoTypeName)
	if obj == nil {
		return nil, fmt.Errorf("struct %s not found in package", schema.GoTypeName)
	}

	structType, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%s is not a struct", schema.GoTypeName)
	}

	// Resolve each field (including embedded struct flattening)
	fields, err := r.resolveSchemaFields(structType, schema.Fields, nil)
	if err != nil {
		return nil, err
	}
	resolved.Fields = fields

	return resolved, nil
}

// resolveSchemaFields resolves fields from a struct type, flattening embedded structs.
// visited tracks type names to prevent infinite recursion from circular embedding.
func (r *Resolver) resolveSchemaFields(structType *types.Struct, annotations []*parser.Field, visited map[string]bool) ([]*Field, error) {
	if visited == nil {
		visited = make(map[string]bool)
	}

	var fields []*Field

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		// Handle embedded (anonymous) fields by flattening
		if field.Anonymous() {
			embeddedFields, err := r.flattenEmbeddedField(field, annotations, visited)
			if err != nil {
				return nil, err
			}
			fields = append(fields, embeddedFields...)
			continue
		}

		// Find annotation for this field
		var fieldAnnotation *parser.Field
		for _, f := range annotations {
			if f.GoName == field.Name() {
				fieldAnnotation = f
				break
			}
		}

		// Resolve field type
		resolvedField, err := r.resolveField(field, tag, fieldAnnotation)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve field %s: %w", field.Name(), err)
		}
		// Skip fields that should be omitted (e.g., json:"-")
		if resolvedField == nil {
			continue
		}

		fields = append(fields, resolvedField)
	}

	return fields, nil
}

// unwrapEmbeddedStruct unwraps a type (through pointers and named types) to get
// the underlying struct, with cycle detection via the visited map.
// Returns nil if the type is not a struct or if a cycle is detected.
// The returned cleanup function must be deferred to remove the type from visited.
func unwrapEmbeddedStruct(t types.Type, visited map[string]bool) (st *types.Struct, cleanup func()) {
	cleanup = func() {} // no-op default

	// Unwrap pointer
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	if named, ok := t.(*types.Named); ok {
		typeName := named.Obj().Name()

		// Cycle detection
		if visited[typeName] {
			return nil, cleanup
		}
		visited[typeName] = true
		cleanup = func() { delete(visited, typeName) }

		underlying, ok := named.Underlying().(*types.Struct)
		if !ok {
			return nil, cleanup
		}
		return underlying, cleanup
	}

	if s, ok := t.(*types.Struct); ok {
		return s, cleanup
	}

	return nil, cleanup
}

// flattenEmbeddedField resolves an embedded struct field and returns its flattened fields.
func (r *Resolver) flattenEmbeddedField(field *types.Var, annotations []*parser.Field, visited map[string]bool) ([]*Field, error) {
	embeddedStruct, cleanup := unwrapEmbeddedStruct(field.Type(), visited)
	defer cleanup()
	if embeddedStruct == nil {
		return nil, nil
	}

	return r.resolveSchemaFields(embeddedStruct, annotations, visited)
}

// extractTypeArg extracts the type argument from a generic instantiation
// e.g., "DataResponse[User]" -> "User", "DataResponse[[]User]" -> "[]User"
func extractTypeArg(aliasOf string) string {
	start := strings.Index(aliasOf, "[")
	end := strings.LastIndex(aliasOf, "]")
	if start > 0 && end > start {
		return aliasOf[start+1 : end]
	}
	return ""
}

// resolveParameter resolves a parameter struct
func (r *Resolver) resolveParameter(param *parser.Parameter) (*ParameterStruct, error) {
	resolved := &ParameterStruct{
		Name:       param.Name,
		Type:       string(param.Type),
		GoTypeName: param.GoTypeName,
		Fields:     make([]*Field, 0),
	}

	// Find the Go struct
	obj := r.pkg.Types.Scope().Lookup(param.GoTypeName)
	if obj == nil {
		return nil, fmt.Errorf("struct %s not found in package", param.GoTypeName)
	}

	structType, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%s is not a struct", param.GoTypeName)
	}

	// Resolve each field (including embedded struct flattening)
	fields, err := r.resolveParameterFields(structType, param.Fields, string(param.Type), nil)
	if err != nil {
		return nil, err
	}
	resolved.Fields = fields

	return resolved, nil
}

// resolveParameterFields resolves fields from a parameter struct, flattening embedded structs.
func (r *Resolver) resolveParameterFields(structType *types.Struct, annotations []*parser.Field, paramType string, visited map[string]bool) ([]*Field, error) {
	if visited == nil {
		visited = make(map[string]bool)
	}

	var fields []*Field

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		// Handle embedded (anonymous) fields by flattening
		if field.Anonymous() {
			embeddedFields, err := r.flattenEmbeddedParamField(field, annotations, paramType, visited)
			if err != nil {
				return nil, err
			}
			fields = append(fields, embeddedFields...)
			continue
		}

		// Find annotation for this field
		var fieldAnnotation *parser.Field
		for _, f := range annotations {
			if f.GoName == field.Name() {
				fieldAnnotation = f
				break
			}
		}

		// Resolve field type
		resolvedField, err := r.resolveFieldWithParamType(field, tag, fieldAnnotation, paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve field %s: %w", field.Name(), err)
		}
		// Skip fields that should be omitted
		if resolvedField == nil {
			continue
		}

		fields = append(fields, resolvedField)
	}

	return fields, nil
}

// flattenEmbeddedParamField resolves an embedded struct field for parameters.
func (r *Resolver) flattenEmbeddedParamField(field *types.Var, annotations []*parser.Field, paramType string, visited map[string]bool) ([]*Field, error) {
	embeddedStruct, cleanup := unwrapEmbeddedStruct(field.Type(), visited)
	defer cleanup()
	if embeddedStruct == nil {
		return nil, nil
	}

	return r.resolveParameterFields(embeddedStruct, annotations, paramType, visited)
}

// resolveField resolves a single struct field.
//
// Returns nil, nil when the field should not appear at all: unexported, or
// tagged json:"-".
func (r *Resolver) resolveField(field *types.Var, tag string, annotation *parser.Field) (*Field, error) {
	// Unexported fields cannot be serialized.
	if !field.Exported() {
		return nil, nil
	}

	fieldName := resolveFieldNameFromTag(tag, field.Name())
	if fieldName == "" {
		return nil, nil
	}

	typeRef := r.resolveTypeRef(field.Type())

	// omitempty/omitzero drop a nil pointer rather than encoding null, so such
	// a field can never appear as null on the wire.
	omitted := omitsWhenEmpty(tag, field.Type())

	resolved := &Field{
		Name:     fieldName,
		GoName:   field.Name(),
		GoType:   field.Type().String(),
		Type:     typeRef,
		Format:   typeRef.Format,
		Required: !omitted,
		Nullable: isNullable(field.Type()) && !omitted,
	}

	applyAnnotationOverrides(resolved, annotation)

	return resolved, nil
}

// resolveAnonymousFields resolves the members of an anonymous struct.
//
// Annotations do not reach here yet: the parser harvests @field comments only
// at the top level of a struct, so nested ones are parsed and dropped. That is
// bug #4, fixed separately.
func (r *Resolver) resolveAnonymousFields(structType *types.Struct) []*Field {
	fields := make([]*Field, 0, structType.NumFields())

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		if field.Anonymous() {
			embedded, err := r.flattenEmbeddedField(field, nil, nil)
			if err != nil {
				continue
			}
			fields = append(fields, embedded...)
			continue
		}

		resolved, err := r.resolveField(field, tag, nil)
		if err != nil || resolved == nil {
			continue
		}
		fields = append(fields, resolved)
	}

	return fields
}

// omitsWhenEmpty reports whether the JSON tag actually drops the field for
// empty/zero values of the given type. omitzero omits any zero value,
// including zero structs. omitempty follows encoding/json's isEmptyValue,
// which never considers structs empty (and arrays only at length zero), so
// such fields always appear on the wire despite the tag.
func omitsWhenEmpty(tag string, fieldType types.Type) bool {
	if strings.Contains(tag, "omitzero") {
		return true
	}
	return strings.Contains(tag, "omitempty") && canBeEmpty(fieldType)
}

// canBeEmpty mirrors encoding/json's isEmptyValue: reports whether some value
// of the type is ever considered empty by omitempty.
func canBeEmpty(fieldType types.Type) bool {
	switch u := fieldType.Underlying().(type) {
	case *types.Pointer, *types.Interface, *types.Map, *types.Slice:
		return true
	case *types.Basic:
		return u.Info()&(types.IsBoolean|types.IsNumeric|types.IsString) != 0
	case *types.Array:
		return u.Len() == 0
	default:
		return false
	}
}

// resolveFieldWithParamType resolves a field for a parameter with the appropriate struct tag
// Returns nil, nil if the field should be skipped (e.g., json:"-" for schema fields or unexported fields)
func (r *Resolver) resolveFieldWithParamType(field *types.Var, tag string, annotation *parser.Field, paramType string) (*Field, error) {
	// Skip unexported (private) fields - they cannot be serialized
	if !field.Exported() {
		return nil, nil
	}

	// Extract name from the appropriate struct tag based on parameter type
	var tagName string
	switch paramType {
	case "path":
		tagName = extractTagName(tag, "path")
	case "query":
		tagName = extractTagName(tag, "query")
	case "header":
		tagName = extractTagName(tag, "header")
	case "cookie":
		tagName = extractTagName(tag, "cookie")
	default:
		// For schemas, use tag fallback chain (json -> xml -> Go field name)
		tagName = resolveFieldNameFromTag(tag, field.Name())
		if tagName == "" {
			// Field should be skipped (e.g., json:"-")
			return nil, nil
		}
	}

	// For parameter types, handle "-" and empty as fallback to Go field name
	if tagName == "-" || tagName == "" {
		tagName = field.Name()
	}

	resolved := &Field{
		Name:   tagName,
		GoName: field.Name(),
		GoType: field.Type().String(),
	}

	// Check if field is required based on parameter type.
	// Path parameters are always required. Query, header, and cookie parameters
	// are optional by default and only required if the tag contains ",required".
	// Schema/JSON fields use the omitempty/omitzero logic.
	switch paramType {
	case "path":
		resolved.Required = true
	case "query", "header", "cookie":
		resolved.Required = strings.Contains(tag, ",required")
	default:
		resolved.Required = !omitsWhenEmpty(tag, field.Type())
	}

	resolved.Type = r.resolveTypeRef(field.Type())
	resolved.Format = resolved.Type.Format

	// Parameters serialize as plain strings, which cannot represent null,
	// so pointer-ness never implies nullable — a pointer only lets the
	// handler distinguish absent from zero. @nullable true can still opt in.
	switch paramType {
	case "path", "query", "header", "cookie":
		resolved.Nullable = false
	default:
		resolved.Nullable = isNullable(field.Type())
	}

	applyAnnotationOverrides(resolved, annotation)

	return resolved, nil
}

// applyAnnotationOverrides applies @field annotation values onto a resolved field
func applyAnnotationOverrides(resolved *Field, annotation *parser.Field) {
	if annotation == nil {
		return
	}
	if annotation.Description != "" {
		resolved.Description = annotation.Description
	}
	if annotation.Format != "" {
		resolved.Format = annotation.Format
	}
	if annotation.Example != "" {
		resolved.Example = annotation.Example
	}
	if annotation.Default != "" {
		resolved.Default = annotation.Default
	}
	if annotation.Pattern != "" {
		resolved.Pattern = annotation.Pattern
	}
	if len(annotation.Enum) > 0 {
		resolved.Enum = annotation.Enum
	}
	if annotation.MinLength != nil {
		resolved.MinLength = annotation.MinLength
	}
	if annotation.MaxLength != nil {
		resolved.MaxLength = annotation.MaxLength
	}
	if annotation.MinItems != nil {
		resolved.MinItems = annotation.MinItems
	}
	if annotation.MaxItems != nil {
		resolved.MaxItems = annotation.MaxItems
	}
	if annotation.UniqueItems {
		resolved.UniqueItems = true
	}
	if annotation.Minimum != nil {
		resolved.Minimum = annotation.Minimum
	}
	if annotation.Maximum != nil {
		resolved.Maximum = annotation.Maximum
	}
	if annotation.ExclusiveMinimum != nil {
		resolved.ExclusiveMinimum = annotation.ExclusiveMinimum
	}
	if annotation.ExclusiveMaximum != nil {
		resolved.ExclusiveMaximum = annotation.ExclusiveMaximum
	}
	if annotation.Required != nil {
		resolved.Required = *annotation.Required
	}
	if annotation.Nullable != nil {
		resolved.Nullable = *annotation.Nullable
	}
	if annotation.Deprecated {
		resolved.Deprecated = true
	}
	if annotation.ReadOnly {
		resolved.ReadOnly = true
	}
	if annotation.WriteOnly {
		resolved.WriteOnly = true
	}
}

// resolveEndpoint resolves an endpoint
func (r *Resolver) resolveEndpoint(endpoint *parser.Endpoint, parameters map[string]*ParameterStruct, schemas map[string]*Schema, defaultContentType string) (*Endpoint, error) {
	resolved := &Endpoint{
		FuncName:        endpoint.FuncName,
		Method:          endpoint.Method,
		Path:            endpoint.Path,
		OperationID:     endpoint.OperationID,
		Summary:         endpoint.Summary,
		Description:     endpoint.Description,
		Tags:            endpoint.Tags,
		Deprecated:      endpoint.Deprecated,
		Auth:            endpoint.Auth,
		Responses:       make(map[string]*Response),
		PathParams:      make([]*ParameterStruct, 0),
		QueryParams:     make([]*ParameterStruct, 0),
		HeaderParams:    make([]*ParameterStruct, 0),
		CookieParams:    make([]*ParameterStruct, 0),
		InlineResponses: make(map[string]*InlineBody),
	}

	// Resolve request body
	if endpoint.Request != nil && endpoint.Request.Body != nil {
		contentType := endpoint.Request.ContentType
		if contentType == "" {
			contentType = defaultContentType
		}
		if contentType == "" {
			contentType = "application/json" // Fallback
		}
		resolved.Request = &RequestBody{
			ContentType: contentType,
			Body:        r.resolveBody(endpoint.Request.Body, schemas),
			Required:    true, // Default to required
		}
	}

	// Resolve responses
	for statusCode, response := range endpoint.Responses {
		contentType := response.ContentType
		// Only apply default if response has a body
		if contentType == "" && response.Body != nil {
			contentType = defaultContentType
		}
		if contentType == "" && response.Body != nil {
			contentType = "application/json" // Fallback
		}

		resolvedResponse := &Response{
			StatusCode:  response.StatusCode,
			Description: response.Description,
			ContentType: contentType,
			Body:        r.resolveBody(response.Body, schemas),
		}

		// Resolve response header references
		for _, ref := range response.HeaderParams {
			if param, ok := parameters[ref]; ok {
				resolvedResponse.Headers = append(resolvedResponse.Headers, param)
			}
		}

		resolved.Responses[statusCode] = resolvedResponse
	}

	// Resolve parameter references
	for _, ref := range endpoint.PathParams {
		if param, ok := parameters[ref]; ok {
			resolved.PathParams = append(resolved.PathParams, param)
		}
	}

	for _, ref := range endpoint.QueryParams {
		if param, ok := parameters[ref]; ok {
			resolved.QueryParams = append(resolved.QueryParams, param)
		}
	}

	for _, ref := range endpoint.HeaderParams {
		if param, ok := parameters[ref]; ok {
			resolved.HeaderParams = append(resolved.HeaderParams, param)
		}
	}

	for _, ref := range endpoint.CookieParams {
		if param, ok := parameters[ref]; ok {
			resolved.CookieParams = append(resolved.CookieParams, param)
		}
	}

	// Resolve inline declarations from function body
	if r.parsed.FuncInlines != nil {
		if inlines := r.parsed.FuncInlines[endpoint.FuncName]; inlines != nil {
			if err := r.resolveInlineDeclarations(resolved, inlines, parameters, schemas, defaultContentType); err != nil {
				return nil, fmt.Errorf("failed to resolve inline declarations: %w", err)
			}
		}
	}

	return resolved, nil
}

// extractTagName extracts a field name from a specific struct tag key
func extractTagName(tag string, key string) string {
	// Parse struct tag
	st := reflect.StructTag(tag)
	value := st.Get(key)
	if value == "" {
		return ""
	}

	// Split by comma to remove options like omitempty
	parts := strings.Split(value, ",")
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

// SupportedTags defines struct tags checked for field names (in fallback order).
// To add support for additional tags, append them to this slice.
var SupportedTags = []string{"json", "xml"}

// resolveFieldNameFromTag determines the field name using the fallback chain.
// It checks tags in order (json, xml) and falls back to the Go field name.
//
// Fallback logic:
//   - If tag is "-" → return "" (skip field entirely)
//   - If tag not present or name part is empty → continue to next tag
//   - If name part exists → use it
//   - If no tags have a name → use Go field name
//
// Returns empty string "" if field should be skipped.
func resolveFieldNameFromTag(tag string, goFieldName string) string {
	st := reflect.StructTag(tag)

	for _, tagKey := range SupportedTags {
		tagValue := st.Get(tagKey)
		if tagValue == "" {
			// Tag not present or empty, continue to next tag
			continue
		}
		if tagValue == "-" {
			// Explicit skip - field should be omitted entirely
			return ""
		}
		// Extract name part (before comma)
		name := tagValue
		if idx := strings.Index(tagValue, ","); idx != -1 {
			name = tagValue[:idx]
		}
		if name == "" {
			// Name part is empty (e.g., `json:",omitempty"`), continue to next tag
			continue
		}
		return name
	}
	// No tags found with a name, use Go field name
	return goFieldName
}

// resolveBody resolves a parser.Body to a Body
func (r *Resolver) resolveBody(body *parser.Body, schemas map[string]*Schema) *Body {
	if body == nil {
		return nil
	}

	resolved := &Body{
		Schema: body.Schema,
		Type:   r.resolveBodyType(body.Schema),
	}

	// Resolve bind target if present
	if body.Bind != nil {
		resolved.Bind = r.resolveBindTarget(body.Bind, schemas)
	}

	return resolved
}

// resolveBodyType turns the text of a @body annotation into a shape.
//
// The text is written by hand — "User", "[]User", "map[string]string" — so it
// is a small grammar rather than a Go type, but deciding whether "string" names
// a primitive or a schema is still type reasoning, and it belongs here rather
// than in the generator.
func (r *Resolver) resolveBodyType(schema string) *TypeRef {
	schema = strings.TrimSpace(schema)

	if elem, ok := strings.CutPrefix(schema, "[]"); ok {
		return &TypeRef{Shape: ShapeArray, Elem: r.resolveBodyType(elem)}
	}
	if elem, ok := strings.CutPrefix(schema, "map[string]"); ok {
		return &TypeRef{Shape: ShapeMap, Elem: r.resolveBodyType(elem)}
	}

	if primitive, format, ok := primitiveByName(schema); ok {
		return &TypeRef{Shape: ShapeScalar, Type: primitive, Format: format}
	}

	// Anything else names a schema. Whether that schema exists is the
	// validator's question, so an unknown name still resolves to a reference
	// and gets reported with a message about the name rather than the shape.
	return &TypeRef{Shape: ShapeRef, Ref: schema}
}

// resolveBindTarget resolves a parser.BindTarget to a BindTarget
func (r *Resolver) resolveBindTarget(bind *parser.BindTarget, schemas map[string]*Schema) *BindTarget {
	if bind == nil {
		return nil
	}

	resolved := &BindTarget{
		Wrapper: bind.Wrapper,
		Field:   bind.Field,
	}

	// Look up the wrapper schema
	if wrapperSchema, ok := schemas[bind.Wrapper]; ok {
		resolved.WrapperSchema = wrapperSchema
	}

	return resolved
}

// resolveInlineDeclarations resolves inline struct declarations from function body.
// @query/@path/@header/@cookie are repeatable per schema — every inline struct in
// each category is resolved and its fields concatenated into one *InlineParams
// (the downstream generator/validator consume a flat Fields list).
func (r *Resolver) resolveInlineDeclarations(endpoint *Endpoint, inlines *parser.FuncInlineInfo, parameters map[string]*ParameterStruct, schemas map[string]*Schema, defaultContentType string) error {
	mergedPath, err := r.mergeInlineParams(inlines.Path, "path")
	if err != nil {
		return err
	}
	endpoint.InlinePathParams = mergedPath

	mergedQuery, err := r.mergeInlineParams(inlines.Query, "query")
	if err != nil {
		return err
	}
	endpoint.InlineQueryParams = mergedQuery

	mergedHeader, err := r.mergeInlineParams(inlines.Header, "header")
	if err != nil {
		return err
	}
	endpoint.InlineHeaderParams = mergedHeader

	mergedCookie, err := r.mergeInlineParams(inlines.Cookie, "cookie")
	if err != nil {
		return err
	}
	endpoint.InlineCookieParams = mergedCookie

	// Resolve inline request body
	if inlines.Request != nil {
		// Step 1: Parse inline annotation using InlineAnnotationSchema
		parsed, err := ParseInlineDeclaration(inlines.Request.Comment, "request")
		if err != nil {
			return fmt.Errorf("failed to parse inline request: %w", err)
		}

		// Step 2: Resolve inline body using parsed annotation
		body, err := r.resolveInlineBody(inlines.Request, parsed, nil, schemas, defaultContentType)
		if err != nil {
			return fmt.Errorf("failed to resolve inline request body: %w", err)
		}
		endpoint.InlineRequest = body
	}

	// Resolve inline responses
	for statusCode, respInfo := range inlines.Responses {
		// Step 1: Parse inline annotation using InlineAnnotationSchema
		parsed, err := ParseInlineDeclaration(respInfo.Comment, "response")
		if err != nil {
			return fmt.Errorf("failed to parse inline response %s: %w", statusCode, err)
		}

		// Step 2: Resolve inline body using parsed annotation (with parameters for header resolution)
		body, err := r.resolveInlineBody(respInfo, parsed, parameters, schemas, defaultContentType)
		if err != nil {
			return fmt.Errorf("failed to resolve inline response %s: %w", statusCode, err)
		}
		endpoint.InlineResponses[statusCode] = body
	}

	return nil
}

// inlineStructType looks up the *types.Struct for an inline var/type declaration
// via TypesInfo.Defs. Works for both `var x struct{...}` (anonymous struct type)
// and `type X struct{...}` (named type whose underlying is a struct) — calling
// .Underlying() on either case returns the *types.Struct directly.
func (r *Resolver) inlineStructType(info *parser.InlineStructInfo) (*types.Struct, error) {
	if info == nil || info.Ident == nil {
		return nil, fmt.Errorf("inline struct missing identifier")
	}
	if r.pkg == nil || r.pkg.TypesInfo == nil {
		return nil, fmt.Errorf("type info unavailable for inline struct %q", info.VarName)
	}
	obj := r.pkg.TypesInfo.Defs[info.Ident]
	if obj == nil {
		return nil, fmt.Errorf("could not resolve type for inline struct %q", info.VarName)
	}
	st, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("inline declaration %q is not a struct", info.VarName)
	}
	return st, nil
}

// mergeInlineParams resolves every inline struct in a repeatable category (e.g.
// all inline @query structs declared in one handler) and concatenates their fields
// into a single *InlineParams. Returns nil when no inline structs exist
// for the category — preserving the "nil means none" contract downstream code relies on.
// OpenAPI emits one flat parameters array per operation regardless of how many
// inline structs the handler declared, so the merge happens here.
func (r *Resolver) mergeInlineParams(infos []*parser.InlineStructInfo, paramType string) (*InlineParams, error) {
	if len(infos) == 0 {
		return nil, nil
	}
	merged := &InlineParams{}
	for _, info := range infos {
		params, err := r.resolveInlineParams(info, paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve inline %s params %q: %w", paramType, info.VarName, err)
		}
		if params == nil {
			continue
		}
		merged.Fields = append(merged.Fields, params.Fields...)
	}
	if len(merged.Fields) == 0 {
		return nil, nil
	}
	return merged, nil
}

// resolveInlineParams resolves an inline parameter struct via the same
// *types.Struct + parsed *Field path as named parameter structs — no AST walk,
// no resolver-side @field parsing.
func (r *Resolver) resolveInlineParams(info *parser.InlineStructInfo, paramType string) (*InlineParams, error) {
	if info == nil {
		return nil, nil
	}
	structType, err := r.inlineStructType(info)
	if err != nil {
		return nil, err
	}

	// Parameters don't support anonymous struct fields, so pass nil for schemaNames.
	fields, err := r.resolveParameterFields(structType, info.Fields, paramType, nil)
	if err != nil {
		return nil, err
	}

	return &InlineParams{
		Fields: fields,
	}, nil
}

// resolveInlineBody resolves an inline request/response body struct using parsed annotation.
// Uses the same *types.Struct + parsed *Field path as @schema structs.
func (r *Resolver) resolveInlineBody(info *parser.InlineStructInfo, parsed *parser.ParsedAnnotation, parameters map[string]*ParameterStruct, schemas map[string]*Schema, defaultContentType string) (*InlineBody, error) {
	if info == nil {
		return nil, nil
	}
	structType, err := r.inlineStructType(info)
	if err != nil {
		return nil, err
	}

	// Build schemaNames for anonymous struct resolution
	schemaNames := make(map[string]bool)
	for name := range schemas {
		schemaNames[name] = true
	}

	fields, err := r.resolveSchemaFields(structType, info.Fields, nil)
	if err != nil {
		return nil, err
	}

	resolved := &InlineBody{
		Fields: fields,
	}

	// Use parsed annotation (already validated by inline parser)
	if parsed != nil {
		// Content type
		if ct := parsed.GetChildValue("@contentType"); ct != "" {
			resolved.ContentType = parser.ExpandContentType(ct)
		}

		// Description
		resolved.Description = parsed.GetChildValue("@description")

		// Bind
		if bindValue := parsed.GetChildValue("@bind"); bindValue != "" {
			bindTarget := parser.ParseBindTarget(bindValue)
			resolved.Bind = r.resolveBindTarget(bindTarget, schemas)
		}

		// Resolve header references (response only)
		if parameters != nil {
			for _, headerChild := range parsed.GetRepeatedChildren("@header") {
				if param, ok := parameters[headerChild.Value]; ok {
					resolved.Headers = append(resolved.Headers, param)
				}
			}
		}
	}

	// Apply defaults for content type
	if resolved.ContentType == "" {
		resolved.ContentType = defaultContentType
	}
	if resolved.ContentType == "" {
		resolved.ContentType = "application/json"
	}

	return resolved, nil
}
