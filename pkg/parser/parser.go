package parser

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/annotation"
)

// Parse reads every annotation in a Go package and returns the result.
//
// The returned *Package is the parser's entire output: no comment side channel,
// no second pass. It carries the loaded *packages.Package too, so the resolver
// works from the same type identities the parser saw rather than loading the
// package a second time.
func Parse(packagePath string) (*Package, error) {
	comments, err := ExtractComments(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract comments: %w", err)
	}

	p := &parser{comments: comments}
	return p.parse()
}

// parser holds the comment scan while the annotation passes run over it.
type parser struct {
	comments *PackageComments
}

func (p *parser) parse() (*Package, error) {
	comments := p.comments

	result := &Package{
		PackageName: comments.Name,
		Pkg:         comments.Pkg,
		FuncInlines: comments.FuncInlines,
		Schemas:     make(map[string]*Schema),
		Parameters:  make(map[string]*Parameter),
		Endpoints:   make([]*Endpoint, 0),
	}

	if err := p.parseAPI(result); err != nil {
		return nil, fmt.Errorf("failed to parse @api: %w", err)
	}

	if err := p.parseSchemas(result); err != nil {
		return nil, fmt.Errorf("failed to parse schemas: %w", err)
	}

	// Parameter structs: @path, @query, @header, @cookie
	if err := p.parseParameters(result); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	if err := p.parseEndpoints(result); err != nil {
		return nil, fmt.Errorf("failed to parse endpoints: %w", err)
	}

	// @field annotations on every struct that has them. Runs after the passes
	// that discover those structs.
	if err := p.parseStructFields(result); err != nil {
		return nil, fmt.Errorf("failed to parse struct fields: %w", err)
	}

	// Type-alias schemas (generic instantiations). Runs after parseStructFields
	// so aliases copy already-populated Fields from their base.
	if err := p.resolveSchemaAliases(result); err != nil {
		return nil, fmt.Errorf("failed to resolve schema aliases: %w", err)
	}

	return result, nil
}

// parseStructFields is the single entry point for @field parsing. It visits
// every source of struct field comments — @schema structs, parameter structs,
// and inline var structs declared in handler bodies — and populates their
// Fields via parseFieldComments. There is no separate "inline field parsing";
// discovery differs by source, but the parsing itself is identical for all.
//
// Maps are visited in sorted order so that when several structs have bad
// annotations, the same one is reported every run.
func (p *parser) parseStructFields(result *Package) error {
	// Every struct type in the package, not only the annotated ones. An
	// embedded struct contributes its fields to whatever embeds it, and it
	// carries its own @field annotations along with them, whether or not it is
	// a @schema in its own right.
	result.StructFields = make(map[string][]*Field, len(p.comments.FieldComments))
	for _, structName := range slices.Sorted(maps.Keys(p.comments.FieldComments)) {
		fields, err := p.parseFieldComments(p.comments.FieldComments[structName], structName)
		if err != nil {
			return err
		}
		result.StructFields[structName] = fields
	}

	for name, schema := range result.Schemas {
		schema.Fields = result.StructFields[name]
	}
	for name, param := range result.Parameters {
		param.Fields = result.StructFields[name]
	}

	// Inline var structs declared in function bodies
	for _, funcName := range slices.Sorted(maps.Keys(p.comments.FuncInlines)) {
		inlines := p.comments.FuncInlines[funcName]
		if inlines == nil {
			continue
		}

		var structs []*InlineStructInfo
		structs = append(structs, inlines.Query...)
		structs = append(structs, inlines.Path...)
		structs = append(structs, inlines.Header...)
		structs = append(structs, inlines.Cookie...)
		if inlines.Request != nil {
			structs = append(structs, inlines.Request)
		}
		for _, status := range slices.Sorted(maps.Keys(inlines.Responses)) {
			structs = append(structs, inlines.Responses[status])
		}

		for _, info := range structs {
			fields, err := p.parseFieldComments(info.FieldComments, funcName+"."+info.VarName)
			if err != nil {
				return err
			}
			info.Fields = fields
		}
	}

	return nil
}

