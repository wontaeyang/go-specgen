package parser

// ParsedPackage represents a complete parsed Go package with all annotations
type ParsedPackage struct {
	// PackageName is the Go package name
	PackageName string

	// API contains the @api annotation data
	API *APIInfo

	// Schemas contains all @schema annotated structs
	Schemas map[string]*Schema

	// Parameters contains all parameter structs (@path, @query, @header, @cookie)
	Parameters map[string]*Parameter

	// Endpoints contains all @endpoint annotated functions
	Endpoints []*Endpoint
}

// APIInfo represents the @api annotation
type APIInfo struct {
	// Title is the API title (required)
	Title string

	// Version is the API version (required)
	Version string

	// Description is the API description
	Description string

	// TermsOfService is the URL to the terms of service
	TermsOfService string

	// Contact contains contact information
	Contact *Contact

	// License contains license information
	License *License

	// Servers is the list of server configurations
	Servers []*Server

	// SecuritySchemes defines available security schemes
	SecuritySchemes map[string]*SecurityScheme

	// Security defines default security requirements
	Security [][]*SecurityRequirement // Array of arrays for OR/AND logic

	// Tags defines API-level tag definitions
	Tags []*Tag

	// DefaultContentType is the default content type for requests/responses
	DefaultContentType string
}

// Contact represents contact information
type Contact struct {
	Name  string
	Email string
	URL   string
}

// License represents license information
type License struct {
	Name string
	URL  string
}

// Server represents a server configuration
type Server struct {
	URL         string
	Description string
}

// SecurityScheme represents a security scheme definition
type SecurityScheme struct {
	Name          string // Name of the scheme
	Type          string // http, apiKey, oauth2, openIdConnect
	Scheme        string // For http type: bearer, basic
	BearerFormat  string // For bearer scheme: JWT, etc.
	In            string // For apiKey: header, query, cookie
	ParameterName string // For apiKey: parameter name
	Description   string
}

// SecurityRequirement represents a security requirement
type SecurityRequirement struct {
	SchemeName string
	Scopes     []string // For OAuth2
}

// Schema represents a @schema annotated struct
type Schema struct {
	// Name is the struct name
	Name string

	// GoTypeName is the full Go type name (for resolution)
	GoTypeName string

	// Description is the schema description
	Description string

	// Deprecated indicates if the schema is deprecated
	Deprecated bool

	// Fields are the struct fields
	Fields []*Field

	// IsGeneric indicates this is a generic struct (has type parameters)
	// Generic structs are templates and not emitted to components
	IsGeneric bool

	// IsTypeAlias indicates this is a type alias (e.g., type X = Y[Z])
	// Type aliases that instantiate generics are emitted to components
	IsTypeAlias bool

	// AliasOf is the type this aliases (for type aliases)
	AliasOf string
}

// Field represents a struct field with @field annotation
type Field struct {
	// Name is the field name (from json/query/path/header/cookie tag)
	Name string

	// GoName is the Go field name
	GoName string

	// GoType is the Go type name
	GoType string

	// Required indicates if the field is required
	Required bool

	// Nullable indicates if the field is nullable (pointer type)
	Nullable bool

	// Description is the field description
	Description string

	// Format is the field format (email, uuid, date-time, etc.)
	Format string

	// Example is an example value
	Example string

	// Enum is a list of allowed values
	Enum []string

	// Default is the default value
	Default string

	// Minimum is the minimum value (for numbers)
	Minimum *float64

	// Maximum is the maximum value (for numbers)
	Maximum *float64

	// MinLength is the minimum length (for strings)
	MinLength *int

	// MaxLength is the maximum length (for strings)
	MaxLength *int

	// MinItems is the minimum number of items (for arrays)
	MinItems *int

	// MaxItems is the maximum number of items (for arrays)
	MaxItems *int

	// UniqueItems indicates array items must be unique
	UniqueItems bool

	// Pattern is a regex pattern (for strings)
	Pattern string

	// Deprecated indicates if the field is deprecated
	Deprecated bool
}

// Parameter represents a parameter struct (@path, @query, @header, @cookie)
type Parameter struct {
	// Name is the struct name
	Name string

	// Type is the parameter type (path, query, header, cookie)
	Type ParameterType

	// GoTypeName is the full Go type name
	GoTypeName string

	// Fields are the parameter fields
	Fields []*Field
}

// ParameterType represents the type of parameter
type ParameterType string

const (
	PathParameter   ParameterType = "path"
	QueryParameter  ParameterType = "query"
	HeaderParameter ParameterType = "header"
	CookieParameter ParameterType = "cookie"
)

// Endpoint represents an @endpoint annotated function
type Endpoint struct {
	// FuncName is the Go function name (for inline declaration lookup)
	FuncName string

	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string

	// Path is the URL path (e.g., /users/{id})
	Path string

	// OperationID is the operation ID
	OperationID string

	// Summary is a short summary
	Summary string

	// Description is a detailed description
	Description string

	// Tags are endpoint tags
	Tags []string

	// Deprecated indicates if the endpoint is deprecated
	Deprecated bool

	// Auth is the security scheme to use (overrides API default)
	Auth string

	// PathParams are the path parameter struct references
	PathParams []string

	// QueryParams are the query parameter struct references
	QueryParams []string

	// HeaderParams are the header parameter struct references
	HeaderParams []string

	// CookieParams are the cookie parameter struct references
	CookieParams []string

	// Request is the request body definition
	Request *RequestBody

	// Responses are the response definitions
	Responses map[string]*Response // Key is status code
}

// RequestBody represents a request body
type RequestBody struct {
	// ContentType is the content type (json, xml, form, etc.)
	ContentType string

	// Body is the body definition with optional bindings
	Body *Body
}

// Response represents a response definition
type Response struct {
	// StatusCode is the HTTP status code (200, 404, etc.)
	StatusCode string

	// ContentType is the content type
	ContentType string

	// Body is the body definition with optional bindings
	Body *Body

	// Description is the response description
	Description string

	// HeaderParams are the response header struct references
	HeaderParams []string
}

// Body represents a @body annotation with optional binding
type Body struct {
	// Schema is the schema name being referenced (e.g., "User", "[]User", "map[string]User")
	Schema string

	// Bind specifies the wrapper and field for envelope responses
	// e.g., @bind DataResponse.Data
	Bind *BindTarget
}

// BindTarget represents @bind Wrapper.Field syntax
type BindTarget struct {
	// Wrapper is the wrapper schema name (e.g., "DataResponse")
	Wrapper string

	// Field is the field to bind the body to (e.g., "Data")
	Field string
}

// Tag represents an API tag definition
type Tag struct {
	// Name is the tag name
	Name string

	// Description is the tag description
	Description string
}
