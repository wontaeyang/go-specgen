package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/parser"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
)

// The validator checks the rules the generator assumes but cannot enforce:
// that annotations reference things that exist, that constraints suit the
// types they are attached to, and that paths and parameters agree.
//
// Every rule is checked; nothing stops at the first failure. The resolved IR
// is ordered, so the errors come out in a stable order too.

// ValidationError is one broken rule, with the annotation path it was found
// at, e.g. "@schema[User].Email".
type ValidationError struct {
	Message string
	Path    string
}

func (e *ValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

// MultiError collects every validation error of one run.
type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%d validation errors:\n", len(e.Errors))
	for i, err := range e.Errors {
		fmt.Fprintf(&sb, "  %d. %s\n", i+1, err.Error())
	}
	return sb.String()
}

// Unwrap exposes the collected errors to errors.Is and errors.As.
func (e *MultiError) Unwrap() []error { return e.Errors }

// validMethods are the HTTP methods an @endpoint may declare.
var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true,
	"DELETE": true, "HEAD": true, "OPTIONS": true,
}

// pathVariablePattern matches the {name} placeholders of a path.
var pathVariablePattern = regexp.MustCompile(`\{([^}]+)\}`)

// pathVariableName is the shape of a usable path variable name.
var pathVariableName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// validator accumulates the errors of one run.
type validator struct {
	// schemas is the resolved schemas by name, for reference checks.
	schemas map[string]*resolver.Schema

	// schemes is the declared security schemes by name, for @security and
	// @auth reference checks.
	schemes map[string]bool

	errors []error
}

// Validate checks a resolved package and returns every rule it breaks as a
// *MultiError, or nil when it breaks none.
func Validate(pkg *resolver.Package) error {
	v := &validator{
		schemas: make(map[string]*resolver.Schema, len(pkg.Schemas)),
		schemes: make(map[string]bool),
	}
	for _, schema := range pkg.Schemas {
		v.schemas[schema.Name] = schema
	}
	if pkg.API != nil {
		for _, scheme := range pkg.API.SecuritySchemes {
			v.schemes[scheme.Name] = true
		}
	}

	if pkg.API == nil {
		v.add("", "missing @api annotation")
	} else {
		v.validateAPI(pkg.API)
	}

	for _, schema := range pkg.Schemas {
		v.validateSchema(schema)
	}

	v.validateUniqueRoutes(pkg.Endpoints)
	v.validateUniqueOperationIDs(pkg.Endpoints)

	for _, endpoint := range pkg.Endpoints {
		v.validateEndpoint(endpoint, pkg.API)
	}

	if len(v.errors) > 0 {
		return &MultiError{Errors: v.errors}
	}
	return nil
}

func (v *validator) add(path, format string, args ...any) {
	v.errors = append(v.errors, &ValidationError{
		Path:    path,
		Message: fmt.Sprintf(format, args...),
	})
}

// validateAPI checks the document metadata.
func (v *validator) validateAPI(api *parser.APIInfo) {
	if api.Title == "" {
		v.add("@api", "missing required @title")
	}
	if api.Version == "" {
		v.add("@api", "missing required @version")
	}

	for _, scheme := range api.SecuritySchemes {
		v.validateSecurityScheme(scheme)
	}

	for i, group := range api.Security {
		for j, requirement := range group {
			if !v.schemes[requirement.SchemeName] {
				v.add(fmt.Sprintf("@api.@security[%d].@with[%d]", i, j),
					"references unknown security scheme: %s", requirement.SchemeName)
			}
		}
	}
}

// validateSecurityScheme checks that a scheme carries the fields its type
// needs.
func (v *validator) validateSecurityScheme(scheme *parser.SecurityScheme) {
	path := fmt.Sprintf("@api.@securityScheme[%s]", scheme.Name)

	switch scheme.Type {
	case "":
		v.add(path, "missing required @type")
	case "http":
		if scheme.Scheme == "" {
			v.add(path, "http security scheme missing @scheme")
		}
	case "apiKey":
		if scheme.In == "" {
			v.add(path, "apiKey security scheme missing @in")
		}
		if scheme.ParameterName == "" {
			v.add(path, "apiKey security scheme missing @name")
		}
	case "oauth2", "openIdConnect":
		// Their extra fields are not modelled yet.
	default:
		v.add(path, "unknown security scheme type: %s", scheme.Type)
	}
}

