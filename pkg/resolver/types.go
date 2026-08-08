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
	Fields []*ResolvedField

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

// ResolvedField contains a field with resolved type information
type ResolvedField struct {
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
	// Schema is the schema name as written (e.g., "User", "[]User",
	// "map[string]User"), kept for validation messages.
	Schema string

	// Type is the shape the body renders as.
	Type *TypeRef

	// Bind contains the resolved wrapper binding
	// When non-nil, the body should be wrapped in the specified envelope
	Bind *ResolvedBindTarget
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
