package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/resolver"
)

// Validator validates business rules for the resolved package
type Validator struct {
	errors []error
}

// ValidationError represents a validation error
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

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		errors: make([]error, 0),
	}
}

// Validate validates the resolved package
func (v *Validator) Validate(pkg *resolver.Package) error {
	v.errors = make([]error, 0)

	// Validate API
	if pkg.API == nil {
		v.addError("", "missing @api annotation")
	} else {
		v.validateAPI(pkg.API)
	}

	// Validate schemas
	for name, schema := range pkg.Schemas {
		v.validateSchema(name, schema)
	}

	// Validate parameters
	for name, param := range pkg.Parameters {
		v.validateParameter(name, param)
	}

	// Validate endpoints
	for _, endpoint := range pkg.Endpoints {
		v.validateEndpoint(endpoint, pkg)
	}

	// Return errors if any
	if len(v.errors) > 0 {
		return &MultiError{Errors: v.errors}
	}

	return nil
}

// MultiError contains multiple validation errors
type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d validation errors:\n", len(e.Errors)))
	for i, err := range e.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// addError adds a validation error
func (v *Validator) addError(path, message string) {
	v.errors = append(v.errors, &ValidationError{
		Path:    path,
		Message: message,
	})
}

// validateAPI validates API info
func (v *Validator) validateAPI(api *resolver.API) {
	if api.Title == "" {
		v.addError("@api", "missing required @title")
	}

	if api.Version == "" {
		v.addError("@api", "missing required @version")
	}

	// Validate security schemes
	for name, scheme := range api.SecuritySchemes {
		v.validateSecurityScheme(name, scheme)
	}

	// Validate security requirements reference existing schemes
	for i, reqs := range api.Security {
		for j, req := range reqs {
			if _, ok := api.SecuritySchemes[req.SchemeName]; !ok {
				v.addError(
					fmt.Sprintf("@api.@security[%d].@with[%d]", i, j),
					fmt.Sprintf("references unknown security scheme: %s", req.SchemeName),
				)
			}
		}
	}
}

// validateSecurityScheme validates a security scheme
func (v *Validator) validateSecurityScheme(name string, scheme *resolver.SecurityScheme) {
	path := fmt.Sprintf("@api.@securityScheme[%s]", name)

	if scheme.Type == "" {
		v.addError(path, "missing required @type")
		return
	}

	switch scheme.Type {
	case "http":
		if scheme.Scheme == "" {
			v.addError(path, "http security scheme missing @scheme")
		}
	case "apiKey":
		if scheme.In == "" {
			v.addError(path, "apiKey security scheme missing @in")
		}
		if scheme.ParameterName == "" {
			v.addError(path, "apiKey security scheme missing @name")
		}
	case "oauth2", "openIdConnect":
		// These would require additional fields we haven't implemented yet
	default:
		v.addError(path, fmt.Sprintf("unknown security scheme type: %s", scheme.Type))
	}
}

// validateSchema validates a schema
func (v *Validator) validateSchema(name string, schema *resolver.Schema) {
	path := fmt.Sprintf("@schema[%s]", name)

	if len(schema.Fields) == 0 {
		v.addError(path, "schema has no fields")
	}

	// Validate each field
	for _, field := range schema.Fields {
		v.validateField(path, field)
	}

	// Check for duplicate field names
	fieldNames := make(map[string]bool)
	for _, field := range schema.Fields {
		if fieldNames[field.Name] {
			v.addError(path, fmt.Sprintf("duplicate field name: %s", field.Name))
		}
		fieldNames[field.Name] = true
	}
}

// validateParameter validates a parameter struct
func (v *Validator) validateParameter(name string, param *resolver.ParameterStruct) {
	path := fmt.Sprintf("@%s[%s]", param.Type, name)

	if len(param.Fields) == 0 {
		v.addError(path, "parameter has no fields")
	}

	// Validate each field
	for _, field := range param.Fields {
		v.validateParameterField(path, param.Type, field)
	}

	// Check for duplicate field names
	fieldNames := make(map[string]bool)
	for _, field := range param.Fields {
		if fieldNames[field.Name] {
			v.addError(path, fmt.Sprintf("duplicate field name: %s", field.Name))
		}
		fieldNames[field.Name] = true
	}
}

