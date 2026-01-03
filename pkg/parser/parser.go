package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/schema"
)

// Parser orchestrates the parsing of a Go package into a ParsedPackage
type Parser struct {
	packagePath string
	comments    *PackageComments
}

// NewParser creates a new parser for the given package path
func NewParser(packagePath string) *Parser {
	return &Parser{
		packagePath: packagePath,
	}
}

// Comments returns the extracted package comments
// This should be called after Parse() to access inline declarations and the loaded package
func (p *Parser) Comments() *PackageComments {
	return p.comments
}

// Parse parses the package and returns a ParsedPackage
func (p *Parser) Parse() (*ParsedPackage, error) {
	// Step 1: Extract comments from AST
	comments, err := ExtractComments(p.packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract comments: %w", err)
	}
	p.comments = comments

	result := &ParsedPackage{
		PackageName: comments.Name,
		Schemas:     make(map[string]*Schema),
		Parameters:  make(map[string]*Parameter),
		Endpoints:   make([]*Endpoint, 0),
	}

	// Step 2: Parse @api annotation
	if err := p.parseAPI(result); err != nil {
		return nil, fmt.Errorf("failed to parse @api: %w", err)
	}

	// Step 3: Parse @schema annotations
	if err := p.parseSchemas(result); err != nil {
		return nil, fmt.Errorf("failed to parse schemas: %w", err)
	}

	// Step 4: Parse parameter structs (@path, @query, @header, @cookie)
	if err := p.parseParameters(result); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Step 5: Parse @endpoint annotations
	if err := p.parseEndpoints(result); err != nil {
		return nil, fmt.Errorf("failed to parse endpoints: %w", err)
	}

	return result, nil
}

