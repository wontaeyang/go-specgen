package resolver

import (
	"fmt"
	"go/ast"
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
	packagePath string
	pkg         *packages.Package
	typeCache   map[string]*TypeInfo
	comments    *parser.PackageComments // For inline type resolution
}

// TypeInfo contains resolved type information
type TypeInfo struct {
	OpenAPIType string
	Format      string
	IsArray     bool
	ItemsType   string
	IsNullable  bool
	IsAnyValue  bool // true for any/interface{} types
}

// NewResolver creates a new resolver for the given package
// If comments is provided, it uses the package from comments for type resolution,
// which enables inline struct resolution. Otherwise, it loads the package itself.
func NewResolver(packagePath string, comments *parser.PackageComments) (*Resolver, error) {
	var pkg *packages.Package

	// Use the package from comments if available (enables inline type resolution)
	if comments != nil && comments.Pkg != nil {
		pkg = comments.Pkg
	} else {
		// Load the package ourselves
		cfg := &packages.Config{
			Mode: packages.NeedName |
				packages.NeedFiles |
				packages.NeedSyntax |
				packages.NeedTypes |
				packages.NeedTypesInfo,
		}

		pkgs, err := packages.Load(cfg, packagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load package: %w", err)
		}

		if len(pkgs) == 0 {
			return nil, fmt.Errorf("no packages found")
		}

		pkg = pkgs[0]
		if len(pkg.Errors) > 0 {
			return nil, fmt.Errorf("package has errors: %v", pkg.Errors)
		}
	}

	return &Resolver{
		packagePath: packagePath,
		pkg:         pkg,
		typeCache:   make(map[string]*TypeInfo),
		comments:    comments,
	}, nil
}

