package resolver

import (
	"fmt"
	"go/types"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/parser"
	"github.com/wontaeyang/go-specgen/pkg/specerr"
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

	// errs accumulates everything that failed to resolve.
	errs specerr.List

	// path names the declaration being resolved right now, so a field error
	// raised several calls deep can say which schema or endpoint it belongs to.
	// Held here for the same reason schemaNames is: threading it would touch
	// every field-resolving function and the embedded-flattening recursion
	// between them, to carry a value that is constant for one declaration.
	// Resolve sets it, in the three places it moves to a new declaration.
	path string
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
		errs:        specerr.List{Stage: "resolve"},
	}
}

// Resolve resolves all types in the parsed package.
//
// Like the parser, it accumulates and keeps going, down to the individual
// field: a struct with four unresolvable fields reports four. The maps are
// walked in name order because otherwise which failure appeared -- back when the
// first one returned -- was down to Go's map iteration.
//
// Nothing here reports on an earlier declaration's absence: a schema that failed
// to resolve is simply missing from the map, and both later loops treat a
// missing entry as nothing to do. Saying "unknown schema" about it is the
// validator's job, and the validator never runs on a package that failed here.
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
	for _, name := range slices.Sorted(maps.Keys(parsed.Schemas)) {
		r.path = fmt.Sprintf("@schema[%s]", name)

		resolvedSchema, err := r.resolveSchema(parsed.Schemas[name])
		if err != nil {
			r.errs.Wrap(r.path, err)
			continue
		}
		resolved.Schemas[name] = resolvedSchema
	}

	// Resolve parameters
	for _, name := range slices.Sorted(maps.Keys(parsed.Parameters)) {
		param := parsed.Parameters[name]
		r.path = fmt.Sprintf("@%s[%s]", param.Type, name)

		resolvedParam, err := r.resolveParameter(param)
		if err != nil {
			r.errs.Wrap(r.path, err)
			continue
		}
		resolved.Parameters[name] = resolvedParam
	}

	// Resolve endpoints
	defaultContentType := ""
	if resolved.API != nil {
		defaultContentType = resolved.API.DefaultContentType
	}
	// Endpoints are already in parser order, which is sorted by function name.
	for _, endpoint := range parsed.Endpoints {
		r.path = fmt.Sprintf("@endpoint[%s %s]", endpoint.Method, endpoint.Path)

		resolvedEndpoint, err := r.resolveEndpoint(endpoint, resolved.Parameters, resolved.Schemas, defaultContentType)
		if err != nil {
			r.errs.Wrap(r.path, err)
			continue
		}
		resolved.Endpoints = append(resolved.Endpoints, resolvedEndpoint)
	}

	if err := r.errs.Err(); err != nil {
		return nil, err
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
	resolved.Fields = r.resolveSchemaFields(structType, schema.Fields, nil)

	return resolved, nil
}

// resolveSchemaFields resolves fields from a struct type, flattening embedded structs.
// visited tracks type names to prevent infinite recursion from circular embedding.
//
// There is no error to return. A type with no OpenAPI representation still
// resolves, to ShapeUnsupported carrying the reason, and the validator reports
// it from there. Its parameter-struct twin returns none either, for a different
// reason: its field errors are real, but they accumulate into r.errs and the
// loop continues, so neither ever handed one back. Both used to, and every
// caller carried an "if err != nil" that could not fire.
func (r *Resolver) resolveSchemaFields(structType *types.Struct, annotations []*parser.Field, visited map[string]bool) []*Field {
	if visited == nil {
		visited = make(map[string]bool)
	}

	var fields []*Field

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		if field.Anonymous() && flattensEmbedded(field, tag, SupportedTags...) {
			fields = append(fields, r.flattenEmbeddedField(field, annotations, visited)...)
			continue
		}

		resolvedField := r.resolveField(field, tag, findAnnotation(annotations, field.Name()))

		// Skip fields that should be omitted (e.g., json:"-")
		if resolvedField == nil {
			continue
		}

		fields = append(fields, resolvedField)
	}

	return fields
}