// validateField validates a schema field
func (v *Validator) validateField(path string, field *resolver.Field) {
	fieldPath := fmt.Sprintf("%s.%s", path, field.GoName)

	// A type with no OpenAPI representation. The resolver marks it rather than
	// guessing a substitute, which is what used to turn a channel into a string.
	if field.Type != nil && field.Type.Shape == resolver.ShapeUnsupported {
		if name := field.Type.Ref; name != "" {
			v.addError(fieldPath, fmt.Sprintf(
				"references struct %s, which has no @schema annotation; annotate %s or use an anonymous struct",
				name, name))
		} else {
			v.addError(fieldPath, fmt.Sprintf("%s has no OpenAPI representation", field.GoType))
		}
	}

	// Validate enum values match type
	if len(field.Enum) > 0 {
		switch openAPIType := field.Type.ScalarName(); openAPIType {
		case "string", "integer":
			// OK - enum supported for string and integer types
		case "array":
			if items := field.Type.Elem.ScalarName(); items != "string" && items != "integer" {
				v.addError(fieldPath, "enum for arrays only supported with string or integer items")
			}
		default:
			v.addError(fieldPath, "enum only supported for string, integer, or array types")
		}
	}

	// Validate min/max constraints
	if field.Minimum != nil && field.Maximum != nil {
		if *field.Minimum > *field.Maximum {
			v.addError(fieldPath, "minimum cannot be greater than maximum")
		}
	}

	// Validate minLength/maxLength constraints
	if field.MinLength != nil && field.MaxLength != nil {
		if *field.MinLength > *field.MaxLength {
			v.addError(fieldPath, "minLength cannot be greater than maxLength")
		}
	}

	// Validate length constraints are for strings
	if (field.MinLength != nil || field.MaxLength != nil) && field.Type.ScalarName() != "string" {
		v.addError(fieldPath, "minLength/maxLength only valid for string types")
	}

	// Validate minItems/maxItems constraints
	if field.MinItems != nil && field.MaxItems != nil {
		if *field.MinItems > *field.MaxItems {
			v.addError(fieldPath, "minItems cannot be greater than maxItems")
		}
	}

	// Validate items constraints are for arrays
	if (field.MinItems != nil || field.MaxItems != nil) && field.Type.ScalarName() != "array" {
		v.addError(fieldPath, "minItems/maxItems only valid for array types")
	}

	// Validate uniqueItems is for arrays
	if field.UniqueItems && field.Type.ScalarName() != "array" {
		v.addError(fieldPath, "uniqueItems only valid for array types")
	}

	// Validate pattern is for strings
	if field.Pattern != "" && field.Type.ScalarName() != "string" {
		v.addError(fieldPath, "pattern only valid for string types")
	}

	// Validate pattern is valid regex
	if field.Pattern != "" {
		if _, err := regexp.Compile(field.Pattern); err != nil {
			v.addError(fieldPath, fmt.Sprintf("invalid pattern regex: %v", err))
		}
	}

	// Validate readOnly and writeOnly are mutually exclusive
	if field.ReadOnly && field.WriteOnly {
		v.addError(fieldPath, "readOnly and writeOnly cannot both be true")
	}
}

// validateParameterField validates a parameter field with type-specific rules
func (v *Validator) validateParameterField(path, paramType string, field *resolver.Field) {
	fieldPath := fmt.Sprintf("%s.%s", path, field.GoName)

	// Path parameters cannot be pointers/nullable
	if paramType == "path" && field.Nullable {
		v.addError(fieldPath, "path parameters cannot be nullable (no pointer types)")
	}

	// Path parameters cannot be arrays
	if paramType == "path" && field.Type.IsArray() {
		v.addError(fieldPath, "path parameters cannot be arrays")
	}

	// Header parameters cannot be arrays
	if paramType == "header" && field.Type.IsArray() {
		v.addError(fieldPath, "header parameters cannot be arrays")
	}

	// Cookie parameters cannot be arrays
	if paramType == "cookie" && field.Type.IsArray() {
		v.addError(fieldPath, "cookie parameters cannot be arrays")
	}

	// Query parameters can be arrays (this is allowed)

	// A parameter is a scalar or a list of scalars, and nothing else. net/http
	// hands parameters over as map[string][]string, so there is no structured
	// value to decode into — an object-shaped parameter would need deepObject,
	// which specgen does not implement.
	if !isParameterShape(field.Type) {
		v.addError(fieldPath, fmt.Sprintf(
			"%s parameters must be a scalar or a list of scalars, but %s is %s",
			paramType, field.GoName, describeShape(field.Type)))
		return
	}

	// Run standard field validation
	v.validateField(path, field)
}

