package parser

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Package is everything the parser found in one Go package. It is the input to
// the resolver: annotation data plus the loaded Go package the resolver needs
// to walk actual Go types.
//
// Schemas, Parameters and Endpoints are in source order. Emission order is the
// resolver's business.
type Package struct {
	// Name is the Go package name.
	Name string

	// GoPkg is the loaded package. It is loaded once, here, and handed over
	// so downstream stages never load it again.
	GoPkg *packages.Package

	API        *APIInfo
	Schemas    []*Schema
	Parameters []*ParameterStruct
	Endpoints  []*Endpoint
}

// APIInfo is the @api annotation: the document-level metadata.
type APIInfo struct {
	// Title and Version are required.
	Title   string
	Version string

	Description    string
	TermsOfService string
	Contact        *Contact
	License        *License
	Servers        []*Server

	// SecuritySchemes are in declaration order. The generator sorts them at
	// emission time; the parser does not.
	SecuritySchemes []*SecurityScheme

	// Security holds the default security requirements. Each @security block
	// is one AND-group of requirements; the outer slice is the OR-list.
	Security [][]*SecurityRequirement

	Tags []*Tag

	// DefaultContentType is already MIME-expanded (see ExpandContentType).
	DefaultContentType string
}

// Contact is the @contact block.
type Contact struct {
	Name  string
	Email string
	URL   string
}

// License is the @license block.
type License struct {
	Name string
	URL  string
}

// Server is one @server block.
type Server struct {
	URL         string
	Description string
}

// SecurityScheme is one @securityScheme block.
type SecurityScheme struct {
	Name          string // scheme name, from the @securityScheme metadata
	Type          string // http, apiKey, oauth2, openIdConnect
	Scheme        string // for http: bearer, basic
	BearerFormat  string // for bearer: JWT, etc.
	In            string // for apiKey: header, query, cookie
	ParameterName string // for apiKey: the parameter name
	Description   string
}

// SecurityRequirement is one @with entry inside a @security block.
type SecurityRequirement struct {
	SchemeName string
	Scopes     []string
}

// Tag is one @tag block.
type Tag struct {
	Name        string
	Description string
}

// Schema is a @schema annotated struct, or a type alias that instantiates a
// generic @schema struct.
type Schema struct {
	Name        string
	Description string

	// Pos is the position of the type declaration.
	Pos token.Position

	Deprecated bool

	// Fields holds the parsed @field annotations in declaration order.
	// Fields without a @field annotation are absent: the resolver walks the
	// Go struct for the complete field list and matches these by GoName.
	Fields []*Field

	// IsGeneric marks a generic struct (has type parameters). Generic structs
	// are templates and are not emitted to components.
	IsGeneric bool

	// IsTypeAlias marks a type alias (type X = Y[Z]). Aliases that
	// instantiate a generic @schema are emitted to components.
	IsTypeAlias bool

	// AliasOf is the aliased type as written, e.g. "Response[User]".
	AliasOf string

	// TypeArgs are the type arguments of AliasOf, e.g. ["User"] for
	// "Response[User]" and ["K", "V"] for "Pair[K, V]".
	TypeArgs []string
}

// Parameter kinds, matching the OpenAPI "in" values.
const (
	ParamPath   = "path"
	ParamQuery  = "query"
	ParamHeader = "header"
	ParamCookie = "cookie"
)

// ParameterStruct is a struct marked @path, @query, @header or @cookie.
type ParameterStruct struct {
	Name string

	// Kind is one of the Param* constants.
	Kind string

	// Pos is the position of the type declaration.
	Pos token.Position

	// Fields holds the parsed @field annotations, same contract as
	// Schema.Fields.
	Fields []*Field
}