// flattensEmbedded reports whether an embedded field's own fields should be
// lifted into the enclosing struct.
//
// encoding/json flattens an embedded struct only when it is untagged. Every
// other case is an ordinary field, and falls through to the normal field path:
//
//	Base              untagged struct      flatten
//	Meta `json:"meta"`  tagged struct      a field named meta
//	Hidden `json:"-"`   tagged "-"         skipped
//	Label             untagged non-struct  a field named Label
//	Slug `json:"slug"`  tagged non-struct   a field named slug
//
// The last two need no special handling: an embedded field's Go name is its
// type name, so the ordinary naming path already produces the right key. Which
// is why this asks one question rather than enumerating five cases.
//
// tagKeys is what makes the rule work for both callers: json (and xml) for
// schema fields, the parameter kind for parameter fields.
func flattensEmbedded(field *types.Var, tag string, tagKeys ...string) bool {
	structTag := reflect.StructTag(tag)
	for _, key := range tagKeys {
		if _, tagged := structTag.Lookup(key); tagged {
			return false
		}
	}

	t := field.Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// time.Time and friends are structs that serialize as scalars, so
	// flattening them would spill their internals into the enclosing object.
	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		if obj.Pkg() != nil && isSpecialType(obj.Pkg().Path(), obj.Name()) {
			return false
		}
	}

	_, isStruct := t.Underlying().(*types.Struct)
	return isStruct
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
func (r *Resolver) flattenEmbeddedField(field *types.Var, annotations []*parser.Field, visited map[string]bool) []*Field {
	embeddedStruct, cleanup := unwrapEmbeddedStruct(field.Type(), visited)
	defer cleanup()
	if embeddedStruct == nil {
		return nil
	}

	return r.resolveSchemaFields(embeddedStruct, r.embeddedAnnotations(field, annotations), visited)
}

// embeddedAnnotations returns the @field annotations to apply to an embedded
// struct's fields.
//
// They belong to the embedded type, not to the struct doing the embedding: a
// field of Base is declared in Base, and that is the only place its @field can
// be written. The enclosing struct's annotations are the fallback for an
// anonymous embedded struct, which has no type name to look up.
func (r *Resolver) embeddedAnnotations(field *types.Var, enclosing []*parser.Field) []*parser.Field {
	t := field.Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	if named, ok := t.(*types.Named); ok {
		if own, found := r.parsed.StructFields[named.Obj().Name()]; found {
			return own
		}
	}

	return enclosing
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
	resolved.Fields = r.resolveParameterFields(structType, param.Fields, string(param.Type), nil)

	return resolved, nil
}

// resolveParameterFields resolves fields from a parameter struct, flattening embedded structs.
func (r *Resolver) resolveParameterFields(structType *types.Struct, annotations []*parser.Field, paramType string, visited map[string]bool) []*Field {
	if visited == nil {
		visited = make(map[string]bool)
	}

	var fields []*Field

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		if field.Anonymous() && flattensEmbedded(field, tag, paramType) {
			fields = append(fields, r.flattenEmbeddedParamField(field, annotations, paramType, visited)...)
			continue
		}

		fieldAnnotation := findAnnotation(annotations, field.Name())

		// Recorded and skipped rather than ending the struct: mis-tagging one
		// field of a parameter struct usually means mis-tagging several, and
		// reporting them one run at a time is the case accumulation is for.
		// Unlike resolveSchemaFields, this one really can fail — a field tagged
		// for one parameter kind inside a struct declared as another — but the
		// failure lands in r.errs rather than coming back up, which is why this
		// function returns no error of its own.
		resolvedField, err := r.resolveFieldWithParamType(field, tag, fieldAnnotation, paramType)
		if err != nil {
			r.errs.Addf(r.path+"."+field.Name(), "%s", err)
			continue
		}
		// Skip fields that should be omitted
		if resolvedField == nil {
			continue
		}

		fields = append(fields, resolvedField)
	}

	return fields
}