// validateSchema checks a named schema and its fields.
func (v *validator) validateSchema(schema *resolver.Schema) {
	path := fmt.Sprintf("@schema[%s]", schema.Name)

	if len(schema.Fields) == 0 {
		v.add(path, "schema has no fields")
	}

	for _, field := range schema.Fields {
		v.validateField(path, field)
	}
	v.validateUniqueNames(path, schema.Fields)
}

// validateUniqueNames checks that no two fields serialize under the same wire
// name, which would silently drop one of them.
func (v *validator) validateUniqueNames(path string, fields []*resolver.Field) {
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		if seen[field.Name] {
			v.add(path, "duplicate field name: %s", field.Name)
		}
		seen[field.Name] = true
	}
}

// validateField checks the constraints of one field against the type it is
// attached to.
func (v *validator) validateField(path string, field *resolver.Field) {
	fieldPath := fmt.Sprintf("%s.%s", path, field.GoName)
	fieldType := emittedType(field)

	if field.Unresolved != "" {
		v.add(fieldPath, "field references struct '%s' which is not a @schema. "+
			"Add @schema annotation to %s or use an anonymous struct",
			field.Unresolved, field.Unresolved)
	}

	if len(field.Enum) > 0 {
		switch fieldType {
		case "string", "integer":
		case "array":
			if field.Type.Items != "string" && field.Type.Items != "integer" {
				v.add(fieldPath, "enum for arrays only supported with string or integer items")
			}
		default:
			v.add(fieldPath, "enum only supported for string, integer, or array types")
		}
	}

	if field.Minimum != nil && field.Maximum != nil && *field.Minimum > *field.Maximum {
		v.add(fieldPath, "minimum cannot be greater than maximum")
	}
	if field.MinLength != nil && field.MaxLength != nil && *field.MinLength > *field.MaxLength {
		v.add(fieldPath, "minLength cannot be greater than maxLength")
	}
	if (field.MinLength != nil || field.MaxLength != nil) && fieldType != "string" {
		v.add(fieldPath, "minLength/maxLength only valid for string types")
	}

	if field.MinItems != nil && field.MaxItems != nil && *field.MinItems > *field.MaxItems {
		v.add(fieldPath, "minItems cannot be greater than maxItems")
	}
	if (field.MinItems != nil || field.MaxItems != nil) && fieldType != "array" {
		v.add(fieldPath, "minItems/maxItems only valid for array types")
	}
	if field.UniqueItems && fieldType != "array" {
		v.add(fieldPath, "uniqueItems only valid for array types")
	}

	if field.Pattern != "" {
		if fieldType != "string" {
			v.add(fieldPath, "pattern only valid for string types")
		}
		if _, err := regexp.Compile(field.Pattern); err != nil {
			v.add(fieldPath, "invalid pattern regex: %v", err)
		}
	}

	if field.ReadOnly && field.WriteOnly {
		v.add(fieldPath, "readOnly and writeOnly cannot both be true")
	}
}

// emittedType is the type keyword a field emits, which the constraint rules
// are written against. An any-typed field emits no type at all.
func emittedType(field *resolver.Field) string {
	switch {
	case field.Type.IsArray || len(field.ItemsInline) > 0:
		return "array"
	case field.Type.IsMap || len(field.Inline) > 0 || len(field.MapValueInline) > 0:
		return "object"
	case field.Type.Ref != "":
		return "object"
	case field.Type.IsAny:
		return ""
	}
	return field.Type.OpenAPI
}