// isParameterShape reports whether a type can be carried as a URL or header
// parameter.
func isParameterShape(t *resolver.TypeRef) bool {
	if t == nil {
		return false
	}
	switch t.Shape {
	case resolver.ShapeScalar:
		return true
	case resolver.ShapeArray:
		return t.Elem != nil && t.Elem.Shape == resolver.ShapeScalar
	}
	return false
}

// describeShape names a type shape for an error message.
func describeShape(t *resolver.TypeRef) string {
	if t == nil {
		return "an unresolved type"
	}
	switch t.Shape {
	case resolver.ShapeMap:
		return "a map"
	case resolver.ShapeObject:
		return "an anonymous struct"
	case resolver.ShapeRef:
		return "the schema " + t.Ref
	case resolver.ShapeAny:
		return "an unconstrained value"
	case resolver.ShapeArray:
		return "a list of " + describeShape(t.Elem)
	case resolver.ShapeUnsupported:
		return t.Reason
	}
	return "a scalar"
}

// validateEndpoint validates an endpoint
func (v *Validator) validateEndpoint(endpoint *resolver.Endpoint, pkg *resolver.Package) {
	path := fmt.Sprintf("@endpoint[%s %s]", endpoint.Method, endpoint.Path)

	// Validate method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true,
		"DELETE": true, "HEAD": true, "OPTIONS": true,
	}
	if !validMethods[endpoint.Method] {
		v.addError(path, fmt.Sprintf("invalid HTTP method: %s", endpoint.Method))
	}

	// Validate path
	if endpoint.Path == "" {
		v.addError(path, "missing path")
	} else {
		v.validatePath(path, endpoint.Path)
	}

	// Extract path variables from path
	pathVars := extractPathVariables(endpoint.Path)

	// Validate path parameters match path variables
	v.validatePathParameters(path, pathVars, endpoint.Parameters)

	// Validate request body
	if endpoint.Request != nil {
		v.validateRequestBody(path, endpoint.Request, pkg.Schemas)
	}

	if len(endpoint.Responses) == 0 {
		v.addError(path, "endpoint must have at least one response")
	}

	for _, response := range endpoint.Responses {
		v.validateResponse(path, response, pkg.Schemas)
	}

	// Validate no parameter name conflicts
	v.validateParameterConflicts(path, endpoint)

	// Validate tags reference defined API-level tags. Deliberately not guarded
	// on the API having declared any: an API with no @tag blocks is exactly the
	// case where every endpoint tag is undefined.
	v.validateEndpointTags(path, endpoint.Tags, pkg.API.Tags)
}

// validatePath validates the path format
func (v *Validator) validatePath(endpointPath, path string) {
	if !strings.HasPrefix(path, "/") {
		v.addError(endpointPath, "path must start with /")
	}

	// Validate path variables are in {var} format
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		varName := match[1]
		if varName == "" {
			v.addError(endpointPath, "empty path variable")
		}
		// Check for valid variable name (alphanumeric and underscore)
		if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(varName) {
			v.addError(endpointPath, fmt.Sprintf("invalid path variable name: %s", varName))
		}
	}
}

// extractPathVariables extracts variable names from path
func extractPathVariables(path string) []string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)
	vars := make([]string, len(matches))
	for i, match := range matches {
		vars[i] = match[1]
	}
	return vars
}

// validatePathParameters checks that the path template and the declared path
// parameters describe the same set of names.
func (v *Validator) validatePathParameters(path string, pathVars []string, params []*resolver.Parameter) {
	// Create map of path variables
	pathVarMap := make(map[string]bool)
	for _, varName := range pathVars {
		pathVarMap[varName] = true
	}

	// Named and in-function declarations are one list now, so there is one
	// place to collect from. Ordered rather than a map so that when several
	// declared parameters are unused, they are reported the same way each run.
	var declared []string
	seen := make(map[string]bool)
	for _, param := range params {
		if param.In != "path" || seen[param.Field.Name] {
			continue
		}
		seen[param.Field.Name] = true
		declared = append(declared, param.Field.Name)
	}

	for _, varName := range pathVars {
		if !seen[varName] {
			v.addError(path, fmt.Sprintf("path variable {%s} has no corresponding @path parameter", varName))
		}
	}

	for _, paramName := range declared {
		if !pathVarMap[paramName] {
			v.addError(path, fmt.Sprintf("@path parameter %s not used in path", paramName))
		}
	}
}