// flattenEmbeddedParamField resolves an embedded struct field for parameters.
func (r *Resolver) flattenEmbeddedParamField(field *types.Var, annotations []*parser.Field, paramType string, visited map[string]bool) []*Field {
	embeddedStruct, cleanup := unwrapEmbeddedStruct(field.Type(), visited)
	defer cleanup()
	if embeddedStruct == nil {
		return nil
	}

	return r.resolveParameterFields(embeddedStruct, r.embeddedAnnotations(field, annotations), paramType, visited)
}

// resolveField resolves a single struct field.
//
// Returns nil when the field should not appear at all: unexported, or tagged
// json:"-".
//
// There is no error to return. Every Go type resolves to some shape, and the
// one that has no OpenAPI representation resolves to ShapeUnsupported carrying
// the reason — which the validator reports, where a reader sees the type named
// alongside every other problem in the package. Its sibling
// resolveFieldWithParamType does error, because a field tagged for one
// parameter kind inside a struct declared as another is a contradiction in the
// annotation rather than a fact about the type.
func (r *Resolver) resolveField(field *types.Var, tag string, annotation *parser.Field) *Field {
	// Unexported fields cannot be serialized.
	if !field.Exported() {
		return nil
	}

	fieldName := resolveFieldNameFromTag(tag, field.Name())
	if fieldName == "" {
		return nil
	}

	typeRef := r.resolveTypeRef(field.Type(), annotationFields(annotation))

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

	return resolved
}

// findAnnotation returns the @field annotation written on a Go field, or nil.
func findAnnotation(annotations []*parser.Field, goName string) *parser.Field {
	for _, annotation := range annotations {
		if annotation.GoName == goName {
			return annotation
		}
	}
	return nil
}

// annotationFields returns the nested @field annotations of an annotation, or
// nil when there is no annotation at all.
func annotationFields(annotation *parser.Field) []*parser.Field {
	if annotation == nil {
		return nil
	}
	return annotation.Fields
}

// resolveAnonymousFields resolves the members of an anonymous struct, attaching
// each field's own @field annotation from the enclosing field's nested set.
func (r *Resolver) resolveAnonymousFields(structType *types.Struct, annotations []*parser.Field) []*Field {
	fields := make([]*Field, 0, structType.NumFields())

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		if field.Anonymous() {
			fields = append(fields, r.flattenEmbeddedField(field, nil, nil)...)
			continue
		}

		resolved := r.resolveField(field, tag, findAnnotation(annotations, field.Name()))
		if resolved == nil {
			continue
		}
		fields = append(fields, resolved)
	}

	return fields
}

// omitsWhenEmpty reports whether the json tag actually drops the field for
// empty/zero values of the given type. omitzero omits any zero value,
// including zero structs. omitempty follows encoding/json's isEmptyValue,
// which never considers structs empty (and arrays only at length zero), so
// such fields always appear on the wire despite the tag.
//
// The options are read from a serialization tag, not from the raw tag text. A
// field is routinely tagged for other packages too, and `validate:"omitempty"`
// is go-playground/validator's spelling of a validation rule rather than a
// serialization one. Scanning the whole tag let any of those silently mark the
// field optional -- the same mistake ",required" used to make in
// resolveFieldWithParamType.
//
// Which tag governs follows SupportedTags, in the order resolveFieldNameFromTag
// walks it, so the tag that decides the field's name is the one that decides
// whether it appears at all. The first tag present wins even when it carries no
// options: `json:"a" xml:"b,omitempty"` is a field encoding/json always writes.
//
// Both keywords are honored under every supported tag, including encoding/xml,
// which implements omitempty but not omitzero. Reading it there is deliberate:
// someone who writes `xml:",omitzero"` is stating the field is absent when zero,
// and a schema that agrees stays right if encoding/xml gains the option later,
// where a special case would quietly disagree with the annotation today. Adding
// a tag to SupportedTags then means one edit rather than one edit plus a table
// of which options that encoder happens to implement.
func omitsWhenEmpty(tag string, fieldType types.Type) bool {
	structTag := reflect.StructTag(tag)

	for _, key := range SupportedTags {
		value, tagged := structTag.Lookup(key)
		if !tagged {
			continue
		}

		options := tagOptions(value)
		if slices.Contains(options, "omitzero") {
			return true
		}
		return slices.Contains(options, "omitempty") && canBeEmpty(fieldType)
	}

	return false
}