// parseAPI parses the @api annotation from package-level comments
func (p *Parser) parseAPI(result *ParsedPackage) error {
	if p.comments.PackageComments == nil {
		return fmt.Errorf("no package-level comments found (missing @api annotation)")
	}

	lines := p.comments.PackageComments.GetAnnotationLines()
	if len(lines) == 0 {
		return fmt.Errorf("no annotations found in package comments")
	}

	// Get @api schema node
	apiNode := schema.AnnotationSchema.GetChild("@api")
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
func (p *Parser) parseSchemas(result *ParsedPackage) error {
	// First pass: parse @schema annotated structs
	for structName, commentBlock := range p.comments.StructComments {
		if !commentBlock.HasAnnotation("@schema") {
			continue
		}

		lines := commentBlock.GetAnnotationLines()
		schemaNode := schema.AnnotationSchema.GetChild("@schema")

		parsed, err := ParseAnnotationBlock(lines, "@schema", schemaNode)
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

		// Parse fields
		if fieldComments, ok := p.comments.FieldComments[structName]; ok {
			for fieldName, fieldComment := range fieldComments {
				if !fieldComment.HasAnnotation("@field") {
					continue
				}

				fieldLines := fieldComment.GetAnnotationLines()
				fieldNode := schema.AnnotationSchema.GetChild("@field")

				// Check if inline format
				if IsInlineFormat(fieldLines) {
					parsedField, err := ParseInlineAnnotation(fieldLines[0], "@field", fieldNode)
					if err != nil {
						return fmt.Errorf("failed to parse inline @field for %s.%s: %w", structName, fieldName, err)
					}

					field := p.convertParsedField(fieldName, parsedField)
					s.Fields = append(s.Fields, field)
				} else {
					parsedField, err := ParseAnnotationBlock(fieldLines, "@field", fieldNode)
					if err != nil {
						return fmt.Errorf("failed to parse @field for %s.%s: %w", structName, fieldName, err)
					}

					field := p.convertParsedField(fieldName, parsedField)
					s.Fields = append(s.Fields, field)
				}
			}
		}

		// Store schema with metadata if present
		if parsed.HasChild("@description") {
			s.Description = parsed.GetChildValue("@description")
		}
		if parsed.HasChild("@deprecated") {
			s.Deprecated = true
		}

		result.Schemas[structName] = s
	}

	// Second pass: detect type aliases that instantiate generic schemas
	// These don't need @schema annotation, they're detected from type alias syntax
	for typeName, typeInfo := range p.comments.TypeInfo {
		// Skip if already processed as @schema
		if _, exists := result.Schemas[typeName]; exists {
			continue
		}

		// Only process type aliases that reference a generic type
		if !typeInfo.IsTypeAlias || typeInfo.AliasOf == "" {
			continue
		}

		// Check if the base type is a generic schema
		baseType := extractBaseType(typeInfo.AliasOf)
		if baseSchema, ok := result.Schemas[baseType]; ok && baseSchema.IsGeneric {
			// Create a schema for this type alias
			s := &Schema{
				Name:        typeName,
				GoTypeName:  typeName,
				IsTypeAlias: true,
				AliasOf:     typeInfo.AliasOf,
				Fields:      make([]*Field, 0),
			}

			// Copy fields from the generic base schema
			// The type parameter will be resolved later
			for _, field := range baseSchema.Fields {
				fieldCopy := *field
				s.Fields = append(s.Fields, &fieldCopy)
			}

			// Inherit description from base if available
			s.Description = baseSchema.Description

			result.Schemas[typeName] = s
		}
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
func (p *Parser) parseParameters(result *ParsedPackage) error {
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

		param := &Parameter{
			Name:       structName,
			Type:       paramType,
			GoTypeName: structName,
			Fields:     make([]*Field, 0),
		}

		// Parse fields
		if fieldComments, ok := p.comments.FieldComments[structName]; ok {
			for fieldName, fieldComment := range fieldComments {
				if !fieldComment.HasAnnotation("@field") {
					continue
				}

				fieldLines := fieldComment.GetAnnotationLines()
				fieldNode := schema.AnnotationSchema.GetChild("@field")

				// Check if inline format
				if IsInlineFormat(fieldLines) {
					parsedField, err := ParseInlineAnnotation(fieldLines[0], "@field", fieldNode)
					if err != nil {
						return fmt.Errorf("failed to parse inline @field for %s.%s: %w", structName, fieldName, err)
					}

					field := p.convertParsedField(fieldName, parsedField)
					param.Fields = append(param.Fields, field)
				} else {
					parsedField, err := ParseAnnotationBlock(fieldLines, "@field", fieldNode)
					if err != nil {
						return fmt.Errorf("failed to parse @field for %s.%s: %w", structName, fieldName, err)
					}

					field := p.convertParsedField(fieldName, parsedField)
					param.Fields = append(param.Fields, field)
				}
			}
		}

		result.Parameters[structName] = param
	}

	return nil
}

// parseEndpoints parses all @endpoint annotated functions
func (p *Parser) parseEndpoints(result *ParsedPackage) error {
	for funcName, commentBlock := range p.comments.FunctionComments {
		if !commentBlock.HasAnnotation("@endpoint") {
			continue
		}

		lines := commentBlock.GetAnnotationLines()
		endpointNode := schema.AnnotationSchema.GetChild("@endpoint")

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
func (p *Parser) convertParsedField(fieldName string, parsed *ParsedAnnotation) *Field {
	field := &Field{
		GoName:      fieldName,
		Name:        fieldName, // Will be resolved from struct tags later
		Description: parsed.GetChildValue("@description"),
		Format:      parsed.GetChildValue("@format"),
		Example:     parsed.GetChildValue("@example"),
		Default:     parsed.GetChildValue("@default"),
		Pattern:     parsed.GetChildValue("@pattern"),
		Deprecated:  parsed.HasChild("@deprecated"),
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
		}
	}

	if max := parsed.GetChildValue("@maximum"); max != "" {
		if val, err := strconv.ParseFloat(max, 64); err == nil {
			field.Maximum = &val
		}
	}

	if minLen := parsed.GetChildValue("@minLength"); minLen != "" {
		if val, err := strconv.Atoi(minLen); err == nil {
			field.MinLength = &val
		}
	}

	if maxLen := parsed.GetChildValue("@maxLength"); maxLen != "" {
		if val, err := strconv.Atoi(maxLen); err == nil {
			field.MaxLength = &val
		}
	}

	if minItems := parsed.GetChildValue("@minItems"); minItems != "" {
		if val, err := strconv.Atoi(minItems); err == nil {
			field.MinItems = &val
		}
	}

	if maxItems := parsed.GetChildValue("@maxItems"); maxItems != "" {
		if val, err := strconv.Atoi(maxItems); err == nil {
			field.MaxItems = &val
		}
	}

	if parsed.HasChild("@uniqueItems") {
		field.UniqueItems = true
	}

	return field
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