// validateUniqueRoutes checks that no two operations share a method and path.
// A document has one operation per method per path, so a repeat would silently
// drop everything but the last one.
func (v *validator) validateUniqueRoutes(endpoints []*resolver.Endpoint) {
	seen := make(map[string]bool, len(endpoints))
	for _, endpoint := range endpoints {
		route := endpoint.Method + " " + endpoint.Path
		if seen[route] {
			v.add(endpointPath(endpoint), "duplicate operation: %s is declared by more than one endpoint", route)
		}
		seen[route] = true
	}
}

// validateUniqueOperationIDs checks that no two operations share an
// operationId. OpenAPI requires it to be unique across the document, because
// client generators use it to name the generated method.
//
// An endpoint without an @operationID is unnamed, not named "": those are not
// compared against each other.
func (v *validator) validateUniqueOperationIDs(endpoints []*resolver.Endpoint) {
	seen := make(map[string]bool, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.OperationID == "" {
			continue
		}
		if seen[endpoint.OperationID] {
			v.add(endpointPath(endpoint), "duplicate @operationID: %s is used by more than one endpoint", endpoint.OperationID)
		}
		seen[endpoint.OperationID] = true
	}
}

// endpointPath is the annotation path an endpoint's errors are reported at.
func endpointPath(endpoint *resolver.Endpoint) string {
	return fmt.Sprintf("@endpoint[%s %s]", endpoint.Method, endpoint.Path)
}

// validateEndpoint checks one operation: its method and path, the parameters
// against the path variables, and its request and responses.
func (v *validator) validateEndpoint(endpoint *resolver.Endpoint, api *parser.APIInfo) {
	path := endpointPath(endpoint)

	if !validMethods[endpoint.Method] {
		v.add(path, "invalid HTTP method: %s", endpoint.Method)
	}

	if endpoint.Path == "" {
		v.add(path, "missing path")
	} else {
		v.validatePath(path, endpoint.Path)
	}

	// An @auth names the one scheme this operation requires, overriding the
	// API default. A name with no scheme behind it emits a security
	// requirement no consumer can satisfy.
	if endpoint.Auth != "" && !v.schemes[endpoint.Auth] {
		v.add(path, "@auth references unknown security scheme: %s", endpoint.Auth)
	}

	v.validateParameters(path, endpoint)

	if endpoint.Request != nil {
		v.validateRequest(path, endpoint.Request)
	}

	if len(endpoint.Responses) == 0 {
		v.add(path, "endpoint must have at least one response")
	}
	for _, response := range endpoint.Responses {
		v.validateResponse(path, response)
	}

	// api is nil when @api is missing; Validate records that separately and
	// still checks the endpoints.
	if api != nil && len(api.Tags) > 0 && len(endpoint.Tags) > 0 {
		v.validateTags(path, endpoint.Tags, api.Tags)
	}
}

// validatePath checks the shape of the path and its variables.
func (v *validator) validatePath(endpointPath, path string) {
	if !strings.HasPrefix(path, "/") {
		v.add(endpointPath, "path must start with /")
	}

	for _, name := range pathVariables(path) {
		if name == "" {
			v.add(endpointPath, "empty path variable")
		}
		if !pathVariableName.MatchString(name) {
			v.add(endpointPath, "invalid path variable name: %s", name)
		}
	}
}

// pathVariables returns the {name} placeholders of a path, in order.
func pathVariables(path string) []string {
	matches := pathVariablePattern.FindAllStringSubmatch(path, -1)
	names := make([]string, len(matches))
	for i, match := range matches {
		names[i] = match[1]
	}
	return names
}

// validateParameters checks the operation parameters: that the path variables
// and the path parameters cover each other exactly, that no two parameters
// share a name, and that each field suits the kind it is emitted as.
func (v *validator) validateParameters(path string, endpoint *resolver.Endpoint) {
	declared := make(map[string]bool)
	for _, param := range endpoint.Parameters {
		if param.In == parser.ParamPath {
			declared[param.Field.Name] = true
		}
	}

	variables := make(map[string]bool)
	for _, name := range pathVariables(endpoint.Path) {
		variables[name] = true
		if !declared[name] {
			v.add(path, "path variable {%s} has no corresponding @path parameter", name)
		}
	}
	for _, param := range endpoint.Parameters {
		if param.In == parser.ParamPath && !variables[param.Field.Name] {
			v.add(path, "@path parameter %s not used in path", param.Field.Name)
		}
	}

	// A name may appear once per operation, whatever kind it is sent as.
	kinds := make(map[string]string, len(endpoint.Parameters))
	for _, param := range endpoint.Parameters {
		if previous, taken := kinds[param.Field.Name]; taken {
			v.add(path, "parameter name conflict: %s appears in both %s and %s parameters",
				param.Field.Name, previous, param.In)
		}
		kinds[param.Field.Name] = param.In

		v.validateParameterField(path, param)
	}
}