// tagOptions returns the comma-separated options of a struct tag value --
// everything after the name -- or nil when it carries only a name.
func tagOptions(value string) []string {
	_, options, hasOptions := strings.Cut(value, ",")
	if !hasOptions {
		return nil
	}
	return strings.Split(options, ",")
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

// parameterKinds are the struct tags that can name a parameter.
var parameterKinds = []string{"path", "query", "header", "cookie"}

// resolveFieldWithParamType resolves one field of a parameter struct.
//
// Naming mirrors encoding/json, so there is one naming rule across the whole
// codebase rather than one for bodies and another for parameters:
//
//	query:"limit"    the parameter is named limit
//	query:"-"        the field is skipped
//	query:"-,"       the parameter is literally named "-", as in encoding/json
//	(no query tag)   the parameter is named by the Go field
//
// A field tagged for a different kind is the exception, and an error. A @query
// struct describes query parameters; a field carrying only path:"tenant_id" is
// either a mistake or a struct doing double duty, and either way the parameter
// specgen emits is not the one the handler reads.
func (r *Resolver) resolveFieldWithParamType(field *types.Var, tag string, annotation *parser.Field, paramType string) (*Field, error) {
	// Unexported fields cannot be bound from a request.
	if !field.Exported() {
		return nil, nil
	}

	structTag := reflect.StructTag(tag)
	value, tagged := structTag.Lookup(paramType)

	name, options, hasOptions := strings.Cut(value, ",")

	switch {
	case !tagged:
		if other := otherParameterKind(structTag, paramType); other != "" {
			return nil, fmt.Errorf(
				"field %s is tagged %q but sits in a @%s struct; tag it %q or move it",
				field.Name(), other, paramType, paramType)
		}
		name = field.Name()

	case name == "-" && !hasOptions:
		return nil, nil

	case name == "":
		// query:",required" names nothing, so the Go field name applies.
		name = field.Name()
	}

	resolved := &Field{
		Name:   name,
		GoName: field.Name(),
		GoType: field.Type().String(),
		Type:   r.resolveTypeRef(field.Type()),

		// Path parameters are part of the URL, so they are always required.
		// The rest are optional unless the tag opts in — read from this kind's
		// own options, not from the raw tag, where a ",required" belonging to
		// some other tag used to count.
		Required: paramType == "path" || slices.Contains(strings.Split(options, ","), "required"),

		// Parameters serialize as plain strings, which cannot represent null,
		// so pointer-ness never implies nullable — a pointer only lets the
		// handler distinguish absent from zero. @nullable can still opt in.
		Nullable: false,
	}
	resolved.Format = resolved.Type.Format

	applyAnnotationOverrides(resolved, annotation)

	return resolved, nil
}

// otherParameterKind reports a parameter tag on this field that names a
// different kind than the struct it is in, or "" if there is none.
func otherParameterKind(tag reflect.StructTag, paramType string) string {
	for _, kind := range parameterKinds {
		if kind == paramType {
			continue
		}
		if _, ok := tag.Lookup(kind); ok {
			return kind
		}
	}
	return ""
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
		FuncName:    endpoint.FuncName,
		Method:      endpoint.Method,
		Path:        endpoint.Path,
		OperationID: endpoint.OperationID,
		Summary:     endpoint.Summary,
		Description: endpoint.Description,
		Tags:        endpoint.Tags,
		Deprecated:  endpoint.Deprecated,
		Auth:        endpoint.Auth,
	}

	// Named request body
	if endpoint.Request != nil && endpoint.Request.Body != nil {
		resolved.Request = &RequestBody{
			ContentType: r.contentType(endpoint.Request.ContentType, defaultContentType),
			Body:        r.resolveBody(endpoint.Request.Body, schemas),
			Required:    true, // Default to required
		}
	}

	// Named responses, keyed by status so inline ones can be merged in below.
	responses := make(map[string]*Response, len(endpoint.Responses))
	for statusCode, response := range endpoint.Responses {
		resolvedResponse := &Response{
			StatusCode:  response.StatusCode,
			Description: responseDescription(response.Description, response.StatusCode),
			Body:        r.resolveBody(response.Body, schemas),
		}

		// A response with no body needs no content type, and emitting one
		// would claim content that is never sent (204, say).
		if response.Body != nil {
			resolvedResponse.ContentType = r.contentType(response.ContentType, defaultContentType)
		}

		for _, ref := range response.HeaderParams {
			resolvedResponse.HeaderRefs = append(resolvedResponse.HeaderRefs, ref)
			if param, ok := parameters[ref]; ok {
				resolvedResponse.Headers = append(resolvedResponse.Headers, param)
			}
		}

		responses[statusCode] = resolvedResponse
	}

	// Named parameters, in declaration order within each location.
	named := []struct {
		refs []string
		in   string
	}{
		{endpoint.PathParams, "path"},
		{endpoint.QueryParams, "query"},
		{endpoint.HeaderParams, "header"},
		{endpoint.CookieParams, "cookie"},
	}
	for _, group := range named {
		for _, ref := range group.refs {
			resolved.ParamRefs = append(resolved.ParamRefs, ParamRef{Name: ref, In: group.in})
			if param, ok := parameters[ref]; ok {
				resolved.Parameters = append(resolved.Parameters, parametersFrom(param.Fields, group.in)...)
			}
		}
	}

	// In-function declarations, appended after the named ones of every
	// location. See the ordering note on sortResponses for why responses are
	// merged rather than concatenated.
	if inlines := r.parsed.FuncInlines[endpoint.FuncName]; inlines != nil {
		if err := r.resolveInlineDeclarations(resolved, responses, inlines, parameters, schemas, defaultContentType); err != nil {
			return nil, fmt.Errorf("failed to resolve inline declarations: %w", err)
		}
	}

	resolved.Responses = sortResponses(responses)

	return resolved, nil
}

