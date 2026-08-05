package resolver

// This file defines the emission-ready IR the generator consumes. It replaces
// the ResolvedPackage family in types.go; during the rewrite transition both
// exist, with ConvertLegacy (compat.go) bridging old to new.
//
// The IR is deliberately pre-ordered: Endpoint.Parameters and
// Endpoint.Responses are built in final emission order so the generator
// iterates without branching on named-vs-inline origin or re-sorting.

// Package is the fully resolved, emission-ordered package.
type Package struct {
	Name string
	API  *ResolvedAPI

	// Schemas holds named @schema types sorted by name, including generic
	// templates (flagged IsGeneric) which the generator skips at emission.
	Schemas []*Schema

	Endpoints []*Endpoint
}

// Schema is a named object schema destined for components.
type Schema struct {
	Name        string
	Description string
	Deprecated  bool
	Fields      []*Field

	// IsGeneric marks generic templates (type parameters present). They are
	// kept in the list so "were there any schemas at all" checks match the
	// legacy behavior, but they are never emitted to components.
	IsGeneric bool
}

// Field is a fully resolved struct field.
type Field struct {
	Name        string // wire name (from json/xml tag or Go name)
	GoName      string
	Description string
	Required    bool
	Nullable    bool
	Format      string
	Type        TypeInfo
	Constraints

	// Anonymous struct support: when non-empty these take precedence over
	// Type, in this order (matching legacy emission).
	Inline         []*Field // field is an anonymous struct
	ItemsInline    []*Field // field is a slice of anonymous structs
	MapValueInline []*Field // field is a map with anonymous struct values
}

// Constraints are the pass-through @field annotation values. They flow from
// annotation to output without transformation.
type Constraints struct {
	Enum             []string
	Default          string
	Example          string
	Pattern          string
	MinLength        *int
	MaxLength        *int
	MinItems         *int
	MaxItems         *int
	UniqueItems      bool
	Minimum          *float64
	Maximum          *float64
	ExclusiveMinimum *float64
	ExclusiveMaximum *float64
	Deprecated       bool
	ReadOnly         bool
	WriteOnly        bool
}

// TypeInfo describes the resolved OpenAPI shape of a field's Go type. The
// generator never parses Go type strings; everything it needs is here.
type TypeInfo struct {
	// OpenAPI is the scalar type ("string", "integer", "number", "boolean",
	// "object") when the field is not an array, map, ref, or any.
	OpenAPI string

	// IsAny marks any/interface{} fields (emitted as an empty schema).
	IsAny bool

	IsArray bool
	// Items is the OpenAPI type of array items when they are not a schema
	// ref. Kept even when ItemsRef is set because parameter schemas ignore
	// refs and emit this instead (legacy behavior).
	Items    string
	ItemsRef string // schema name when items reference a named schema

	IsMap bool
	// MapValue mirrors Items for map values, including the legacy "string"
	// fallback for unresolvable value types.
	MapValue    string
	MapValueRef string

	// Ref is the schema name when the field itself is a named schema.
	Ref string
}

// Endpoint is an operation with pre-ordered parameters and responses.
type Endpoint struct {
	Method      string
	Path        string
	OperationID string
	Summary     string
	Description string
	Auth        string
	Tags        []string
	Deprecated  bool

	// Parameters in final emission order: named path, query, header, cookie
	// groups first, then inline path, query, header, cookie fields.
	Parameters []*Param

	Request *Request

	// Responses in final emission order: named statuses sorted, then inline
	// statuses sorted with named-wins conflict resolution already applied.
	Responses []*Response
}

// Param is a single operation parameter.
type Param struct {
	In    string // "path", "query", "header", "cookie"
	Field *Field
}

// Request is a request body. A nil Content renders as an empty requestBody
// (the legacy behavior for @request blocks without a @body).
type Request struct {
	ContentType string
	Required    bool
	Content     *Content
}

// Response is a single response. A nil Content renders with no content block
// (204-style responses and bodyless declarations).
type Response struct {
	Status      string
	Description string
	ContentType string
	Headers     []*Field
	Content     *Content
}

// Content is a body: either a reference to named schema types (Ref) or
// resolved inline struct fields (Fields), optionally wrapped via Bind.
type Content struct {
	Ref    *TypeRef
	Fields []*Field
	Bind   *BindTarget
}

// TypeRef references named schema types or primitives for @body values like
// "User", "[]User", or "map[string]int". Exactly one of Schema/Primitive is
// set: Schema names a components entry, Primitive is a ready OpenAPI type.
type TypeRef struct {
	IsArray   bool
	IsMap     bool
	Schema    string
	Primitive string
}

// BindTarget wraps a body inside a named envelope schema, replacing the
// wrapper field whose Go name is Field. A nil Wrapper falls back to plain
// body emission (legacy behavior when the wrapper schema is unknown).
type BindTarget struct {
	Field   string
	Wrapper *Schema
}