// validateParameterField checks one parameter against the rules of its kind.
// Parameters are sent as text in a URL or a header, which limits their shape.
func (v *validator) validateParameterField(path string, param *resolver.Param) {
	kindPath := fmt.Sprintf("%s.@%s", path, param.In)
	fieldPath := fmt.Sprintf("%s.%s", kindPath, param.Field.GoName)
	isArray := emittedType(param.Field) == "array"

	switch param.In {
	case parser.ParamPath:
		if param.Field.Nullable {
			v.add(fieldPath, "path parameters cannot be nullable (no pointer types)")
		}
		if isArray {
			v.add(fieldPath, "path parameters cannot be arrays")
		}
	case parser.ParamHeader:
		if isArray {
			v.add(fieldPath, "header parameters cannot be arrays")
		}
	case parser.ParamCookie:
		if isArray {
			v.add(fieldPath, "cookie parameters cannot be arrays")
		}
	}
	// Query parameters may be arrays: repeated values are how a list is sent.

	v.validateField(kindPath, param.Field)
}

// validateRequest checks the request body.
func (v *validator) validateRequest(path string, request *resolver.Request) {
	requestPath := path + ".@request"

	if request.ContentType == "" {
		v.add(requestPath, "missing @contentType")
	}
	if request.Content == nil {
		return
	}
	v.validateContent(requestPath, request.Content)
}

// validateResponse checks one response.
func (v *validator) validateResponse(path string, response *resolver.Response) {
	responsePath := fmt.Sprintf("%s.@response[%s]", path, response.Status)

	if !parser.ValidStatusCode(response.Status) {
		v.add(responsePath, "invalid status code: %s", response.Status)
	}

	// A response without content is legitimate: 204 and friends.
	if response.Content == nil {
		return
	}
	v.validateContent(responsePath, response.Content)
}

// validateContent checks that a body references things that exist.
func (v *validator) validateContent(path string, content *resolver.Content) {
	// A referenced schema must exist; a primitive body needs no lookup, and an
	// inline body carries its own fields.
	if content.Ref != nil && content.Ref.Schema != "" {
		if _, ok := v.schemas[content.Ref.Schema]; !ok {
			v.add(path, "references unknown schema: %s", content.Ref.Schema)
		}
	}

	for _, field := range content.Fields {
		v.validateField(path, field)
	}

	if content.Bind != nil {
		v.validateBind(path, content.Bind)
	}
}

// validateBind checks that a @bind names a wrapper schema that exists and a
// field of it to put the body in.
func (v *validator) validateBind(path string, bind *resolver.BindTarget) {
	bindPath := path + ".@bind"

	if bind.Wrapper == nil {
		v.add(bindPath, "references unknown wrapper schema: %s", bind.Name)
		return
	}

	for _, field := range bind.Wrapper.Fields {
		if field.GoName == bind.Field {
			return
		}
	}
	v.add(bindPath, "wrapper schema %q has no field %q", bind.Name, bind.Field)
}

// validateTags checks that the tags of an operation are defined at API level.
func (v *validator) validateTags(path string, endpointTags []string, apiTags []*parser.Tag) {
	defined := make(map[string]bool, len(apiTags))
	for _, tag := range apiTags {
		defined[tag.Name] = true
	}

	for _, name := range endpointTags {
		if !defined[name] {
			v.add(path, "endpoint uses undefined tag: %s (define it at API level with @tag)", name)
		}
	}
}