// contentType applies the fallback chain for a declared content type: the
// annotation's own value, then the API default, then JSON.
func (r *Resolver) contentType(declared, apiDefault string) string {
	if declared != "" {
		return declared
	}
	if apiDefault != "" {
		return apiDefault
	}
	return "application/json"
}

// parametersFrom attaches a location to each of a parameter struct's fields.
func parametersFrom(fields []*Field, in string) []*Parameter {
	params := make([]*Parameter, 0, len(fields))
	for _, field := range fields {
		params = append(params, &Parameter{In: in, Field: field})
	}
	return params
}

// responseDescription is what a response says about itself when its annotation
// did not say.
//
// The Response Object requires a description, so the field is emitted either
// way and the only question is whether it is emitted empty. It used to depend
// on which form declared the response: a named @response rendered
// description: "", an in-function one rendered "Response for status 200".
//
// Both now fall back to the status code's reason phrase — "OK", "Created",
// "No Content" — which is what a hand-written document puts in that slot, and
// which net/http already knows so there is no table here to drift. Wildcard
// ranges and "default" cover no single status and so have no phrase; they
// describe what they cover instead.
func responseDescription(declared, statusCode string) string {
	if declared != "" {
		return declared
	}

	if statusCode == "default" {
		return "Default response"
	}
	if code, err := strconv.Atoi(statusCode); err == nil {
		if phrase := http.StatusText(code); phrase != "" {
			return phrase
		}
	}
	return fmt.Sprintf("Response for status %s", statusCode)
}

