package resolver

// Package contains the fully resolved parsed package with type information
type Package struct {
	// Original parsed package
	PackageName string
	API         *API
	Schemas     map[string]*Schema
	Parameters  map[string]*ParameterStruct
	Endpoints   []*Endpoint
}

// API contains resolved API info
type API struct {
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

// Schema contains a schema with resolved type information
type Schema struct {
	Name        string
	GoTypeName  string
	Description string
	Deprecated  bool
	Fields      []*Field

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

// ParameterStruct contains a parameter struct with resolved type information
type ParameterStruct struct {
	Name       string
	Type       string // "path", "query", "header", "cookie"
	GoTypeName string
	Fields     []*Field
}

// Shape is the structural kind of a resolved Go type. It is what decides how
// the generator emits a field, and it is exhaustive: every Go type a field can
// have lands on exactly one of these.
type Shape int

const (
	// ShapeScalar is a value with an OpenAPI primitive type. TypeRef.Type says
	// which one. time.Time and the other special types land here too — they are
	// Go structs that serialize as scalars.
	ShapeScalar Shape = iota

	// ShapeArray is a slice or fixed-size array. Elem is the element type.
	ShapeArray

	// ShapeMap is a map. Elem is the value type; OpenAPI keys are always strings.
	ShapeMap

	// ShapeObject is an anonymous struct, inlined at the point of use.
	// Fields are its members.
	ShapeObject

	// ShapeRef is a named struct carrying @schema. Ref is its name, and the
	// field emits a $ref rather than repeating the definition.
	ShapeRef

	// ShapeAny is any/interface{} — an empty schema, which accepts any JSON value.
	ShapeAny

	// ShapeTypeParam is a generic type parameter. Generic structs are templates
	// and never reach components, but they still have to resolve: validateSchema
	// rejects a schema with no fields.
	ShapeTypeParam

	// ShapeUnsupported is a type with no OpenAPI representation. Reason says
	// which type and why, for the error message.
	ShapeUnsupported
)

// TypeRef describes a resolved Go type as a shape the generator can emit
// directly.
//
// It is recursive because Go types are: map[string][]User is a map whose
// element is an array whose element is a reference. The flat alternative — a
// bool per container kind plus a single element type name — could only ever
// describe one level, which is why nested containers used to lose their inner
// one.
type TypeRef struct {
	// Shape decides which of the remaining fields are meaningful.
	Shape Shape

	// Type is the OpenAPI primitive name when Shape is ShapeScalar:
	// "string", "integer", "number", or "boolean".
	Type string

	// Format refines a scalar: "int64", "date-time", "byte", and so on.
	// It is carried at every level, so an element keeps the format its own
	// type has.
	Format string

	// Ref is the schema name when Shape is ShapeRef. It is also set when
	// Shape is ShapeUnsupported because the type is a struct without @schema,
	// so callers can name the type they want annotated.
	Ref string

	// Elem is the element type of an array or the value type of a map.
	Elem *TypeRef

	// Fields are an anonymous struct's members when Shape is ShapeObject.
	Fields []*Field

	// Reason describes an unsupported type as a noun phrase ("a channel",
	// "a struct (Coords) with no @schema annotation"), so it reads correctly in
	// any sentence that reports it.
	Reason string
}

// IsArray reports whether this is an array, which several callers need without
// caring about the element.
func (t *TypeRef) IsArray() bool { return t != nil && t.Shape == ShapeArray }

// ScalarName is the OpenAPI type name to use when a value has to be described
// by a single word — an enum's item type, a header schema. Containers and
// objects have no such word, and report "".
func (t *TypeRef) ScalarName() string {
	if t == nil {
		return ""
	}
	switch t.Shape {
	case ShapeScalar:
		return t.Type
	case ShapeArray:
		return "array"
	case ShapeMap, ShapeObject:
		return "object"
	}
	return ""
}

// Field contains a field with resolved type information
type Field struct {
	// From annotation or inferred
	Name        string
	GoName      string
	Description string
	Required    bool
	Nullable    bool
	Deprecated  bool
	ReadOnly    bool
	WriteOnly   bool

	// GoType is the original Go type string, kept for error messages.
	GoType string

	// Type is the resolved shape of the field's Go type.
	Type *TypeRef

	// Format is the field's format after @format has had its say. It seeds
	// from Type.Format and overrides it, since an annotation names the format
	// of the field rather than of some element inside it.
	Format string

	// Validation constraints
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
}

// Endpoint contains an endpoint with resolved types.
//
// Parameters and Responses are emission-ready: merged across named and
// in-function declarations, ordered, and with conflicts already settled. Doing
// that here rather than in the generator makes both rules testable without
// rendering a document, and keeps the generator from being the only place that
// knows what happens when two declarations claim the same status code.
type Endpoint struct {
	FuncName    string
	Method      string
	Path        string
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
	Auth        string

	// Request is the request body, from either @request form.
	Request *RequestBody

	// Parameters are every parameter the operation accepts, in emission order.
	Parameters []*Parameter

	// Responses are every response, sorted by status code.
	Responses []*Response
}

// Parameter is one OpenAPI parameter: a resolved field plus where it goes.
//
// The field itself is the same shape whether it came from a named @query struct
// or an in-function one, so only the location has to be attached here.
type Parameter struct {
	// In is the location: "path", "query", "header", or "cookie".
	In string

	// Field is the parameter's name, type, and constraints.
	Field *Field
}

// InlineBody contains resolved inline body fields with optional binding
type InlineBody struct {
	ContentType string
	Fields      []*Field
	Bind        *BindTarget
	Headers     []*ParameterStruct // Response headers (for inline responses)
	Description string             // Response description
}

// RequestBody contains a request body with resolved schema
type RequestBody struct {
	ContentType string

	// Body is a named body: @request { @body User }.
	Body *Body

	// Inline is an in-function body: the struct declared under @request.
	// Exactly one of Body and Inline is set.
	Inline *InlineBody

	Required bool
}

// Response contains a response with resolved schema
type Response struct {
	StatusCode  string
	Description string
	ContentType string

	// Body is a named body: @response 200 { @body User }.
	Body *Body

	// Inline is an in-function body: the struct declared under @response.
	// At most one of Body and Inline is set.
	Inline *InlineBody

	Headers []*ParameterStruct
}

// Body contains the resolved body with optional binding
type Body struct {
	// Schema is the schema name as written (e.g., "User", "[]User",
	// "map[string]User"), kept for validation messages.
	Schema string

	// Type is the shape the body renders as.
	Type *TypeRef

	// Bind contains the resolved wrapper binding
	// When non-nil, the body should be wrapped in the specified envelope
	Bind *BindTarget
}

// BindTarget represents a resolved @bind Wrapper.Field annotation
type BindTarget struct {
	// Wrapper is the wrapper schema name (e.g., "DataResponse")
	Wrapper string

	// Field is the field to bind the body to (e.g., "Data")
	Field string

	// WrapperSchema is the resolved wrapper schema (for inlining)
	WrapperSchema *Schema
}
