package resolver

// ResolvedPackage contains the fully resolved parsed package with type information
type ResolvedPackage struct {
	// Original parsed package
	PackageName string
	API         *ResolvedAPI
	Schemas     map[string]*ResolvedSchema
	Parameters  map[string]*ResolvedParameter
	Endpoints   []*ResolvedEndpoint
}

// ResolvedAPI contains resolved API info
type ResolvedAPI struct {
	Title              string
	Version            string
	Description        string
	TermsOfService     string
	Contact            *Contact
	License            *License
	Servers            []*Server
	SecuritySchemes    map[string]*SecurityScheme
	Security           [][]*SecurityRequirement
	Tags               []*Tag
	DefaultContentType string
}

// Contact information
type Contact struct {
	Name  string
	Email string
	URL   string
}

// License information
type License struct {
	Name string
	URL  string
}

// Server information
type Server struct {
	URL         string
	Description string
}

// SecurityScheme defines a security scheme
type SecurityScheme struct {
	Name          string
	Type          string
	Scheme        string
	BearerFormat  string
	In            string
	ParameterName string
	Description   string
}

// SecurityRequirement defines a security requirement
type SecurityRequirement struct {
	SchemeName string
	Scopes     []string
}

// Tag represents an API tag definition
type Tag struct {
	Name        string
	Description string
}

// ResolvedSchema contains a schema with resolved type information
type ResolvedSchema struct {
	Name        string
	GoTypeName  string
	Description string
	Deprecated  bool
	Fields      []*ResolvedField

	// IsGeneric indicates this is a generic struct (has type parameters)
	// Generic structs are templates and should not be emitted to components
	IsGeneric bool

	// IsTypeAlias indicates this is a type alias (e.g., type X = Y[Z])
	// Type aliases that instantiate generics should be emitted to components
	IsTypeAlias bool

	// AliasOf is the type this aliases (for type aliases)
	// e.g., "DataResponse[User]"
	AliasOf string

	// TypeArg is the resolved type argument for generic instantiations
	// e.g., for "DataResponse[User]", this would be "User"
	TypeArg string
}

// ResolvedParameter contains a parameter struct with resolved type information
type ResolvedParameter struct {
	Name       string
	Type       string // "path", "query", "header", "cookie"
	GoTypeName string
	Fields     []*ResolvedField
}

// ResolvedField contains a field with resolved type information
type ResolvedField struct {
	// From annotation or inferred
	Name        string
	GoName      string
	Description string
	Required    bool
	Nullable    bool
	Deprecated  bool

	// Type information (resolved from Go type)
	GoType      string // Original Go type string
	OpenAPIType string // "string", "integer", "number", "boolean", "array", "object"
	Format      string // "int32", "int64", "float", "double", "byte", "binary", "date", "date-time", "password", "email", "uuid", etc.

	// Array information
	IsArray           bool
	ItemsType         string           // For arrays, the OpenAPI type of items
	ItemsInlineFields []*ResolvedField // For arrays of anonymous structs, the resolved fields

	// Map information
	IsMap                bool
	MapValueInlineFields []*ResolvedField // For maps of anonymous structs, the resolved value fields

	// Any value type (any/interface{})
	IsAnyValue bool // True if field accepts any JSON value

	// Struct type information
	IsUnresolvedStruct bool   // True if field references a struct that is not a @schema
	UnresolvedTypeName string // The name of the unresolved struct type (for error messages)

	// Anonymous struct support
	InlineFields []*ResolvedField // For anonymous structs, the resolved fields to inline

	// Validation constraints
	Enum        []string
	Default     string
	Example     string
	Pattern     string
	MinLength   *int
	MaxLength   *int
	MinItems    *int
	MaxItems    *int
	UniqueItems bool
	Minimum     *float64
	Maximum     *float64
}

// ResolvedEndpoint contains an endpoint with resolved types
type ResolvedEndpoint struct {
	FuncName     string
	Method       string
	Path         string
	OperationID  string
	Summary      string
	Description  string
	Tags         []string
	Deprecated   bool
	Auth         string
	Request      *ResolvedRequestBody
	Responses    map[string]*ResolvedResponse
	PathParams   []*ResolvedParameter
	QueryParams  []*ResolvedParameter
	HeaderParams []*ResolvedParameter
	CookieParams []*ResolvedParameter

	// Inline declarations (resolved from function body annotations)
	InlinePathParams   *ResolvedInlineParams
	InlineQueryParams  *ResolvedInlineParams
	InlineHeaderParams *ResolvedInlineParams
	InlineCookieParams *ResolvedInlineParams
	InlineRequest      *ResolvedInlineBody
	InlineResponses    map[string]*ResolvedInlineBody // Key is status code
}

// ResolvedInlineParams contains resolved inline parameter fields
type ResolvedInlineParams struct {
	Fields []*ResolvedField
}

// ResolvedInlineBody contains resolved inline body fields with optional binding
type ResolvedInlineBody struct {
	ContentType string
	Fields      []*ResolvedField
	Bind        *ResolvedBindTarget
	Headers     []*ResolvedParameter // Response headers (for inline responses)
	Description string               // Response description
}

// ResolvedRequestBody contains a request body with resolved schema
type ResolvedRequestBody struct {
	ContentType string
	Body        *ResolvedBody
	Required    bool
}

// ResolvedResponse contains a response with resolved schema
type ResolvedResponse struct {
	StatusCode  string
	Description string
	ContentType string
	Body        *ResolvedBody
	Headers     []*ResolvedParameter
}

// ResolvedBody contains the resolved body with optional binding
type ResolvedBody struct {
	// Schema is the schema name being referenced (e.g., "User", "[]User", "map[string]User")
	Schema string

	// Bind contains the resolved wrapper binding
	// When non-nil, the body should be wrapped in the specified envelope
	Bind *ResolvedBindTarget

	// IsArray indicates Schema is an array type (e.g., []User)
	IsArray bool

	// IsMap indicates Schema is a map type (e.g., map[string]User)
	IsMap bool

	// ElementType is the element type for arrays/maps (e.g., "User")
	ElementType string
}

// ResolvedBindTarget represents a resolved @bind Wrapper.Field annotation
type ResolvedBindTarget struct {
	// Wrapper is the wrapper schema name (e.g., "DataResponse")
	Wrapper string

	// Field is the field to bind the body to (e.g., "Data")
	Field string

	// WrapperSchema is the resolved wrapper schema (for inlining)
	WrapperSchema *ResolvedSchema
}