// Resolve resolves all types in the parsed package
func (r *Resolver) Resolve(parsed *parser.ParsedPackage) (*ResolvedPackage, error) {
	resolved := &ResolvedPackage{
		PackageName: parsed.PackageName,
		Schemas:     make(map[string]*ResolvedSchema),
		Parameters:  make(map[string]*ResolvedParameter),
		Endpoints:   make([]*ResolvedEndpoint, 0),
	}

	// Resolve API info (no type resolution needed, just copy)
	if parsed.API != nil {
		resolved.API = r.resolveAPI(parsed.API)
	}

	// Build schema names map for detecting unresolved struct references
	schemaNames := make(map[string]bool)
	for name := range parsed.Schemas {
		schemaNames[name] = true
	}

	// Resolve schemas
	for name, schema := range parsed.Schemas {
		resolvedSchema, err := r.resolveSchema(schema, schemaNames)
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
func (r *Resolver) resolveAPI(api *parser.APIInfo) *ResolvedAPI {
	resolved := &ResolvedAPI{
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
func (r *Resolver) resolveSchema(schema *parser.Schema, schemaNames map[string]bool) (*ResolvedSchema, error) {
	resolved := &ResolvedSchema{
		Name:        schema.Name,
		GoTypeName:  schema.GoTypeName,
		Description: schema.Description,
		Deprecated:  schema.Deprecated,
		Fields:      make([]*ResolvedField, 0),
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

	// Resolve each field
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		// Find annotation for this field
		var fieldAnnotation *parser.Field
		for _, f := range schema.Fields {
			if f.GoName == field.Name() {
				fieldAnnotation = f
				break
			}
		}

		// Resolve field type
		resolvedField, err := r.resolveField(field, tag, fieldAnnotation, schemaNames)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve field %s: %w", field.Name(), err)
		}
		// Skip fields that should be omitted (e.g., json:"-")
		if resolvedField == nil {
			continue
		}

		resolved.Fields = append(resolved.Fields, resolvedField)
	}

	return resolved, nil
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
func (r *Resolver) resolveParameter(param *parser.Parameter) (*ResolvedParameter, error) {
	resolved := &ResolvedParameter{
		Name:       param.Name,
		Type:       string(param.Type),
		GoTypeName: param.GoTypeName,
		Fields:     make([]*ResolvedField, 0),
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

	// Resolve each field
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		// Find annotation for this field
		var fieldAnnotation *parser.Field
		for _, f := range param.Fields {
			if f.GoName == field.Name() {
				fieldAnnotation = f
				break
			}
		}

		// Resolve field type
		resolvedField, err := r.resolveFieldWithParamType(field, tag, fieldAnnotation, string(param.Type))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve field %s: %w", field.Name(), err)
		}
		// Skip fields that should be omitted (e.g., json:"-" for schema fields)
		if resolvedField == nil {
			continue
		}

		resolved.Fields = append(resolved.Fields, resolvedField)
	}

	return resolved, nil
}

// resolveField resolves a single struct field
// schemaNames contains the names of all known @schema types for detecting unresolved struct references
// Returns nil, nil if the field should be skipped (e.g., json:"-" or unexported fields)
func (r *Resolver) resolveField(field *types.Var, tag string, annotation *parser.Field, schemaNames map[string]bool) (*ResolvedField, error) {
	// Skip unexported (private) fields - they cannot be serialized
	if !field.Exported() {
		return nil, nil
	}

	// Resolve field name using tag fallback chain (json -> xml -> Go field name)
	// Returns "" if field should be skipped
	fieldName := resolveFieldNameFromTag(tag, field.Name())
	if fieldName == "" {
		// Field should be skipped (e.g., json:"-")
		return nil, nil
	}

	resolved := &ResolvedField{
		Name:   fieldName,
		GoName: field.Name(),
		GoType: field.Type().String(),
	}

	// Check if field is required/nullable from JSON tag
	resolved.Required = !strings.Contains(tag, "omitempty")

	// Check for anonymous struct types and resolve their fields inline
	if inlineFields := r.resolveAnonymousStruct(field.Type(), schemaNames); inlineFields != nil {
		resolved.InlineFields = inlineFields
		resolved.OpenAPIType = "object"
	} else if itemFields := r.resolveSliceOfAnonymousStruct(field.Type(), schemaNames); itemFields != nil {
		// Slice of anonymous struct
		resolved.IsArray = true
		resolved.OpenAPIType = "array"
		resolved.ItemsType = "object"
		resolved.ItemsInlineFields = itemFields
	} else if mapValueFields := r.resolveMapOfAnonymousStruct(field.Type(), schemaNames); mapValueFields != nil {
		// Map with anonymous struct values
		resolved.IsMap = true
		resolved.OpenAPIType = "object"
		resolved.MapValueInlineFields = mapValueFields
	} else {
		// Check for unresolved struct types (named structs not in @schema)
		r.checkUnresolvedStruct(field.Type(), resolved, schemaNames)

		// Resolve Go type to OpenAPI type
		typeInfo := r.resolveType(field.Type())
		resolved.OpenAPIType = typeInfo.OpenAPIType
		resolved.Format = typeInfo.Format
		resolved.IsArray = typeInfo.IsArray
		resolved.ItemsType = typeInfo.ItemsType
		resolved.Nullable = typeInfo.IsNullable
		resolved.IsAnyValue = typeInfo.IsAnyValue
	}

	// Apply annotation overrides if present
	if annotation != nil {
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
		resolved.Deprecated = annotation.Deprecated
	}

	return resolved, nil
}

// resolveAnonymousStruct checks if a type is an anonymous struct and resolves its fields inline
// Returns nil if the type is not an anonymous struct
func (r *Resolver) resolveAnonymousStruct(t types.Type, schemaNames map[string]bool) []*ResolvedField {
	// Unwrap pointer
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Check if this is an anonymous struct (a *types.Struct without being wrapped in *types.Named)
	structType, ok := t.(*types.Struct)
	if !ok {
		return nil
	}

	// This is an anonymous struct - resolve its fields
	fields := make([]*ResolvedField, 0, structType.NumFields())

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		// Skip unexported (private) fields - they cannot be serialized
		if !field.Exported() {
			continue
		}

		// Resolve field name using tag fallback chain (json -> xml -> Go field name)
		// Returns "" if field should be skipped
		fieldName := resolveFieldNameFromTag(tag, field.Name())
		if fieldName == "" {
			// Field should be skipped (e.g., json:"-")
			continue
		}

		resolvedField := &ResolvedField{
			Name:   fieldName,
			GoName: field.Name(),
			GoType: field.Type().String(),
		}

		// Check if field is required from JSON tag
		resolvedField.Required = !strings.Contains(tag, "omitempty")

		// Check for nested anonymous structs (recursive)
		if nestedFields := r.resolveAnonymousStruct(field.Type(), schemaNames); nestedFields != nil {
			resolvedField.InlineFields = nestedFields
			resolvedField.OpenAPIType = "object"
		} else if itemFields := r.resolveSliceOfAnonymousStruct(field.Type(), schemaNames); itemFields != nil {
			// Nested slice of anonymous struct
			resolvedField.IsArray = true
			resolvedField.OpenAPIType = "array"
			resolvedField.ItemsType = "object"
			resolvedField.ItemsInlineFields = itemFields
		} else if mapValueFields := r.resolveMapOfAnonymousStruct(field.Type(), schemaNames); mapValueFields != nil {
			// Nested map with anonymous struct values
			resolvedField.IsMap = true
			resolvedField.OpenAPIType = "object"
			resolvedField.MapValueInlineFields = mapValueFields
		} else {
			// Check for unresolved struct types
			r.checkUnresolvedStruct(field.Type(), resolvedField, schemaNames)

			// Resolve Go type to OpenAPI type
			typeInfo := r.resolveType(field.Type())
			resolvedField.OpenAPIType = typeInfo.OpenAPIType
			resolvedField.Format = typeInfo.Format
			resolvedField.IsArray = typeInfo.IsArray
			resolvedField.ItemsType = typeInfo.ItemsType
			resolvedField.Nullable = typeInfo.IsNullable
			resolvedField.IsAnyValue = typeInfo.IsAnyValue
		}

		fields = append(fields, resolvedField)
	}

	return fields
}

// resolveSliceOfAnonymousStruct checks if a type is a slice/array of anonymous struct
// and resolves the element's fields inline. Returns nil if not a slice of anonymous struct.
func (r *Resolver) resolveSliceOfAnonymousStruct(t types.Type, schemaNames map[string]bool) []*ResolvedField {
	// Unwrap pointer
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Check if this is a slice
	slice, ok := t.(*types.Slice)
	if !ok {
		return nil
	}

	// Check if the element type is an anonymous struct
	return r.resolveAnonymousStruct(slice.Elem(), schemaNames)
}

// resolveMapOfAnonymousStruct checks if a type is a map with anonymous struct values
// and resolves the value's fields inline. Returns nil if not a map of anonymous struct.
func (r *Resolver) resolveMapOfAnonymousStruct(t types.Type, schemaNames map[string]bool) []*ResolvedField {
	// Unwrap pointer
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Check if this is a map
	mapType, ok := t.(*types.Map)
	if !ok {
		return nil
	}

	// Check if the value type is an anonymous struct
	return r.resolveAnonymousStruct(mapType.Elem(), schemaNames)
}

// checkUnresolvedStruct checks if a type is a named struct that is not a known @schema
// and marks the resolved field accordingly
func (r *Resolver) checkUnresolvedStruct(t types.Type, resolved *ResolvedField, schemaNames map[string]bool) {
	// Unwrap pointer
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Check for slice/array - check element type
	if slice, ok := t.(*types.Slice); ok {
		r.checkUnresolvedStruct(slice.Elem(), resolved, schemaNames)
		return
	}

	// Check for map - check value type
	if mapType, ok := t.(*types.Map); ok {
		r.checkUnresolvedStruct(mapType.Elem(), resolved, schemaNames)
		return
	}

	// Check for named types (could be struct or alias to slice/map)
	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		underlying := named.Underlying()

		// Skip special standard library types (time.Time, url.URL, etc.)
		// These are handled specially by the resolver
		if obj.Pkg() != nil && isSpecialType(obj.Pkg().Path(), obj.Name()) {
			return
		}

		// Check if underlying is a struct
		if _, isStruct := underlying.(*types.Struct); isStruct {
			typeName := obj.Name()
			// If not in known schemas, mark as unresolved
			if !schemaNames[typeName] {
				resolved.IsUnresolvedStruct = true
				resolved.UnresolvedTypeName = typeName
			}
			return
		}

		// If underlying is slice/map, recurse into element type
		// This handles custom types like `type Addresses []Address`
		r.checkUnresolvedStruct(underlying, resolved, schemaNames)
	}
}

// resolveFieldWithParamType resolves a field for a parameter with the appropriate struct tag
// Returns nil, nil if the field should be skipped (e.g., json:"-" for schema fields or unexported fields)
func (r *Resolver) resolveFieldWithParamType(field *types.Var, tag string, annotation *parser.Field, paramType string) (*ResolvedField, error) {
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

	resolved := &ResolvedField{
		Name:   tagName,
		GoName: field.Name(),
		GoType: field.Type().String(),
	}

	// Check if field is required/nullable from tag
	resolved.Required = !strings.Contains(tag, "omitempty")

	// Resolve Go type to OpenAPI type
	typeInfo := r.resolveType(field.Type())
	resolved.OpenAPIType = typeInfo.OpenAPIType
	resolved.Format = typeInfo.Format
	resolved.IsArray = typeInfo.IsArray
	resolved.ItemsType = typeInfo.ItemsType
	resolved.Nullable = typeInfo.IsNullable
	resolved.IsAnyValue = typeInfo.IsAnyValue

	// Apply annotation overrides if present
	if annotation != nil {
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
		if annotation.Nullable {
			resolved.Nullable = true
		}
		if annotation.Deprecated {
			resolved.Deprecated = true
		}
	}

	return resolved, nil
}

// resolveType resolves a Go type to OpenAPI type information
func (r *Resolver) resolveType(t types.Type) *TypeInfo {
	// Check cache first
	typeStr := t.String()
	if cached, ok := r.typeCache[typeStr]; ok {
		return cached
	}

	info := &TypeInfo{}

	// Handle pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		info.IsNullable = true
		t = ptr.Elem()
	}

	// Handle slices/arrays
	if slice, ok := t.(*types.Slice); ok {
		info.IsArray = true
		elemInfo := r.resolveType(slice.Elem())
		info.ItemsType = elemInfo.OpenAPIType
		info.OpenAPIType = "array"
		r.typeCache[typeStr] = info
		return info
	}

	// Handle type aliases (e.g., "any" is an alias for interface{})
	if alias, ok := t.(*types.Alias); ok {
		// Recurse on the aliased type
		return r.resolveType(alias.Rhs())
	}

	// Handle named types
	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		pkgPath := ""
		if obj.Pkg() != nil {
			pkgPath = obj.Pkg().Path()
		}

		// Check for special standard library types
		if specialType := resolveSpecialType(pkgPath, obj.Name()); specialType != nil {
			info.OpenAPIType = specialType.openAPIType
			info.Format = specialType.format
			r.typeCache[typeStr] = info
			return info
		}

		// Recurse on underlying type
		return r.resolveType(named.Underlying())
	}

	// Handle basic types
	if basic, ok := t.(*types.Basic); ok {
		switch basic.Kind() {
		case types.Bool:
			info.OpenAPIType = "boolean"
		case types.Int, types.Int8, types.Int16, types.Int32:
			info.OpenAPIType = "integer"
			if basic.Kind() == types.Int32 {
				info.Format = "int32"
			}
		case types.Int64:
			info.OpenAPIType = "integer"
			info.Format = "int64"
		case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			info.OpenAPIType = "integer"
		case types.Float32:
			info.OpenAPIType = "number"
			info.Format = "float"
		case types.Float64:
			info.OpenAPIType = "number"
			info.Format = "double"
		case types.String:
			info.OpenAPIType = "string"
		default:
			// Unknown basic type, default to string
			info.OpenAPIType = "string"
		}
	} else if _, ok := t.(*types.Interface); ok {
		// Interface type (any/interface{}) - any JSON value
		info.IsAnyValue = true
	} else {
		// Unknown type, default to string
		info.OpenAPIType = "string"
	}

	r.typeCache[typeStr] = info
	return info
}

