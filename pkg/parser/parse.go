package parser

import (
	"errors"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

// This file turns harvested comments into the IR, guided by the grammar.
//
// Errors accumulate per top-level item — per schema, per parameter struct, per
// endpoint — so one broken annotation does not hide the rest of the package.
// Within a single item the first error wins, because everything after it is
// usually a cascade of the same mistake.

// StatusCodePattern matches a response status code: a three-digit code (404)
// or a range (4XX). The literal "default" is also a valid status but is not
// matched here; ValidStatusCode covers both.
var StatusCodePattern = regexp.MustCompile(`^[1-5](\d{2}|XX)$`)

// ValidStatusCode reports whether status is a usable OpenAPI response key.
func ValidStatusCode(status string) bool {
	return status == "default" || StatusCodePattern.MatchString(status)
}

// Parse loads the Go package at dir and parses every annotation in it.
func Parse(dir string) (*Package, error) {
	pkg, err := load(dir)
	if err != nil {
		return nil, err
	}

	src, err := harvest(pkg)
	if err != nil {
		return nil, err
	}

	out := &Package{Name: src.name, GoPkg: pkg}
	var errs []error

	api, err := parseAPI(src.api)
	if err != nil {
		errs = append(errs, err)
	} else {
		out.API = api
	}

	for _, td := range src.types {
		if !annotatable(td) || !td.Doc.HasAnnotation("@schema") {
			continue
		}
		schema, err := parseSchema(td)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out.Schemas = append(out.Schemas, schema)
	}

	// Alias schemas come after the declared ones, because they copy fields
	// from the generic schema they instantiate.
	out.Schemas = append(out.Schemas, aliasSchemas(src.types, out.Schemas)...)

	for _, td := range src.types {
		if !annotatable(td) {
			continue
		}
		kind := parameterKind(td.Doc)
		if kind == "" {
			continue
		}
		param, err := parseParameterStruct(td, kind)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out.Parameters = append(out.Parameters, param)
	}

	for _, fn := range src.funcs {
		if !fn.Doc.HasAnnotation("@endpoint") {
			continue
		}
		endpoint, err := parseEndpoint(fn)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out.Endpoints = append(out.Endpoints, endpoint)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

// annotatable reports whether a type declaration can carry a @schema or
// parameter marker: struct declarations and type aliases can, other named
// types (type Status string) cannot.
func annotatable(td *typeDecl) bool {
	return td.IsStruct || td.IsTypeAlias
}

// parameterKind returns the parameter kind a struct's comment marks it as, or
// "" when it carries no parameter marker.
func parameterKind(doc *CommentBlock) string {
	for _, kind := range []string{ParamPath, ParamQuery, ParamHeader, ParamCookie} {
		if doc.HasAnnotation("@" + kind) {
			return kind
		}
	}
	return ""
}

// parseAPI parses the @api block from the package-level comments.
func parseAPI(doc *CommentBlock) (*APIInfo, error) {
	if doc == nil {
		return nil, errorf(token.Position{}, "no package-level comments found (missing @api annotation)")
	}

	lines := doc.AnnotationLines()
	if len(lines) == 0 {
		return nil, errorf(doc.Pos, "no annotations found in package comments")
	}

	parsed, err := ParseAnnotationBlock(lines, "@api", apiNode)
	if err != nil {
		return nil, wrapf(doc.Pos, err, "failed to parse @api")
	}

	api := &APIInfo{
		Title:              parsed.ChildValue("@title"),
		Version:            parsed.ChildValue("@version"),
		Description:        parsed.ChildValue("@description"),
		TermsOfService:     parsed.ChildValue("@termsOfService"),
		DefaultContentType: ExpandContentType(parsed.ChildValue("@defaultContentType")),
	}

	if api.Title == "" {
		return nil, errorf(parsed.Pos, "@api missing required @title")
	}
	if api.Version == "" {
		return nil, errorf(parsed.Pos, "@api missing required @version")
	}

	if contact := parsed.Child("@contact"); contact != nil {
		api.Contact = &Contact{
			Name:  contact.ChildValue("@name"),
			Email: contact.ChildValue("@email"),
			URL:   contact.ChildValue("@url"),
		}
	}

	if license := parsed.Child("@license"); license != nil {
		api.License = &License{
			Name: license.ChildValue("@name"),
			URL:  license.ChildValue("@url"),
		}
	}

	for _, server := range parsed.RepeatedChildren("@server") {
		api.Servers = append(api.Servers, &Server{
			URL:         server.Metadata,
			Description: server.ChildValue("@description"),
		})
	}

	for _, scheme := range parsed.RepeatedChildren("@securityScheme") {
		api.SecuritySchemes = append(api.SecuritySchemes, &SecurityScheme{
			Name:          scheme.Metadata,
			Type:          scheme.ChildValue("@type"),
			Scheme:        scheme.ChildValue("@scheme"),
			BearerFormat:  scheme.ChildValue("@bearerFormat"),
			In:            scheme.ChildValue("@in"),
			ParameterName: scheme.ChildValue("@name"),
			Description:   scheme.ChildValue("@description"),
		})
	}

	// Each @security block is one group of requirements that must all hold;
	// several blocks are alternatives.
	for _, security := range parsed.RepeatedChildren("@security") {
		var group []*SecurityRequirement
		for _, with := range security.RepeatedChildren("@with") {
			req := &SecurityRequirement{SchemeName: with.Metadata, Scopes: []string{}}
			for _, scope := range with.RepeatedChildren("@scope") {
				req.Scopes = append(req.Scopes, scope.Value)
			}
			group = append(group, req)
		}
		api.Security = append(api.Security, group)
	}

	for _, tag := range parsed.RepeatedChildren("@tag") {
		api.Tags = append(api.Tags, &Tag{
			Name:        tag.Metadata,
			Description: tag.ChildValue("@description"),
		})
	}

	return api, nil
}

// parseSchema parses a @schema annotated type declaration.
func parseSchema(td *typeDecl) (*Schema, error) {
	parsed, err := ParseAnnotationBlock(td.Doc.AnnotationLines(), "@schema", schemaNode)
	if err != nil {
		return nil, wrapf(td.Pos, err, "failed to parse @schema for %s", td.Name)
	}

	schema := &Schema{
		Name:        td.Name,
		Pos:         td.Pos,
		Description: parsed.ChildValue("@description"),
		Deprecated:  parsed.HasChild("@deprecated"),
		IsGeneric:   td.IsGeneric,
		IsTypeAlias: td.IsTypeAlias,
		AliasOf:     td.AliasOf,
		TypeArgs:    typeArgs(td.AliasOf),
	}

	schema.Fields, err = parseFields(td.Fields, td.Name)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

// aliasSchemas derives schemas for type aliases that instantiate a generic
// @schema, e.g. `type UserResponse = Response[User]`. The alias inherits the
// generic's description and field annotations; the resolver reads the actual
// Go types off the instantiated struct.
//
// Runs after the declared schemas are parsed so the base fields exist to copy.
func aliasSchemas(types []*typeDecl, declared []*Schema) []*Schema {
	byName := make(map[string]*Schema, len(declared))
	for _, s := range declared {
		byName[s.Name] = s
	}

	var derived []*Schema
	for _, td := range types {
		if !td.IsTypeAlias || td.AliasOf == "" || byName[td.Name] != nil {
			continue
		}

		base := byName[baseTypeName(td.AliasOf)]
		if base == nil || !base.IsGeneric {
			continue
		}

		alias := &Schema{
			Name:        td.Name,
			Pos:         td.Pos,
			Description: base.Description,
			IsTypeAlias: true,
			AliasOf:     td.AliasOf,
			TypeArgs:    typeArgs(td.AliasOf),
		}
		for _, field := range base.Fields {
			copied := *field
			alias.Fields = append(alias.Fields, &copied)
		}
		derived = append(derived, alias)
	}

	return derived
}

// baseTypeName strips the type arguments from a generic instantiation:
// "Response[User]" → "Response".
func baseTypeName(typeName string) string {
	if idx := strings.Index(typeName, "["); idx > 0 {
		return typeName[:idx]
	}
	return typeName
}

// typeArgs returns the type arguments of a generic instantiation:
// "Response[User]" → ["User"], "Pair[K, V]" → ["K", "V"].
func typeArgs(typeName string) []string {
	start := strings.Index(typeName, "[")
	end := strings.LastIndex(typeName, "]")
	if start <= 0 || end <= start {
		return nil
	}

	var args []string
	for _, arg := range strings.Split(typeName[start+1:end], ",") {
		args = append(args, strings.TrimSpace(arg))
	}
	return args
}

// parseParameterStruct parses a struct marked as a parameter group.
func parseParameterStruct(td *typeDecl, kind string) (*ParameterStruct, error) {
	fields, err := parseFields(td.Fields, td.Name)
	if err != nil {
		return nil, err
	}
	return &ParameterStruct{
		Name:   td.Name,
		Kind:   kind,
		Pos:    td.Pos,
		Fields: fields,
	}, nil
}

// parseEndpoint parses an @endpoint annotated function, including the
// declarations annotated inside its body.
func parseEndpoint(fn *funcDecl) (*Endpoint, error) {
	lines := fn.Doc.AnnotationLines()
	parsed, err := ParseAnnotationBlock(lines, "@endpoint", endpointNode)
	if err != nil {
		return nil, wrapf(fn.Pos, err, "failed to parse @endpoint for %s", fn.Name)
	}

	parts := strings.Fields(parsed.Metadata)
	if len(parts) < 2 {
		return nil, errorf(parsed.Pos, "@endpoint for %s missing method and path: %s", fn.Name, parsed.Metadata)
	}

	endpoint := &Endpoint{
		FuncName:     fn.Name,
		Pos:          fn.Pos,
		Method:       parts[0],
		Path:         parts[1],
		OperationID:  parsed.ChildValue("@operationID"),
		Summary:      parsed.ChildValue("@summary"),
		Description:  parsed.ChildValue("@description"),
		Auth:         parsed.ChildValue("@auth"),
		Deprecated:   parsed.HasChild("@deprecated"),
		Tags:         references(parsed, "@tag"),
		PathParams:   references(parsed, "@path"),
		QueryParams:  references(parsed, "@query"),
		HeaderParams: references(parsed, "@header"),
		CookieParams: references(parsed, "@cookie"),
	}

	if request := parsed.Child("@request"); request != nil {
		endpoint.Request = &RequestBody{
			ContentType: ExpandContentType(request.ChildValue("@contentType")),
			Body:        parseBody(request),
		}
	}

	for _, response := range parsed.RepeatedChildren("@response") {
		endpoint.addResponse(&Response{
			Status:      response.Metadata,
			Pos:         response.Pos,
			ContentType: ExpandContentType(response.ChildValue("@contentType")),
			Description: response.ChildValue("@description"),
			Body:        parseBody(response),
			Headers:     references(response, "@header"),
		})
	}

	inline, err := parseInline(fn)
	if err != nil {
		return nil, wrapf(fn.Pos, err, "in function %s", fn.Name)
	}
	endpoint.Inline = inline

	return endpoint, nil
}

// addResponse appends a response, replacing an earlier one with the same
// status. Repeating a status code is last-wins rather than an error: the
// historical IR keyed responses by status, so the second block silently
// replaced the first.
func (e *Endpoint) addResponse(response *Response) {
	for i, existing := range e.Responses {
		if existing.Status == response.Status {
			e.Responses[i] = response
			return
		}
	}
	e.Responses = append(e.Responses, response)
}

// references returns the values of a repeatable reference annotation, e.g. the
// parameter struct names of the @query annotations on an endpoint.
func references(parsed *Annotation, name string) []string {
	var refs []string
	for _, child := range parsed.RepeatedChildren(name) {
		if child.Value != "" {
			refs = append(refs, child.Value)
		}
	}
	return refs
}

// parseBody parses the @body of a request or response block, with its optional
// @bind. Returns nil when the block declares no body.
func parseBody(parsed *Annotation) *Body {
	body := parsed.Child("@body")
	if body == nil {
		return nil
	}
	return &Body{
		Schema: body.Value,
		Bind:   ParseBindTarget(parsed.ChildValue("@bind")),
	}
}

// ParseBindTarget parses a @bind value: "DataResponse.Data" names the wrapper
// schema and the field the body is bound into. Returns nil for anything that
// is not Wrapper.Field.
func ParseBindTarget(value string) *BindTarget {
	wrapper, field, ok := strings.Cut(strings.TrimSpace(value), ".")
	if !ok {
		return nil
	}
	return &BindTarget{
		Wrapper: strings.TrimSpace(wrapper),
		Field:   strings.TrimSpace(field),
	}
}

// parseInline parses the declarations annotated inside a handler body. Returns
// nil when the body has none.
func parseInline(fn *funcDecl) (*EndpointInline, error) {
	if len(fn.Inlines) == 0 {
		return nil, nil
	}

	inline := &EndpointInline{}
	for _, decl := range fn.Inlines {
		declared, err := parseInlineStruct(decl)
		if err != nil {
			return nil, err
		}

		switch decl.Marker {
		case "@path":
			inline.Path = append(inline.Path, declared)
		case "@query":
			inline.Query = append(inline.Query, declared)
		case "@header":
			inline.Header = append(inline.Header, declared)
		case "@cookie":
			inline.Cookie = append(inline.Cookie, declared)
		case "@request":
			// One request body per handler: a second one is a mistake, not a
			// composition. Named @schema types are how bodies compose.
			if inline.Request != nil {
				return nil, errorf(decl.Pos, "duplicate inline @request on %q (previous: %q); only one inline @request per handler is allowed",
					decl.VarName, inline.Request.VarName)
			}
			inline.Request = declared
		case "@response":
			if previous := inline.response(decl.Status); previous != nil {
				return nil, errorf(decl.Pos, "duplicate inline @response %s on %q (previous: %q); each status code can have only one inline response per handler",
					decl.Status, decl.VarName, previous.VarName)
			}
			inline.Responses = append(inline.Responses, &InlineResponse{
				Status:       decl.Status,
				InlineStruct: *declared,
			})
		}
	}

	return inline, nil
}

// response returns the inline response already declared for a status, or nil.
func (i *EndpointInline) response(status string) *InlineResponse {
	for _, response := range i.Responses {
		if response.Status == status {
			return response
		}
	}
	return nil
}

// parseInlineStruct parses one annotated declaration from a handler body: its
// marker annotation and the @field annotations of the struct it declares.
func parseInlineStruct(decl *inlineDecl) (*InlineStruct, error) {
	fields, err := parseFields(decl.Fields, decl.VarName)
	if err != nil {
		return nil, err
	}

	parsed, err := ParseAnnotationBlock(decl.Lines, decl.Marker, InlineGrammar[decl.Marker])
	if err != nil {
		return nil, wrapf(decl.Pos, err, "failed to parse inline %s on %q", decl.Marker, decl.VarName)
	}

	return &InlineStruct{
		VarName:     decl.VarName,
		Pos:         decl.Pos,
		Struct:      decl.Struct,
		Fields:      fields,
		ContentType: ExpandContentType(parsed.ChildValue("@contentType")),
		Description: parsed.ChildValue("@description"),
		Bind:        ParseBindTarget(parsed.ChildValue("@bind")),
		Headers:     references(parsed, "@header"),
	}, nil
}

// parseFields parses the @field annotation of every annotated field of a
// struct, in declaration order.
//
// Fields with no @field annotation produce no entry: the resolver walks the Go
// struct for the complete field list and matches these by GoName.
func parseFields(fields []*fieldDecl, owner string) ([]*Field, error) {
	var parsed []*Field
	for _, decl := range fields {
		if !decl.Doc.HasAnnotation("@field") {
			continue
		}

		annotation, err := ParseAnnotationBlock(decl.Doc.AnnotationLines(), "@field", fieldNode)
		if err != nil {
			return nil, wrapf(decl.Pos, err, "failed to parse @field for %s.%s", owner, decl.GoName)
		}

		field, err := newField(decl, annotation)
		if err != nil {
			return nil, wrapf(decl.Pos, err, "invalid @field for %s.%s", owner, decl.GoName)
		}
		parsed = append(parsed, field)
	}
	return parsed, nil
}

// newField converts a parsed @field annotation into a Field.
func newField(decl *fieldDecl, a *Annotation) (*Field, error) {
	field := &Field{
		GoName:      decl.GoName,
		Pos:         decl.Pos,
		Description: a.ChildValue("@description"),
		Format:      a.ChildValue("@format"),
		Constraints: Constraints{
			Default:     a.ChildValue("@default"),
			Example:     a.ChildValue("@example"),
			Pattern:     a.ChildValue("@pattern"),
			UniqueItems: a.HasChild("@uniqueItems"),
			Deprecated:  a.HasChild("@deprecated"),
			ReadOnly:    a.HasChild("@readOnly"),
			WriteOnly:   a.HasChild("@writeOnly"),
		},
	}

	if enum := a.ChildValue("@enum"); enum != "" {
		for _, value := range strings.Split(enum, ",") {
			field.Enum = append(field.Enum, strings.TrimSpace(value))
		}
	}

	var err error
	if field.MinLength, err = intChild(a, "@minLength"); err != nil {
		return nil, err
	}
	if field.MaxLength, err = intChild(a, "@maxLength"); err != nil {
		return nil, err
	}
	if field.MinItems, err = intChild(a, "@minItems"); err != nil {
		return nil, err
	}
	if field.MaxItems, err = intChild(a, "@maxItems"); err != nil {
		return nil, err
	}
	if field.Minimum, err = floatChild(a, "@minimum"); err != nil {
		return nil, err
	}
	if field.Maximum, err = floatChild(a, "@maximum"); err != nil {
		return nil, err
	}
	if field.ExclusiveMinimum, err = floatChild(a, "@exclusiveMinimum"); err != nil {
		return nil, err
	}
	if field.ExclusiveMaximum, err = floatChild(a, "@exclusiveMaximum"); err != nil {
		return nil, err
	}
	if field.Required, err = boolChild(a, "@required"); err != nil {
		return nil, err
	}
	if field.Nullable, err = boolChild(a, "@nullable"); err != nil {
		return nil, err
	}

	return field, nil
}

// intChild reads an optional integer child annotation. A missing annotation is
// nil, not zero: the resolver treats nil as "not specified".
func intChild(a *Annotation, name string) (*int, error) {
	raw := a.ChildValue(name)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, errorf(a.ChildPos(name), "%s value %q is not a valid integer", name, raw)
	}
	return &value, nil
}

// floatChild reads an optional number child annotation.
func floatChild(a *Annotation, name string) (*float64, error) {
	raw := a.ChildValue(name)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, errorf(a.ChildPos(name), "%s value %q is not a valid number", name, raw)
	}
	return &value, nil
}

// boolChild reads an optional boolean child annotation.
func boolChild(a *Annotation, name string) (*bool, error) {
	raw := a.ChildValue(name)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, errorf(a.ChildPos(name), "%s value %q is not a valid boolean (use true or false)", name, raw)
	}
	return &value, nil
}

// contentTypes maps the short names an annotation may use to MIME types. The
// "empty" sentinel means "no content" and expands to the empty string.
var contentTypes = map[string]string{
	"json":      "application/json",
	"xml":       "application/xml",
	"form":      "application/x-www-form-urlencoded",
	"multipart": "multipart/form-data",
	"text":      "text/plain",
	"csv":       "text/csv",
	"binary":    "application/octet-stream",
	"html":      "text/html",
	"empty":     "",
}

// ExpandContentType expands a short content-type name to its MIME type.
// Anything already containing a slash, or not in the table, passes through.
func ExpandContentType(shortName string) string {
	if strings.Contains(shortName, "/") {
		return shortName
	}
	if mimeType, ok := contentTypes[strings.ToLower(shortName)]; ok {
		return mimeType
	}
	return shortName
}