// parseFieldComments parses @field annotations from a map of per-field comment
// blocks into *Field values. Single source of truth for @field parsing, used by
// @schema structs, parameter structs, and in-function var structs alike.
//
// Field order here does not reach the output — the resolver walks the Go struct
// in declaration order and looks each field's annotation up by name. Sorting is
// for the error path: without it, which of several bad fields gets reported
// would vary run to run.
func (p *parser) parseFieldComments(fieldComments map[string]*FieldComments, context string) ([]*Field, error) {
	var fields []*Field

	for _, fieldName := range slices.Sorted(maps.Keys(fieldComments)) {
		harvested := fieldComments[fieldName]
		if harvested == nil {
			continue
		}

		// A field's own annotation, and the annotations on the fields of its
		// type when that type is an anonymous struct. Either may be absent: a
		// field can carry an @field with no nested struct, or an inline struct
		// whose own field is unannotated.
		nested, err := p.parseFieldComments(harvested.Fields, context+"."+fieldName)
		if err != nil {
			return nil, err
		}

		field, err := parseFieldAnnotation(fieldName, harvested.Comment, context)
		if err != nil {
			return nil, err
		}

		switch {
		case field != nil:
			field.Fields = nested
		case len(nested) > 0:
			// No @field of its own, but something below it is annotated, so
			// the field has to exist to carry them down.
			field = &Field{GoName: fieldName, Name: fieldName, Fields: nested}
		default:
			continue
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// parseFieldAnnotation parses one field's @field comment, in either the inline
// or the block form. Returns nil when the comment carries no @field at all.
func parseFieldAnnotation(fieldName string, comment *CommentBlock, context string) (*Field, error) {
	if comment == nil || !comment.HasAnnotation("@field") {
		return nil, nil
	}

	lines := comment.GetAnnotationLines()
	fieldNode := annotation.Schema.GetChild("@field")

	var parsed *ParsedAnnotation
	var err error
	if IsInlineFormat(lines) {
		parsed, err = ParseInlineAnnotation(lines[0], "@field", fieldNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse inline @field for %s.%s: %w", context, fieldName, err)
		}
	} else {
		parsed, err = ParseAnnotationBlock(lines, "@field", fieldNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse @field for %s.%s: %w", context, fieldName, err)
		}
	}

	field, err := convertParsedField(fieldName, parsed)
	if err != nil {
		return nil, fmt.Errorf("invalid @field for %s.%s: %w", context, fieldName, err)
	}
	return field, nil
}

// parseAPI parses the @api annotation from package-level comments
func (p *parser) parseAPI(result *Package) error {
	if p.comments.PackageComments == nil {
		return fmt.Errorf("no package-level comments found (missing @api annotation)")
	}

	lines := p.comments.PackageComments.GetAnnotationLines()
	if len(lines) == 0 {
		return fmt.Errorf("no annotations found in package comments")
	}

	// Get @api schema node
	apiNode := annotation.Schema.GetChild("@api")
	if apiNode == nil {
		return fmt.Errorf("@api schema node not found")
	}

	// Parse @api annotation
	parsed, err := ParseAnnotationBlock(lines, "@api", apiNode)
	if err != nil {
		return err
	}

	// Convert to APIInfo
	api := &APIInfo{
		Servers:         make([]*Server, 0),
		SecuritySchemes: make(map[string]*SecurityScheme),
		Security:        make([][]*SecurityRequirement, 0),
	}

	// Required fields
	api.Title = parsed.GetChildValue("@title")
	api.Version = parsed.GetChildValue("@version")

	if api.Title == "" {
		return fmt.Errorf("@api missing required @title")
	}
	if api.Version == "" {
		return fmt.Errorf("@api missing required @version")
	}

	// Optional fields
	api.Description = parsed.GetChildValue("@description")
	api.TermsOfService = parsed.GetChildValue("@termsOfService")
	api.DefaultContentType = ExpandContentType(parsed.GetChildValue("@defaultContentType"))

	// Contact
	if contact := parsed.Children["@contact"]; contact != nil {
		api.Contact = &Contact{
			Name:  contact.GetChildValue("@name"),
			Email: contact.GetChildValue("@email"),
			URL:   contact.GetChildValue("@url"),
		}
	}

	// License
	if license := parsed.Children["@license"]; license != nil {
		api.License = &License{
			Name: license.GetChildValue("@name"),
			URL:  license.GetChildValue("@url"),
		}
	}

	// Servers
	for _, serverParsed := range parsed.GetRepeatedChildren("@server") {
		server := &Server{
			URL:         serverParsed.Metadata,
			Description: serverParsed.GetChildValue("@description"),
		}
		api.Servers = append(api.Servers, server)
	}

	// Security Schemes
	for _, schemeParsed := range parsed.GetRepeatedChildren("@securityScheme") {
		scheme := &SecurityScheme{
			Name:          schemeParsed.Metadata,
			Type:          schemeParsed.GetChildValue("@type"),
			Scheme:        schemeParsed.GetChildValue("@scheme"),
			BearerFormat:  schemeParsed.GetChildValue("@bearerFormat"),
			In:            schemeParsed.GetChildValue("@in"),
			ParameterName: schemeParsed.GetChildValue("@name"),
			Description:   schemeParsed.GetChildValue("@description"),
		}
		api.SecuritySchemes[scheme.Name] = scheme
	}

	// Security requirements
	for _, securityParsed := range parsed.GetRepeatedChildren("@security") {
		// Each @security block is an OR group
		var requirements []*SecurityRequirement

		for _, withParsed := range securityParsed.GetRepeatedChildren("@with") {
			req := &SecurityRequirement{
				SchemeName: withParsed.Metadata,
				Scopes:     make([]string, 0),
			}

			// Get scopes
			for _, scopeParsed := range withParsed.GetRepeatedChildren("@scope") {
				req.Scopes = append(req.Scopes, scopeParsed.Value)
			}

			requirements = append(requirements, req)
		}

		api.Security = append(api.Security, requirements)
	}

	// Tags
	for _, tagParsed := range parsed.GetRepeatedChildren("@tag") {
		tag := &Tag{
			Name:        tagParsed.Metadata,
			Description: tagParsed.GetChildValue("@description"),
		}
		api.Tags = append(api.Tags, tag)
	}

	result.API = api
	return nil
}

// parseSchemas parses all @schema annotated structs
func (p *parser) parseSchemas(result *Package) error {
	// First pass: parse @schema annotated structs
	for structName, commentBlock := range p.comments.StructComments {
		if !commentBlock.HasAnnotation("@schema") {
			continue
		}

		lines := commentBlock.GetAnnotationLines()
		def := annotation.Schema.GetChild("@schema")

		parsed, err := ParseAnnotationBlock(lines, "@schema", def)
		if err != nil {
			return fmt.Errorf("failed to parse @schema for %s: %w", structName, err)
		}

		s := &Schema{
			Name:       structName,
			GoTypeName: structName,
			Fields:     make([]*Field, 0),
		}

		// Populate generics info from TypeInfo
		if typeInfo, ok := p.comments.TypeInfo[structName]; ok {
			s.IsGeneric = typeInfo.IsGeneric
			s.IsTypeAlias = typeInfo.IsTypeAlias
			s.AliasOf = typeInfo.AliasOf
		}

		// Struct-level metadata only. Field parsing is handled by parseStructFields
		// so @schema and inline var structs share one parsing path.
		if parsed.HasChild("@description") {
			s.Description = parsed.GetChildValue("@description")
		}
		if parsed.HasChild("@deprecated") {
			s.Deprecated = true
		}

		result.Schemas[structName] = s
	}

	return nil
}

// resolveSchemaAliases detects type aliases that instantiate generic schemas
// (e.g. `type UserResponse = DataResponse[User]`) and creates derived Schema
// entries by copying fields from the base. Runs after parseStructFields so the
// base schema's Fields are already populated.
func (p *parser) resolveSchemaAliases(result *Package) error {
	for typeName, typeInfo := range p.comments.TypeInfo {
		if _, exists := result.Schemas[typeName]; exists {
			continue
		}
		if !typeInfo.IsTypeAlias || typeInfo.AliasOf == "" {
			continue
		}

		baseType := extractBaseType(typeInfo.AliasOf)
		baseSchema, ok := result.Schemas[baseType]
		if !ok || !baseSchema.IsGeneric {
			continue
		}

		s := &Schema{
			Name:        typeName,
			GoTypeName:  typeName,
			IsTypeAlias: true,
			AliasOf:     typeInfo.AliasOf,
			Fields:      make([]*Field, 0),
			Description: baseSchema.Description,
		}
		for _, field := range baseSchema.Fields {
			fieldCopy := *field
			s.Fields = append(s.Fields, &fieldCopy)
		}
		result.Schemas[typeName] = s
	}
	return nil
}

// extractBaseType extracts the base type name from a generic instantiation
// e.g., "DataResponse[User]" -> "DataResponse"
func extractBaseType(typeName string) string {
	if idx := strings.Index(typeName, "["); idx > 0 {
		return typeName[:idx]
	}
	return typeName
}

// parseParameters parses all parameter structs (@path, @query, @header, @cookie)
func (p *parser) parseParameters(result *Package) error {
	for structName, commentBlock := range p.comments.StructComments {
		var paramType ParameterType

		// Determine parameter type
		if commentBlock.HasAnnotation("@path") {
			paramType = PathParameter
		} else if commentBlock.HasAnnotation("@query") {
			paramType = QueryParameter
		} else if commentBlock.HasAnnotation("@header") {
			paramType = HeaderParameter
		} else if commentBlock.HasAnnotation("@cookie") {
			paramType = CookieParameter
		} else {
			continue
		}

		// Fields are filled in by parseStructFields, which parses every
		// struct's annotations once.
		result.Parameters[structName] = &Parameter{
			Name:       structName,
			Type:       paramType,
			GoTypeName: structName,
		}
	}

	return nil
}

// parseEndpoints parses all @endpoint annotated functions
func (p *parser) parseEndpoints(result *Package) error {
	for funcName, commentBlock := range p.comments.FunctionComments {
		if !commentBlock.HasAnnotation("@endpoint") {
			continue
		}

		lines := commentBlock.GetAnnotationLines()
		endpointNode := annotation.Schema.GetChild("@endpoint")

		parsed, err := ParseAnnotationBlock(lines, "@endpoint", endpointNode)
		if err != nil {
			return fmt.Errorf("failed to parse @endpoint for %s: %w", funcName, err)
		}

		// Extract method and path from metadata
		metadata := parsed.Metadata
		parts := strings.Fields(metadata)
		if len(parts) < 2 {
			return fmt.Errorf("@endpoint for %s missing method and path: %s", funcName, metadata)
		}

		endpoint := &Endpoint{
			FuncName:     funcName,
			Method:       parts[0],
			Path:         parts[1],
			OperationID:  parsed.GetChildValue("@operationID"),
			Summary:      parsed.GetChildValue("@summary"),
			Description:  parsed.GetChildValue("@description"),
			Auth:         parsed.GetChildValue("@auth"),
			PathParams:   extractRepeatedReferences(parsed, "@path"),
			QueryParams:  extractRepeatedReferences(parsed, "@query"),
			HeaderParams: extractRepeatedReferences(parsed, "@header"),
			CookieParams: extractRepeatedReferences(parsed, "@cookie"),
			Responses:    make(map[string]*Response),
		}

		// Parse tags (multiple annotations)
		endpoint.Tags = extractRepeatedReferences(parsed, "@tag")

		// Parse deprecated flag
		if parsed.HasChild("@deprecated") {
			endpoint.Deprecated = true
		}

		// Parse request
		if request := parsed.Children["@request"]; request != nil {
			endpoint.Request = &RequestBody{
				ContentType: ExpandContentType(request.GetChildValue("@contentType")),
				Body:        parseBody(request),
			}
		}

		// Parse responses
		for _, responseParsed := range parsed.GetRepeatedChildren("@response") {
			statusCode := responseParsed.Metadata
			resp := &Response{
				StatusCode:   statusCode,
				ContentType:  ExpandContentType(responseParsed.GetChildValue("@contentType")),
				Body:         parseBody(responseParsed),
				Description:  responseParsed.GetChildValue("@description"),
				HeaderParams: extractRepeatedReferences(responseParsed, "@header"),
			}
			endpoint.Responses[statusCode] = resp
		}

		result.Endpoints = append(result.Endpoints, endpoint)
	}

	return nil
}

// convertParsedField converts a ParsedAnnotation to a Field
// convertParsedField needs no parser state; it is a pure translation from a
// parsed annotation tree to a Field.
func convertParsedField(fieldName string, parsed *ParsedAnnotation) (*Field, error) {
	field := &Field{
		GoName:      fieldName,
		Name:        fieldName, // Will be resolved from struct tags later
		Description: parsed.GetChildValue("@description"),
		Format:      parsed.GetChildValue("@format"),
		Example:     parsed.GetChildValue("@example"),
		Default:     parsed.GetChildValue("@default"),
		Pattern:     parsed.GetChildValue("@pattern"),
		Deprecated:  parsed.HasChild("@deprecated"),
		ReadOnly:    parsed.HasChild("@readOnly"),
		WriteOnly:   parsed.HasChild("@writeOnly"),
	}

	// Parse enum (comma-separated)
	if enum := parsed.GetChildValue("@enum"); enum != "" {
		field.Enum = strings.Split(enum, ",")
		for i := range field.Enum {
			field.Enum[i] = strings.TrimSpace(field.Enum[i])
		}
	}

	// Parse numeric fields
	if min := parsed.GetChildValue("@minimum"); min != "" {
		if val, err := strconv.ParseFloat(min, 64); err == nil {
			field.Minimum = &val
		} else {
			return nil, fmt.Errorf("@minimum value %q is not a valid number", min)
		}
	}

	if max := parsed.GetChildValue("@maximum"); max != "" {
		if val, err := strconv.ParseFloat(max, 64); err == nil {
			field.Maximum = &val
		} else {
			return nil, fmt.Errorf("@maximum value %q is not a valid number", max)
		}
	}

	if exMin := parsed.GetChildValue("@exclusiveMinimum"); exMin != "" {
		if val, err := strconv.ParseFloat(exMin, 64); err == nil {
			field.ExclusiveMinimum = &val
		} else {
			return nil, fmt.Errorf("@exclusiveMinimum value %q is not a valid number", exMin)
		}
	}

	if exMax := parsed.GetChildValue("@exclusiveMaximum"); exMax != "" {
		if val, err := strconv.ParseFloat(exMax, 64); err == nil {
			field.ExclusiveMaximum = &val
		} else {
			return nil, fmt.Errorf("@exclusiveMaximum value %q is not a valid number", exMax)
		}
	}

	if minLen := parsed.GetChildValue("@minLength"); minLen != "" {
		if val, err := strconv.Atoi(minLen); err == nil {
			field.MinLength = &val
		} else {
			return nil, fmt.Errorf("@minLength value %q is not a valid integer", minLen)
		}
	}

	if maxLen := parsed.GetChildValue("@maxLength"); maxLen != "" {
		if val, err := strconv.Atoi(maxLen); err == nil {
			field.MaxLength = &val
		} else {
			return nil, fmt.Errorf("@maxLength value %q is not a valid integer", maxLen)
		}
	}

	if minItems := parsed.GetChildValue("@minItems"); minItems != "" {
		if val, err := strconv.Atoi(minItems); err == nil {
			field.MinItems = &val
		} else {
			return nil, fmt.Errorf("@minItems value %q is not a valid integer", minItems)
		}
	}

	if maxItems := parsed.GetChildValue("@maxItems"); maxItems != "" {
		if val, err := strconv.Atoi(maxItems); err == nil {
			field.MaxItems = &val
		} else {
			return nil, fmt.Errorf("@maxItems value %q is not a valid integer", maxItems)
		}
	}

	if parsed.HasChild("@uniqueItems") {
		field.UniqueItems = true
	}

	if req := parsed.GetChildValue("@required"); req != "" {
		val, err := strconv.ParseBool(req)
		if err != nil {
			return nil, fmt.Errorf("@required value %q is not a valid boolean (use true or false)", req)
		}
		field.Required = &val
	}

	if null := parsed.GetChildValue("@nullable"); null != "" {
		val, err := strconv.ParseBool(null)
		if err != nil {
			return nil, fmt.Errorf("@nullable value %q is not a valid boolean (use true or false)", null)
		}
		field.Nullable = &val
	}

	return field, nil
}

// extractRepeatedReferences extracts references from repeated children annotations
func extractRepeatedReferences(parsed *ParsedAnnotation, name string) []string {
	result := make([]string, 0)
	for _, child := range parsed.GetRepeatedChildren(name) {
		// For reference annotations, the value is in child.Value
		if child.Value != "" {
			result = append(result, child.Value)
		}
	}
	return result
}

// ExpandContentType expands short content-type names to full MIME types
func ExpandContentType(shortName string) string {
	// If already a full MIME type, return as-is
	if strings.Contains(shortName, "/") {
		return shortName
	}

	// Map of short names to MIME types
	// Note: Only json struct tags are currently supported for schema field names
	// xml/csv struct tag support may be added in future versions
	contentTypeMap := map[string]string{
		"json":      "application/json",
		"xml":       "application/xml",
		"form":      "application/x-www-form-urlencoded",
		"multipart": "multipart/form-data",
		"text":      "text/plain",
		"csv":       "text/csv",
		"binary":    "application/octet-stream",
		"html":      "text/html",
		"empty":     "", // No content
	}

	if mimeType, ok := contentTypeMap[strings.ToLower(shortName)]; ok {
		return mimeType
	}

	// If not found in map, return as-is (could be custom type)
	return shortName
}

// parseBody parses @body annotation from a request/response block
// Returns nil if no body is defined
// New syntax: @body User @bind DataResponse.Data
// The @body value is the schema (User, []User, map[string]User)
// The @bind is optional and specifies the wrapper (Wrapper.Field format)
func parseBody(parsed *ParsedAnnotation) *Body {
	// Check for @body annotation
	if bodyParsed := parsed.Children["@body"]; bodyParsed != nil {
		body := &Body{
			Schema: bodyParsed.Metadata, // Schema name is in metadata (e.g., "User", "[]User")
		}

		// Parse @bind annotation (sibling of @body in the response/request block)
		// New syntax: @bind DataResponse.Data
		if bindParsed := parsed.Children["@bind"]; bindParsed != nil {
			body.Bind = ParseBindTarget(bindParsed.Value)
		}

		return body
	}

	// Fallback to @schema annotation (legacy style, but keeping for simplicity)
	if schema := parsed.GetChildValue("@schema"); schema != "" {
		return &Body{
			Schema: schema,
			Bind:   nil,
		}
	}

	return nil
}

// ParseBindTarget parses a @bind value into a BindTarget
// Format: "Wrapper.Field" (e.g., "DataResponse.Data")
func ParseBindTarget(value string) *BindTarget {
	parts := strings.SplitN(strings.TrimSpace(value), ".", 2)
	if len(parts) != 2 {
		return nil
	}

	return &BindTarget{
		Wrapper: strings.TrimSpace(parts[0]),
		Field:   strings.TrimSpace(parts[1]),
	}
}