// validateRequestBody validates a request body
func (v *Validator) validateRequestBody(path string, request *resolver.RequestBody, schemas map[string]*resolver.Schema) {
	if request.ContentType == "" {
		v.addError(path+".@request", "missing @contentType")
	}

	// An in-function @request is its own body, so only the named form can be
	// missing one.
	if request.Inline != nil {
		if request.Inline.Bind != nil {
			v.validateBindTarget(path+".@request", request.Inline.Bind, schemas)
		}
		return
	}

	if request.Body == nil || request.Body.Schema == "" {
		v.addError(path+".@request", "missing @body")
		return
	}

	v.validateBodySchemaExists(path+".@request", request.Body, schemas)

	if request.Body.Bind != nil {
		v.validateBindTarget(path+".@request", request.Body.Bind, schemas)
	}
}

// validateBodySchemaExists checks that a body naming a schema names one that
// exists. The body's shape already says which name that is, whatever nesting
// the annotation was written with.
func (v *Validator) validateBodySchemaExists(path string, body *resolver.Body, schemas map[string]*resolver.Schema) {
	ref := body.Type
	for ref != nil && (ref.Shape == resolver.ShapeArray || ref.Shape == resolver.ShapeMap) {
		ref = ref.Elem
	}

	if ref == nil || ref.Shape != resolver.ShapeRef {
		return
	}

	if _, ok := schemas[ref.Ref]; !ok {
		v.addError(path, fmt.Sprintf("references unknown schema: %s", ref.Ref))
	}
}

// validateResponse validates a response
func (v *Validator) validateResponse(path string, response *resolver.Response, schemas map[string]*resolver.Schema) {
	statusCode := response.StatusCode
	responsePath := fmt.Sprintf("%s.@response[%s]", path, statusCode)

	// Validate status code: 3-digit (200), range (2XX), or "default"
	if statusCode != "default" && !regexp.MustCompile(`^[1-5](\d{2}|XX)$`).MatchString(statusCode) {
		v.addError(responsePath, fmt.Sprintf("invalid status code: %s", statusCode))
	}

	// An in-function @response is its own body, so there is no schema name to
	// check — only the envelope it may be bound into.
	if response.Inline != nil {
		if response.Inline.Bind != nil {
			v.validateBindTarget(responsePath, response.Inline.Bind, schemas)
		}
		return
	}

	// A named body is optional: 204 No Content has none.
	if response.Body != nil && response.Body.Schema != "" {
		v.validateBodySchemaExists(responsePath, response.Body, schemas)

		if response.Body.Bind != nil {
			v.validateBindTarget(responsePath, response.Body.Bind, schemas)
		}
	}
}

// validateParameterConflicts checks for duplicate parameters.
//
// OpenAPI identifies a parameter by (name, location), not by name: a path "id"
// and a query "id" are two different parameters and both are legal. Keying on
// the name alone rejected that, which is legal input specgen refused to
// describe.
//
// The duplicates that do matter are two declarations of the same name in the
// same location, since the emitted document would list one parameter twice.
func (v *Validator) validateParameterConflicts(path string, endpoint *resolver.Endpoint) {
	type key struct{ name, in string }

	seen := make(map[key]bool)
	for _, param := range endpoint.Parameters {
		k := key{param.Field.Name, param.In}
		if seen[k] {
			v.addError(path, fmt.Sprintf("duplicate %s parameter: %s", param.In, param.Field.Name))
		}
		seen[k] = true
	}
}

// validateBindTarget validates that a @bind target references a valid wrapper schema and field
func (v *Validator) validateBindTarget(path string, bind *resolver.BindTarget, schemas map[string]*resolver.Schema) {
	bindPath := path + ".@bind"

	// Check wrapper schema exists
	if bind.WrapperSchema == nil {
		v.addError(bindPath, fmt.Sprintf("references unknown wrapper schema: %s", bind.Wrapper))
		return
	}

	// Check field exists in wrapper schema
	found := false
	for _, field := range bind.WrapperSchema.Fields {
		if field.GoName == bind.Field {
			found = true
			break
		}
	}
	if !found {
		v.addError(bindPath, fmt.Sprintf("wrapper schema %q has no field %q", bind.Wrapper, bind.Field))
	}
}

// validateEndpointTags validates that endpoint tags reference defined API-level tags
func (v *Validator) validateEndpointTags(path string, endpointTags []string, apiTags []*resolver.Tag) {
	// Build map of defined tag names
	definedTags := make(map[string]bool)
	for _, tag := range apiTags {
		definedTags[tag.Name] = true
	}

	// Check each endpoint tag
	for _, tagName := range endpointTags {
		if !definedTags[tagName] {
			v.addError(path, fmt.Sprintf("endpoint uses undefined tag: %s (define it at API level with @tag)", tagName))
		}
	}
}
