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
func (v *Validator) Validate(pkg *resolver.ResolvedPackage) error {
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
func (v *Validator) validateAPI(api *resolver.ResolvedAPI) {
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
func (v *Validator) validateSchema(name string, schema *resolver.ResolvedSchema) {
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
func (v *Validator) validateParameter(name string, param *resolver.ResolvedParameter) {
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
func (v *Validator) validateField(path string, field *resolver.ResolvedField) {
	fieldPath := fmt.Sprintf("%s.%s", path, field.GoName)

	// Check for unresolved struct types (named structs not marked with @schema)
	if field.IsUnresolvedStruct {
		v.addError(fieldPath, fmt.Sprintf(
			"field references struct '%s' which is not a @schema. Add @schema annotation to %s or use an anonymous struct",
			field.UnresolvedTypeName, field.UnresolvedTypeName,
		))
	}

	// Validate enum values match type
	if len(field.Enum) > 0 {
		switch field.OpenAPIType {
		case "string", "integer":
			// OK - enum supported for string and integer types
		case "array":
			if field.ItemsType != "string" && field.ItemsType != "integer" {
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
	if (field.MinLength != nil || field.MaxLength != nil) && field.OpenAPIType != "string" {
		v.addError(fieldPath, "minLength/maxLength only valid for string types")
	}

	// Validate minItems/maxItems constraints
	if field.MinItems != nil && field.MaxItems != nil {
		if *field.MinItems > *field.MaxItems {
			v.addError(fieldPath, "minItems cannot be greater than maxItems")
		}
	}

	// Validate items constraints are for arrays
	if (field.MinItems != nil || field.MaxItems != nil) && field.OpenAPIType != "array" {
		v.addError(fieldPath, "minItems/maxItems only valid for array types")
	}

	// Validate uniqueItems is for arrays
	if field.UniqueItems && field.OpenAPIType != "array" {
		v.addError(fieldPath, "uniqueItems only valid for array types")
	}

	// Validate pattern is for strings
	if field.Pattern != "" && field.OpenAPIType != "string" {
		v.addError(fieldPath, "pattern only valid for string types")
	}

	// Validate pattern is valid regex
	if field.Pattern != "" {
		if _, err := regexp.Compile(field.Pattern); err != nil {
			v.addError(fieldPath, fmt.Sprintf("invalid pattern regex: %v", err))
		}
	}
}

// validateParameterField validates a parameter field with type-specific rules
func (v *Validator) validateParameterField(path, paramType string, field *resolver.ResolvedField) {
	fieldPath := fmt.Sprintf("%s.%s", path, field.GoName)

	// Path parameters cannot be pointers/nullable
	if paramType == "path" && field.Nullable {
		v.addError(fieldPath, "path parameters cannot be nullable (no pointer types)")
	}

	// Path parameters cannot be arrays
	if paramType == "path" && field.IsArray {
		v.addError(fieldPath, "path parameters cannot be arrays")
	}

	// Header parameters cannot be arrays
	if paramType == "header" && field.IsArray {
		v.addError(fieldPath, "header parameters cannot be arrays")
	}

	// Cookie parameters cannot be arrays
	if paramType == "cookie" && field.IsArray {
		v.addError(fieldPath, "cookie parameters cannot be arrays")
	}

	// Query parameters can be arrays (this is allowed)

	// Run standard field validation
	v.validateField(path, field)
}

// validateEndpoint validates an endpoint
func (v *Validator) validateEndpoint(endpoint *resolver.ResolvedEndpoint, pkg *resolver.ResolvedPackage) {
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

	// Validate path parameters match path variables (including inline params)
	v.validatePathParametersWithInline(path, pathVars, endpoint.PathParams, endpoint.InlinePathParams)

	// Validate request body
	if endpoint.Request != nil {
		v.validateRequestBody(path, endpoint.Request, pkg.Schemas)
	}

	// Validate responses (including inline responses)
	hasResponses := len(endpoint.Responses) > 0 || len(endpoint.InlineResponses) > 0
	if !hasResponses {
		v.addError(path, "endpoint must have at least one response")
	}

	for statusCode, response := range endpoint.Responses {
		v.validateResponse(path, statusCode, response, pkg.Schemas)
	}
	// Inline responses don't need schema validation (they have inline fields)

	// Validate no parameter name conflicts
	v.validateParameterConflicts(path, endpoint)

	// Validate tags reference defined API-level tags
	if len(pkg.API.Tags) > 0 && len(endpoint.Tags) > 0 {
		v.validateEndpointTags(path, endpoint.Tags, pkg.API.Tags)
	}
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

// validatePathParameters validates that path parameters match path variables
func (v *Validator) validatePathParameters(path string, pathVars []string, params []*resolver.ResolvedParameter) {
	// Create map of path variables
	pathVarMap := make(map[string]bool)
	for _, varName := range pathVars {
		pathVarMap[varName] = true
	}

	// Collect all parameter field names
	paramFieldMap := make(map[string]bool)
	for _, param := range params {
		for _, field := range param.Fields {
			paramFieldMap[field.Name] = true
		}
	}

	// Check that all path variables have corresponding parameters
	for _, varName := range pathVars {
		if !paramFieldMap[varName] {
			v.addError(path, fmt.Sprintf("path variable {%s} has no corresponding @path parameter", varName))
		}
	}

	// Check that all path parameters are used in the path
	for paramName := range paramFieldMap {
		if !pathVarMap[paramName] {
			v.addError(path, fmt.Sprintf("@path parameter %s not used in path", paramName))
		}
	}
}

// validatePathParametersWithInline validates path parameters including inline declarations
func (v *Validator) validatePathParametersWithInline(path string, pathVars []string, params []*resolver.ResolvedParameter, inlineParams *resolver.ResolvedInlineParams) {
	// Create map of path variables
	pathVarMap := make(map[string]bool)
	for _, varName := range pathVars {
		pathVarMap[varName] = true
	}

	// Collect all parameter field names from both regular and inline params
	paramFieldMap := make(map[string]bool)
	for _, param := range params {
		for _, field := range param.Fields {
			paramFieldMap[field.Name] = true
		}
	}
	// Also collect from inline params
	if inlineParams != nil {
		for _, field := range inlineParams.Fields {
			paramFieldMap[field.Name] = true
		}
	}

	// Check that all path variables have corresponding parameters
	for _, varName := range pathVars {
		if !paramFieldMap[varName] {
			v.addError(path, fmt.Sprintf("path variable {%s} has no corresponding @path parameter", varName))
		}
	}

	// Check that all path parameters are used in the path
	for paramName := range paramFieldMap {
		if !pathVarMap[paramName] {
			v.addError(path, fmt.Sprintf("@path parameter %s not used in path", paramName))
		}
	}
}

// validateRequestBody validates a request body
func (v *Validator) validateRequestBody(path string, request *resolver.ResolvedRequestBody, schemas map[string]*resolver.ResolvedSchema) {
	if request.ContentType == "" {
		v.addError(path+".@request", "missing @contentType")
	}

	if request.Body == nil || request.Body.Schema == "" {
		v.addError(path+".@request", "missing @body")
		return
	}

	// Validate schema exists (use ElementType which extracts the base type from []T or map[string]T)
	schemaToCheck := request.Body.ElementType
	if schemaToCheck == "" {
		schemaToCheck = request.Body.Schema
	}
	if !isPrimitiveType(schemaToCheck) {
		if _, ok := schemas[schemaToCheck]; !ok {
			v.addError(path+".@request", fmt.Sprintf("references unknown schema: %s", schemaToCheck))
		}
	}
}

// validateResponse validates a response
func (v *Validator) validateResponse(path, statusCode string, response *resolver.ResolvedResponse, schemas map[string]*resolver.ResolvedSchema) {
	responsePath := fmt.Sprintf("%s.@response[%s]", path, statusCode)

	// Validate status code is numeric
	if !regexp.MustCompile(`^\d{3}$`).MatchString(statusCode) {
		v.addError(responsePath, fmt.Sprintf("invalid status code: %s", statusCode))
	}

	// Schema is optional for responses (e.g., 204 No Content)
	if response.Body != nil && response.Body.Schema != "" {
		// Validate schema exists (use ElementType which extracts the base type from []T or map[string]T)
		schemaToCheck := response.Body.ElementType
		if schemaToCheck == "" {
			schemaToCheck = response.Body.Schema
		}
		if !isPrimitiveType(schemaToCheck) {
			if _, ok := schemas[schemaToCheck]; !ok {
				v.addError(responsePath, fmt.Sprintf("references unknown schema: %s", schemaToCheck))
			}
		}
	}
}

// isPrimitiveType checks if a type name is a Go primitive (no schema lookup needed)
func isPrimitiveType(typeName string) bool {
	primitives := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "byte": true,
	}
	return primitives[typeName]
}

// validateParameterConflicts checks for parameter name conflicts across different parameter types
func (v *Validator) validateParameterConflicts(path string, endpoint *resolver.ResolvedEndpoint) {
	allParams := make(map[string]string) // name -> type

	// Collect all parameter names with their types
	for _, param := range endpoint.PathParams {
		for _, field := range param.Fields {
			if prevType, exists := allParams[field.Name]; exists {
				v.addError(path, fmt.Sprintf("parameter name conflict: %s appears in both %s and path parameters", field.Name, prevType))
			}
			allParams[field.Name] = "path"
		}
	}

	for _, param := range endpoint.QueryParams {
		for _, field := range param.Fields {
			if prevType, exists := allParams[field.Name]; exists {
				v.addError(path, fmt.Sprintf("parameter name conflict: %s appears in both %s and query parameters", field.Name, prevType))
			}
			allParams[field.Name] = "query"
		}
	}

	for _, param := range endpoint.HeaderParams {
		for _, field := range param.Fields {
			if prevType, exists := allParams[field.Name]; exists {
				v.addError(path, fmt.Sprintf("parameter name conflict: %s appears in both %s and header parameters", field.Name, prevType))
			}
			allParams[field.Name] = "header"
		}
	}

	for _, param := range endpoint.CookieParams {
		for _, field := range param.Fields {
			if prevType, exists := allParams[field.Name]; exists {
				v.addError(path, fmt.Sprintf("parameter name conflict: %s appears in both %s and cookie parameters", field.Name, prevType))
			}
			allParams[field.Name] = "cookie"
		}
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