// Field is one parsed @field annotation, bound to a Go struct field by GoName.
type Field struct {
	GoName string

	// Pos is the position of the field declaration.
	Pos token.Position

	Description string
	Format      string

	// Required and Nullable are tri-state overrides: nil means "no override",
	// and the resolver derives the value from the Go type and struct tags.
	Required *bool
	Nullable *bool

	// Fields holds the @field annotations of the anonymous struct this field
	// spells out, same contract as Schema.Fields. A field that carries no
	// annotation of its own still appears when its anonymous struct has
	// annotated fields, so the nested annotations have a carrier.
	Fields []*Field

	Constraints
}

// Constraints are the pass-through @field values: they flow from annotation to
// output untransformed.
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

// Endpoint is an @endpoint annotated function.
type Endpoint struct {
	FuncName string

	// Pos is the position of the function declaration.
	Pos token.Position

	Method      string
	Path        string
	OperationID string
	Summary     string
	Description string

	// Auth names a security scheme, overriding the API default.
	Auth string

	Tags       []string
	Deprecated bool

	// PathParams and friends name parameter structs, in annotation order.
	PathParams   []string
	QueryParams  []string
	HeaderParams []string
	CookieParams []string

	Request *RequestBody

	// Responses are in declaration order. A repeated status code is
	// last-wins: the second @response replaces the first in place.
	Responses []*Response

	// Inline holds the declarations found in the function body, nil when
	// there are none.
	Inline *EndpointInline
}

// RequestBody is the @request block of an endpoint.
type RequestBody struct {
	// ContentType is already MIME-expanded.
	ContentType string

	// Body is nil when the block has no @body.
	Body *Body
}

// Response is one @response block of an endpoint.
type Response struct {
	// Status is the literal status text: "200", "4XX" or "default".
	Status      string
	Description string

	// Pos is the position of the @response line.
	Pos token.Position

	// ContentType is already MIME-expanded.
	ContentType string

	// Body is nil for bodyless responses (204 and friends).
	Body *Body

	// Headers name @header parameter structs, in annotation order.
	Headers []string
}

// Body is a @body annotation with its optional @bind.
type Body struct {
	// Schema is the referenced type as written: "User", "[]User",
	// "map[string]User".
	Schema string

	// Bind wraps the body inside an envelope schema.
	Bind *BindTarget
}

// BindTarget is a @bind Wrapper.Field value.
type BindTarget struct {
	Wrapper string
	Field   string
}

// EndpointInline holds the declarations annotated inside a handler body.
type EndpointInline struct {
	Path   []*InlineStruct
	Query  []*InlineStruct
	Header []*InlineStruct
	Cookie []*InlineStruct

	// Request is at most one per handler; a duplicate is a parse error.
	Request *InlineStruct

	// Responses are in declaration order; a duplicate status code within one
	// handler is a parse error.
	Responses []*InlineResponse
}

// InlineStruct is one annotated var or type declaration inside a handler body.
type InlineStruct struct {
	VarName string

	// Pos is the position of the declared identifier.
	Pos token.Position

	// Struct is the Go type of the declaration, resolved at parse time. It is
	// nil for a standalone response, which declares nothing.
	Struct *types.Struct

	// Fields holds the parsed @field annotations of the struct's own fields,
	// in declaration order. Same contract as Schema.Fields.
	Fields []*Field

	// ContentType is already MIME-expanded. Empty for parameter structs,
	// which have no annotation body.
	ContentType string

	Description string
	Bind        *BindTarget

	// Headers name @header parameter structs. Responses only.
	Headers []string
}

// InlineResponse is a response declared inside a handler body, in one of two
// shapes: an annotated struct declaration, where the struct is the body, or a
// standalone @response, which names its body with @body because no declaration
// follows it.
type InlineResponse struct {
	// Status is the literal status text, defaulting to "200" when the
	// annotation carries none.
	Status string

	// Body is set for a standalone response, and nil when the declared struct
	// is the body. Exactly one of Body and InlineStruct.Struct is set.
	Body *Body

	InlineStruct
}