// resolveEndpoint resolves an endpoint
func (r *Resolver) resolveEndpoint(endpoint *parser.Endpoint, parameters map[string]*ResolvedParameter, schemas map[string]*ResolvedSchema, defaultContentType string) (*ResolvedEndpoint, error) {
	resolved := &ResolvedEndpoint{
		FuncName:        endpoint.FuncName,
		Method:          endpoint.Method,
		Path:            endpoint.Path,
		OperationID:     endpoint.OperationID,
		Summary:         endpoint.Summary,
		Description:     endpoint.Description,
		Tags:            endpoint.Tags,
		Deprecated:      endpoint.Deprecated,
		Auth:            endpoint.Auth,
		Responses:       make(map[string]*ResolvedResponse),
		PathParams:      make([]*ResolvedParameter, 0),
		QueryParams:     make([]*ResolvedParameter, 0),
		HeaderParams:    make([]*ResolvedParameter, 0),
		CookieParams:    make([]*ResolvedParameter, 0),
		InlineResponses: make(map[string]*ResolvedInlineBody),
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
		resolved.Request = &ResolvedRequestBody{
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

		resolvedResponse := &ResolvedResponse{
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
	if r.comments != nil && r.comments.FuncInlines != nil {
		if inlines := r.comments.FuncInlines[endpoint.FuncName]; inlines != nil {
			if err := r.resolveInlineDeclarations(resolved, inlines, parameters, schemas, defaultContentType); err != nil {
				return nil, fmt.Errorf("failed to resolve inline declarations: %w", err)
			}
		}
	}

	return resolved, nil
}

// extractJSONName extracts the JSON field name from a struct tag
func extractJSONName(tag string) string {
	// Parse struct tag
	st := reflect.StructTag(tag)
	jsonTag := st.Get("json")
	if jsonTag == "" {
		return ""
	}

	// Split by comma to remove options like omitempty
	parts := strings.Split(jsonTag, ",")
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
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

// resolveBody resolves a parser.Body to a ResolvedBody
func (r *Resolver) resolveBody(body *parser.Body, schemas map[string]*ResolvedSchema) *ResolvedBody {
	if body == nil {
		return nil
	}

	resolved := &ResolvedBody{
		Schema: body.Schema,
	}

	// Parse schema to determine if it's an array or map
	schema := strings.TrimSpace(body.Schema)
	if strings.HasPrefix(schema, "[]") {
		resolved.IsArray = true
		resolved.ElementType = strings.TrimPrefix(schema, "[]")
	} else if strings.HasPrefix(schema, "map[string]") {
		resolved.IsMap = true
		resolved.ElementType = strings.TrimPrefix(schema, "map[string]")
	} else {
		resolved.ElementType = schema
	}

	// Resolve bind target if present
	if body.Bind != nil {
		resolved.Bind = r.resolveBindTarget(body.Bind, schemas)
	}

	return resolved
}

// resolveBindTarget resolves a parser.BindTarget to a ResolvedBindTarget
func (r *Resolver) resolveBindTarget(bind *parser.BindTarget, schemas map[string]*ResolvedSchema) *ResolvedBindTarget {
	if bind == nil {
		return nil
	}

	resolved := &ResolvedBindTarget{
		Wrapper: bind.Wrapper,
		Field:   bind.Field,
	}

	// Look up the wrapper schema
	if wrapperSchema, ok := schemas[bind.Wrapper]; ok {
		resolved.WrapperSchema = wrapperSchema
	}

	return resolved
}

// resolveInlineDeclarations resolves inline struct declarations from function body
func (r *Resolver) resolveInlineDeclarations(endpoint *ResolvedEndpoint, inlines *parser.FuncInlineInfo, parameters map[string]*ResolvedParameter, schemas map[string]*ResolvedSchema, defaultContentType string) error {
	// Resolve inline path parameters
	if inlines.Path != nil {
		params, err := r.resolveInlineParams(inlines.Path, "path")
		if err != nil {
			return fmt.Errorf("failed to resolve inline path params: %w", err)
		}
		endpoint.InlinePathParams = params
	}

	// Resolve inline query parameters
	if inlines.Query != nil {
		params, err := r.resolveInlineParams(inlines.Query, "query")
		if err != nil {
			return fmt.Errorf("failed to resolve inline query params: %w", err)
		}
		endpoint.InlineQueryParams = params
	}

	// Resolve inline header parameters
	if inlines.Header != nil {
		params, err := r.resolveInlineParams(inlines.Header, "header")
		if err != nil {
			return fmt.Errorf("failed to resolve inline header params: %w", err)
		}
		endpoint.InlineHeaderParams = params
	}

	// Resolve inline cookie parameters
	if inlines.Cookie != nil {
		params, err := r.resolveInlineParams(inlines.Cookie, "cookie")
		if err != nil {
			return fmt.Errorf("failed to resolve inline cookie params: %w", err)
		}
		endpoint.InlineCookieParams = params
	}

	// Resolve inline request body
	if inlines.Request != nil {
		// Step 1: Parse inline annotation using InlineAnnotationSchema
		parsed, err := ParseInlineAnnotation(inlines.Request.Comment, "request")
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
		parsed, err := ParseInlineAnnotation(respInfo.Comment, "response")
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

// resolveInlineParams resolves an inline parameter struct
func (r *Resolver) resolveInlineParams(info *parser.InlineStructInfo, paramType string) (*ResolvedInlineParams, error) {
	if info == nil || info.StructType == nil {
		return nil, nil
	}

	// Parameters don't support anonymous struct fields, so pass nil for schemaNames
	fields, err := r.resolveInlineStructFields(info.StructType, info.FieldComments, paramType, nil)
	if err != nil {
		return nil, err
	}

	return &ResolvedInlineParams{
		Fields: fields,
	}, nil
}

// resolveInlineBody resolves an inline request/response body struct using parsed annotation
func (r *Resolver) resolveInlineBody(info *parser.InlineStructInfo, parsed *parser.ParsedAnnotation, parameters map[string]*ResolvedParameter, schemas map[string]*ResolvedSchema, defaultContentType string) (*ResolvedInlineBody, error) {
	if info == nil || info.StructType == nil {
		return nil, nil
	}

	// Build schemaNames for anonymous struct resolution
	schemaNames := make(map[string]bool)
	for name := range schemas {
		schemaNames[name] = true
	}

	// Resolve AST struct fields (inline resolver's job)
	fields, err := r.resolveInlineStructFields(info.StructType, info.FieldComments, "json", schemaNames)
	if err != nil {
		return nil, err
	}

	resolved := &ResolvedInlineBody{
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

// resolveInlineStructFields resolves fields from an AST struct type
// schemaNames is optional - if provided, anonymous struct resolution is enabled
func (r *Resolver) resolveInlineStructFields(structType *ast.StructType, fieldComments map[string]*parser.CommentBlock, tagType string, schemaNames map[string]bool) ([]*ResolvedField, error) {
	if structType == nil || structType.Fields == nil {
		return nil, nil
	}

	fields := make([]*ResolvedField, 0, len(structType.Fields.List))

	for _, astField := range structType.Fields.List {
		if len(astField.Names) == 0 {
			continue // Skip embedded fields
		}

		fieldName := astField.Names[0].Name

		// Get struct tag
		var tag string
		if astField.Tag != nil {
			// Remove quotes from tag literal
			tag = astField.Tag.Value
			if len(tag) >= 2 && tag[0] == '`' && tag[len(tag)-1] == '`' {
				tag = tag[1 : len(tag)-1]
			}
		}

		// Resolve field type using TypesInfo
		var fieldType types.Type
		if r.pkg != nil && r.pkg.TypesInfo != nil {
			if typeAndValue, ok := r.pkg.TypesInfo.Types[astField.Type]; ok {
				fieldType = typeAndValue.Type
			}
		}

		// Extract name from appropriate struct tag
		var resolvedName string
		var shouldSkip bool
		switch tagType {
		case "path":
			resolvedName = extractTagName(tag, "path")
		case "query":
			resolvedName = extractTagName(tag, "query")
		case "header":
			resolvedName = extractTagName(tag, "header")
		case "cookie":
			resolvedName = extractTagName(tag, "cookie")
		default:
			// For schemas, use tag fallback chain (json -> xml -> Go field name)
			resolvedName = resolveFieldNameFromTag(tag, fieldName)
			if resolvedName == "" {
				// Field should be skipped (e.g., json:"-")
				shouldSkip = true
			}
		}

		// Skip fields that should be omitted
		if shouldSkip {
			continue
		}

		// For parameter types, handle "-" and empty as fallback to Go field name
		if resolvedName == "" || resolvedName == "-" {
			resolvedName = fieldName
		}

		resolved := &ResolvedField{
			GoName:   fieldName,
			Name:     resolvedName,
			Required: !strings.Contains(tag, "omitempty"),
		}

		// Resolve type info
		if fieldType != nil {
			resolved.GoType = fieldType.String()

			// Check for anonymous structs (only when schemaNames is provided)
			if schemaNames != nil {
				if inlineFields := r.resolveAnonymousStruct(fieldType, schemaNames); inlineFields != nil {
					resolved.InlineFields = inlineFields
					resolved.OpenAPIType = "object"
				} else if itemFields := r.resolveSliceOfAnonymousStruct(fieldType, schemaNames); itemFields != nil {
					// Slice of anonymous struct
					resolved.IsArray = true
					resolved.OpenAPIType = "array"
					resolved.ItemsType = "object"
					resolved.ItemsInlineFields = itemFields
				} else if mapValueFields := r.resolveMapOfAnonymousStruct(fieldType, schemaNames); mapValueFields != nil {
					// Map with anonymous struct values
					resolved.IsMap = true
					resolved.OpenAPIType = "object"
					resolved.MapValueInlineFields = mapValueFields
				}
			}

			// If not an anonymous struct, resolve type normally
			if resolved.OpenAPIType == "" {
				typeInfo := r.resolveType(fieldType)
				resolved.OpenAPIType = typeInfo.OpenAPIType
				resolved.Format = typeInfo.Format
				resolved.IsArray = typeInfo.IsArray
				resolved.ItemsType = typeInfo.ItemsType
				resolved.Nullable = typeInfo.IsNullable
				resolved.IsAnyValue = typeInfo.IsAnyValue
			}
		} else {
			// Fallback to string if type resolution fails
			resolved.GoType = "string"
			resolved.OpenAPIType = "string"
		}

		// Apply field annotations if present
		if comment := fieldComments[fieldName]; comment != nil {
			r.applyFieldAnnotations(resolved, comment)
		}

		fields = append(fields, resolved)
	}

	return fields, nil
}

// applyFieldAnnotations applies @field annotations to a resolved field
func (r *Resolver) applyFieldAnnotations(field *ResolvedField, comment *parser.CommentBlock) {
	if comment == nil {
		return
	}

	// Parse inline @field annotation
	for _, line := range comment.Lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "@field") {
			continue
		}

		// Extract content between braces
		start := strings.Index(line, "{")
		end := strings.LastIndex(line, "}")
		if start == -1 || end == -1 || end <= start {
			continue
		}
		content := line[start+1 : end]

		// Parse individual annotations
		parts := strings.Split(content, "@")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Split into annotation name and value
			spaceIdx := strings.Index(part, " ")
			var annotName, annotValue string
			if spaceIdx > 0 {
				annotName = "@" + part[:spaceIdx]
				annotValue = strings.TrimSpace(part[spaceIdx+1:])
			} else {
				annotName = "@" + part
			}

			switch annotName {
			case "@description":
				field.Description = annotValue
			case "@format":
				field.Format = annotValue
			case "@example":
				field.Example = annotValue
			case "@default":
				field.Default = annotValue
			case "@pattern":
				field.Pattern = annotValue
			case "@enum":
				field.Enum = strings.Split(annotValue, ",")
				for i := range field.Enum {
					field.Enum[i] = strings.TrimSpace(field.Enum[i])
				}
			case "@deprecated":
				field.Deprecated = true
			case "@minimum":
				if val, err := parseFloat(annotValue); err == nil {
					field.Minimum = &val
				}
			case "@maximum":
				if val, err := parseFloat(annotValue); err == nil {
					field.Maximum = &val
				}
			case "@minLength":
				if val, err := parseInt(annotValue); err == nil {
					field.MinLength = &val
				}
			case "@maxLength":
				if val, err := parseInt(annotValue); err == nil {
					field.MaxLength = &val
				}
			case "@minItems":
				if val, err := parseInt(annotValue); err == nil {
					field.MinItems = &val
				}
			case "@maxItems":
				if val, err := parseInt(annotValue); err == nil {
					field.MaxItems = &val
				}
			case "@uniqueItems":
				field.UniqueItems = true
			}
		}
	}
}

// parseFloat parses a string to float64
func parseFloat(s string) (float64, error) {
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	return val, err
}

// parseInt parses a string to int
func parseInt(s string) (int, error) {
	var val int
	_, err := fmt.Sscanf(s, "%d", &val)
	return val, err
}