// sortResponses puts responses in emission order: by status code as a string.
//
// That is not numeric order, and deliberately so — the set includes wildcard
// ranges and "default", which have no number. String order puts 200 before 4XX
// before 5XX before default, which is the order a reader expects and the one
// examples/responses has always produced.
func sortResponses(byStatus map[string]*Response) []*Response {
	responses := make([]*Response, 0, len(byStatus))
	for _, status := range slices.Sorted(maps.Keys(byStatus)) {
		responses = append(responses, byStatus[status])
	}
	return responses
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
func (r *Resolver) resolveInlineDeclarations(endpoint *Endpoint, responses map[string]*Response, inlines *parser.FuncInlineInfo, parameters map[string]*ParameterStruct, schemas map[string]*Schema, defaultContentType string) error {
	// In-function parameters follow the named ones, grouped by location in the
	// same order, so a handler mixing both styles still emits path parameters
	// before query parameters.
	inlineGroups := []struct {
		infos []*parser.InlineStructInfo
		in    string
	}{
		{inlines.Path, "path"},
		{inlines.Query, "query"},
		{inlines.Header, "header"},
		{inlines.Cookie, "cookie"},
	}
	for _, group := range inlineGroups {
		fields, err := r.mergeInlineParams(group.infos, group.in)
		if err != nil {
			return err
		}
		endpoint.Parameters = append(endpoint.Parameters, parametersFrom(fields, group.in)...)
	}

	if inlines.Request != nil {
		parsed, err := ParseInlineDeclaration(inlines.Request.Comment, "request")
		if err != nil {
			return fmt.Errorf("failed to parse inline request: %w", err)
		}

		body, err := r.resolveInlineBody(inlines.Request, parsed, nil, schemas, defaultContentType)
		if err != nil {
			return fmt.Errorf("failed to resolve inline request body: %w", err)
		}

		// A named @request wins: it names a schema, which is more specific than
		// a struct declared in the handler.
		if endpoint.Request == nil {
			endpoint.Request = &RequestBody{
				ContentType: body.ContentType,
				Inline:      body,
				Required:    true,
			}
		}
	}

	for _, statusCode := range slices.Sorted(maps.Keys(inlines.Responses)) {
		respInfo := inlines.Responses[statusCode]

		parsed, err := ParseInlineDeclaration(respInfo.Comment, "response")
		if err != nil {
			return fmt.Errorf("failed to parse inline response %s: %w", statusCode, err)
		}

		body, err := r.resolveInlineBody(respInfo, parsed, parameters, schemas, defaultContentType)
		if err != nil {
			return fmt.Errorf("failed to resolve inline response %s: %w", statusCode, err)
		}

		// A named @response for the same status wins, for the same reason.
		if _, taken := responses[statusCode]; taken {
			continue
		}

		responses[statusCode] = &Response{
			StatusCode:  statusCode,
			Description: responseDescription(body.Description, statusCode),
			ContentType: body.ContentType,
			Inline:      body,
			Headers:     body.Headers,
			HeaderRefs:  body.HeaderRefs,
		}
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

// mergeInlineParams resolves every inline struct of one location (all the
// inline @query structs in a handler, say) into one flat field list.
//
// OpenAPI emits a single parameters array per operation no matter how many
// structs the handler declared, so the merge belongs here rather than at
// emission time.
func (r *Resolver) mergeInlineParams(infos []*parser.InlineStructInfo, paramType string) ([]*Field, error) {
	var merged []*Field

	for _, info := range infos {
		structType, err := r.inlineStructType(info)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve inline %s params %q: %w", paramType, info.VarName, err)
		}

		merged = append(merged, r.resolveParameterFields(structType, info.Fields, paramType, nil)...)
	}

	return merged, nil
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

	resolved := &InlineBody{
		Fields: r.resolveSchemaFields(structType, info.Fields, nil),
	}

	// Use parsed annotation (already validated by inline parser)
	if parsed != nil {
		// Content type
		if ct := parsed.GetChildValue("@contentType"); ct != "" {
			resolved.ContentType = parser.ExpandContentType(ct)
		}

		// Description
		resolved.Description = parsed.GetChildValue("@description")

		// Bind. Keyed on the annotation being present rather than on its value
		// being non-empty, so a bare @bind is the same error here as a malformed
		// one -- both name no target, and skipping either dropped it in silence.
		if parsed.HasChild("@bind") {
			bindTarget, err := parser.ParseBindTarget(parsed.GetChildValue("@bind"))
			if err != nil {
				return nil, err
			}
			resolved.Bind = r.resolveBindTarget(bindTarget, schemas)
		}

		// Resolve header references (response only)
		if parameters != nil {
			for _, headerChild := range parsed.GetRepeatedChildren("@header") {
				resolved.HeaderRefs = append(resolved.HeaderRefs, headerChild.Value)
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
