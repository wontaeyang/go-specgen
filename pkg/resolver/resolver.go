package resolver

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"reflect"
	"slices"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/parser"
)

// The resolver joins the parsed annotations to the Go types they describe and
// produces the emission-ready IR: schemas sorted by name, and endpoints whose
// parameters and responses are already in the order the generator emits them.
//
// Errors accumulate per top-level item — per schema, per parameter struct — so
// one bad declaration does not hide the rest of the package.

// paramKinds are the parameter kinds in emission order.
var paramKinds = []string{parser.ParamPath, parser.ParamQuery, parser.ParamHeader, parser.ParamCookie}

// resolver holds the lookups resolution needs: the Go package scope and the
// tables that turn annotation names into resolved declarations.
type resolver struct {
	scope *types.Scope

	// schemas is every @schema by name, including generic templates. It backs
	// reference detection: a field typed with one of these emits a $ref.
	schemas map[string]*parser.Schema

	// resolved is the schemas already resolved, by name, for @bind lookups.
	resolved map[string]*Schema

	// groups is the fields of each resolved parameter struct, by name.
	groups map[string][]*Field

	// defaultContentType is the API-level default, applied when an annotation
	// declares none.
	defaultContentType string
}

// Resolve resolves a parsed package into the emission-ready IR.
func Resolve(pkg *parser.Package) (*Package, error) {
	if pkg.GoPkg == nil || pkg.GoPkg.Types == nil {
		return nil, errors.New("package was parsed without Go type information")
	}

	r := &resolver{
		scope:    pkg.GoPkg.Types.Scope(),
		schemas:  make(map[string]*parser.Schema, len(pkg.Schemas)),
		resolved: make(map[string]*Schema, len(pkg.Schemas)),
		groups:   make(map[string][]*Field, len(pkg.Parameters)),
	}
	for _, schema := range pkg.Schemas {
		r.schemas[schema.Name] = schema
	}
	if pkg.API != nil {
		r.defaultContentType = pkg.API.DefaultContentType
	}

	out := &Package{Name: pkg.Name, API: pkg.API}
	var errs []error

	// Schemas are emitted sorted by name, so they are resolved in that order
	// too and their errors come out in a stable order.
	schemas := slices.Clone(pkg.Schemas)
	slices.SortFunc(schemas, func(a, b *parser.Schema) int {
		return strings.Compare(a.Name, b.Name)
	})
	for _, schema := range schemas {
		resolved, err := r.resolveSchema(schema)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out.Schemas = append(out.Schemas, resolved)
		r.resolved[resolved.Name] = resolved
	}

	for _, param := range pkg.Parameters {
		fields, err := r.resolveParameterStruct(param)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		r.groups[param.Name] = fields
	}

	for _, endpoint := range pkg.Endpoints {
		resolved, err := r.resolveEndpoint(endpoint)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out.Endpoints = append(out.Endpoints, resolved)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

// errorf builds a positioned error, rendered as "file.go:12:3: message".
func errorf(pos token.Position, format string, args ...any) *parser.Error {
	return &parser.Error{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

// resolveSchema resolves the Go struct behind a @schema and its fields.
func (r *resolver) resolveSchema(schema *parser.Schema) (*Schema, error) {
	st, err := r.structType(schema.Name, schema.Pos)
	if err != nil {
		return nil, err
	}

	return &Schema{
		Name:        schema.Name,
		Description: schema.Description,
		Deprecated:  schema.Deprecated,
		IsGeneric:   schema.IsGeneric,
		Fields:      r.resolveStructFields(st, schema.Fields, "", nil),
	}, nil
}

// resolveParameterStruct resolves a struct marked @path, @query, @header or
// @cookie. The kind it declares decides how its fields resolve, wherever they
// are later referenced from.
func (r *resolver) resolveParameterStruct(param *parser.ParameterStruct) ([]*Field, error) {
	st, err := r.structType(param.Name, param.Pos)
	if err != nil {
		return nil, err
	}
	return r.resolveStructFields(st, param.Fields, param.Kind, nil), nil
}

// structType looks up a declared type and returns the struct behind it. Type
// aliases resolve through to the struct they name, which is how a generic
// instantiation (type UserResponse = Response[User]) reaches its substituted
// fields.
func (r *resolver) structType(name string, pos token.Position) (*types.Struct, error) {
	obj := r.scope.Lookup(name)
	if obj == nil {
		return nil, errorf(pos, "struct %s not found in package", name)
	}

	st, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		return nil, errorf(pos, "%s is not a struct", name)
	}
	return st, nil
}

// resolveStructFields resolves every serializable field of a struct, flattening
// embedded structs and applying the @field annotations, which bind by Go name.
//
// kind selects the serialization: "" resolves the fields as a JSON body,
// otherwise they are parameters of that kind ("path", "query", "header",
// "cookie"), which changes the tag consulted for the wire name, the rule that
// decides required, and nullability.
//
// visited carries the embedded types already being walked, so a cycle of
// mutually embedding structs terminates. It starts nil.
func (r *resolver) resolveStructFields(st *types.Struct, annos []*parser.Field, kind string, visited map[string]bool) []*Field {
	// Binding is scope-blind: an annotation matches any field with that Go
	// name, including one flattened in from an embedded struct.
	byGoName := make(map[string]*parser.Field, len(annos))
	for _, anno := range annos {
		if _, taken := byGoName[anno.GoName]; !taken {
			byGoName[anno.GoName] = anno
		}
	}

	var fields []*Field
	for i := range st.NumFields() {
		field := st.Field(i)
		tag := reflect.StructTag(st.Tag(i))

		// Embedded fields contribute their own fields, at this level.
		if field.Anonymous() {
			fields = append(fields, r.flattenEmbedded(field, annos, kind, visited)...)
			continue
		}

		// Unexported fields are never serialized.
		if !field.Exported() {
			continue
		}

		if resolved := r.resolveField(field, tag, byGoName[field.Name()], kind); resolved != nil {
			fields = append(fields, resolved)
		}
	}

	return fields
}

// flattenEmbedded resolves an embedded field into the fields it contributes to
// the embedding struct.
func (r *resolver) flattenEmbedded(field *types.Var, annos []*parser.Field, kind string, visited map[string]bool) []*Field {
	if visited == nil {
		visited = make(map[string]bool)
	}

	st, done := unwrapEmbedded(field.Type(), visited)
	defer done()
	if st == nil {
		return nil
	}
	return r.resolveStructFields(st, annos, kind, visited)
}

// unwrapEmbedded unwraps an embedded field's type, through a pointer and a
// named type, to the struct it embeds. It returns nil when the type is not a
// struct or when it is already being walked, which is how a cycle of mutually
// embedding structs terminates. The returned function must be called once the
// struct has been walked.
func unwrapEmbedded(t types.Type, visited map[string]bool) (*types.Struct, func()) {
	done := func() {}

	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	if named, ok := t.(*types.Named); ok {
		name := named.Obj().Name()
		if visited[name] {
			return nil, done
		}
		visited[name] = true
		done = func() { delete(visited, name) }

		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			return nil, done
		}
		return st, done
	}

	if st, ok := t.(*types.Struct); ok {
		return st, done
	}
	return nil, done
}

// resolveField resolves one struct field. It returns nil for a field that is
// not serialized at all (json:"-").
func (r *resolver) resolveField(field *types.Var, tag reflect.StructTag, anno *parser.Field, kind string) *Field {
	var name string
	if kind == "" {
		// Only a body field can be skipped outright (json:"-"); a parameter
		// always falls back to its Go name.
		name = bodyFieldName(tag, field.Name())
		if name == "" {
			return nil
		}
	} else {
		name = paramFieldName(tag, kind, field.Name())
	}

	out := &Field{Name: name, GoName: field.Name()}
	omitted := omitsWhenEmpty(tag, field.Type())

	switch {
	case kind == parser.ParamPath:
		// A path parameter is part of the URL, so it is always required.
		out.Required = true
	case kind != "":
		// Query, header and cookie parameters are optional unless the tag
		// opts in, e.g. `query:"q,required"`.
		out.Required = hasTagOption(tag, kind, "required")
	default:
		out.Required = !omitted
	}

	if kind == "" {
		r.resolveBodyType(out, field.Type(), omitted)
	} else {
		// Parameters serialize as plain text, which cannot carry null: a
		// pointer only lets the handler tell absent from zero. @nullable true
		// can still opt in.
		out.Type, out.Format, _ = r.typeInfo(field.Type())
	}

	applyOverrides(out, anno)
	return out
}

// resolveBodyType fills in the type of a body field. Anonymous structs are
// inlined because they have no name to reference; everything else resolves
// through typeInfo.
func (r *resolver) resolveBodyType(out *Field, t types.Type, omitted bool) {
	if st := anonymousStruct(t); st != nil {
		out.Inline = r.anonymousFields(st)
		out.Type = TypeInfo{OpenAPI: "object"}
		return
	}
	if st := anonymousStruct(sliceElem(t)); st != nil {
		out.ItemsInline = r.anonymousFields(st)
		out.Type = TypeInfo{IsArray: true, Items: "object"}
		return
	}
	if st := anonymousStruct(mapElem(t)); st != nil {
		out.MapValueInline = r.anonymousFields(st)
		out.Type = TypeInfo{IsMap: true, MapValue: "string"}
		return
	}

	var nullable bool
	out.Type, out.Format, nullable = r.typeInfo(t)

	// A tag that omits the empty value drops a nil pointer instead of encoding
	// null, so the field can never appear as null on the wire.
	out.Nullable = nullable && !omitted
	out.Unresolved = r.unresolvedStruct(t)
}

// anonymousFields resolves the fields of an anonymous struct.
//
// PINNED: the @field annotations of the enclosing declaration are not passed
// down, so an annotated field inside an anonymous struct emits bare. That is
// what examples/inline/inline.yaml records today (the nested items lose their
// @description and @minimum), and it stays until the golden is regenerated.
func (r *resolver) anonymousFields(st *types.Struct) []*Field {
	var annotations []*parser.Field
	return r.resolveStructFields(st, annotations, "", nil)
}

// applyOverrides applies an @field annotation on top of what the Go type
// implied. Every value is an override: an unset value in the annotation leaves
// the resolved value alone.
func applyOverrides(field *Field, anno *parser.Field) {
	if anno == nil {
		return
	}

	if anno.Description != "" {
		field.Description = anno.Description
	}
	if anno.Format != "" {
		field.Format = anno.Format
	}
	if anno.Required != nil {
		field.Required = *anno.Required
	}
	if anno.Nullable != nil {
		field.Nullable = *anno.Nullable
	}

	// Constraints are pure pass-through and nothing else sets them, so the
	// whole block copies over.
	field.Constraints = Constraints(anno.Constraints)
}

// resolveEndpoint resolves an endpoint into an operation with its parameters
// and responses already in emission order.
func (r *resolver) resolveEndpoint(endpoint *parser.Endpoint) (*Endpoint, error) {
	params, err := r.resolveParameters(endpoint)
	if err != nil {
		return nil, err
	}

	responses, err := r.resolveResponses(endpoint)
	if err != nil {
		return nil, err
	}

	return &Endpoint{
		Method:      endpoint.Method,
		Path:        endpoint.Path,
		OperationID: endpoint.OperationID,
		Summary:     endpoint.Summary,
		Description: endpoint.Description,
		Auth:        endpoint.Auth,
		Tags:        endpoint.Tags,
		Deprecated:  endpoint.Deprecated,
		Parameters:  params,
		Request:     r.resolveRequest(endpoint),
		Responses:   responses,
	}, nil
}

// resolveParameters builds the operation parameters in emission order: the
// referenced parameter structs by kind first, then the inline declarations by
// kind. Fields are emitted under the kind they were referenced as, which is
// not necessarily the kind their struct was declared with.
func (r *resolver) resolveParameters(endpoint *parser.Endpoint) ([]*Param, error) {
	var params []*Param

	for _, kind := range paramKinds {
		for _, name := range namedParams(endpoint, kind) {
			fields, err := r.group(name, kind, endpoint.Pos)
			if err != nil {
				return nil, err
			}
			for _, field := range fields {
				params = append(params, &Param{In: kind, Field: field})
			}
		}
	}

	if endpoint.Inline == nil {
		return params, nil
	}

	for _, kind := range paramKinds {
		for _, decl := range inlineParams(endpoint.Inline, kind) {
			for _, field := range r.resolveStructFields(decl.Struct, decl.Fields, kind, nil) {
				params = append(params, &Param{In: kind, Field: field})
			}
		}
	}

	return params, nil
}

// group returns the resolved fields of a referenced parameter struct. A name
// with no declaration behind it is an error: dropping it would emit an
// operation quietly missing the parameters the annotation asked for.
func (r *resolver) group(name, kind string, pos token.Position) ([]*Field, error) {
	fields, declared := r.groups[name]
	if !declared {
		return nil, errorf(pos, "@%s references unknown parameter struct: %s", kind, name)
	}
	return fields, nil
}

// namedParams returns the parameter structs an endpoint references for a kind.
func namedParams(endpoint *parser.Endpoint, kind string) []string {
	switch kind {
	case parser.ParamPath:
		return endpoint.PathParams
	case parser.ParamQuery:
		return endpoint.QueryParams
	case parser.ParamHeader:
		return endpoint.HeaderParams
	case parser.ParamCookie:
		return endpoint.CookieParams
	}
	return nil
}

// inlineParams returns the inline parameter declarations of a kind, in
// declaration order. A handler may declare several per kind; OpenAPI emits one
// flat parameter list, so they concatenate.
func inlineParams(inline *parser.EndpointInline, kind string) []*parser.InlineStruct {
	switch kind {
	case parser.ParamPath:
		return inline.Path
	case parser.ParamQuery:
		return inline.Query
	case parser.ParamHeader:
		return inline.Header
	case parser.ParamCookie:
		return inline.Cookie
	}
	return nil
}

// resolveRequest builds the request body. A named @request with a body wins
// over an inline declaration; a named @request without one produces no request
// body at all.
func (r *resolver) resolveRequest(endpoint *parser.Endpoint) *Request {
	if endpoint.Request != nil && endpoint.Request.Body != nil {
		body := endpoint.Request.Body
		return &Request{
			ContentType: r.contentType(endpoint.Request.ContentType),
			Required:    true,
			Content: &Content{
				Ref:  typeRef(body.Schema),
				Bind: r.bindTarget(body.Bind),
			},
		}
	}

	if endpoint.Inline == nil || endpoint.Inline.Request == nil {
		return nil
	}

	decl := endpoint.Inline.Request
	request := &Request{
		ContentType: r.contentType(decl.ContentType),
		Required:    true,
	}
	if fields := r.resolveStructFields(decl.Struct, decl.Fields, "", nil); len(fields) > 0 {
		request.Content = &Content{Fields: fields, Bind: r.bindTarget(decl.Bind)}
	}
	return request
}

// resolveResponses builds the responses in emission order: the statuses
// declared in the endpoint comment sorted, then the inline ones sorted, with
// a declared status winning over an inline one.
func (r *resolver) resolveResponses(endpoint *parser.Endpoint) ([]*Response, error) {
	named := slices.Clone(endpoint.Responses)
	slices.SortFunc(named, func(a, b *parser.Response) int {
		return strings.Compare(a.Status, b.Status)
	})

	var responses []*Response
	for _, response := range named {
		resolved, err := r.namedResponse(response)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resolved)
	}

	if endpoint.Inline == nil {
		return responses, nil
	}

	inline := slices.Clone(endpoint.Inline.Responses)
	slices.SortFunc(inline, func(a, b *parser.InlineResponse) int {
		return strings.Compare(a.Status, b.Status)
	})
	for _, response := range inline {
		taken := slices.ContainsFunc(named, func(other *parser.Response) bool {
			return other.Status == response.Status
		})
		if taken {
			continue
		}
		resolved, err := r.inlineResponse(response)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resolved)
	}

	return responses, nil
}

// namedResponse resolves an @response block of the endpoint comment. The
// content type is defaulted only when the response has a body, so a bodyless
// response emits no content at all.
func (r *resolver) namedResponse(response *parser.Response) (*Response, error) {
	headers, err := r.headerFields(response.Headers, response.Pos)
	if err != nil {
		return nil, err
	}

	out := &Response{
		Status:      response.Status,
		Description: response.Description,
		ContentType: response.ContentType,
		Headers:     headers,
	}

	if response.Body == nil {
		return out, nil
	}
	out.ContentType = r.contentType(response.ContentType)

	// An explicitly empty content type (@contentType empty) suppresses the
	// content block even though a body was named.
	if response.Body.Schema != "" && out.ContentType != "" {
		out.Content = &Content{
			Ref:  typeRef(response.Body.Schema),
			Bind: r.bindTarget(response.Body.Bind),
		}
	}
	return out, nil
}

// inlineResponse resolves a response declared in the handler body. The struct
// is the body, so there is always a content type, and an undescribed response
// gets a generated description.
func (r *resolver) inlineResponse(response *parser.InlineResponse) (*Response, error) {
	headers, err := r.headerFields(response.Headers, response.Pos)
	if err != nil {
		return nil, err
	}

	description := response.Description
	if description == "" {
		description = fmt.Sprintf("Response for status %s", response.Status)
	}

	out := &Response{
		Status:      response.Status,
		Description: description,
		ContentType: r.contentType(response.ContentType),
		Headers:     headers,
	}
	if fields := r.resolveStructFields(response.Struct, response.Fields, "", nil); len(fields) > 0 {
		out.Content = &Content{Fields: fields, Bind: r.bindTarget(response.Bind)}
	}
	return out, nil
}

// headerFields flattens the @header parameter structs a response references
// into fields, in group-then-field order.
func (r *resolver) headerFields(refs []string, pos token.Position) ([]*Field, error) {
	var fields []*Field
	for _, ref := range refs {
		group, err := r.group(ref, parser.ParamHeader, pos)
		if err != nil {
			return nil, err
		}
		fields = append(fields, group...)
	}
	return fields, nil
}

// contentType applies the content-type defaulting chain: what the annotation
// declared, else the API default, else JSON.
func (r *resolver) contentType(declared string) string {
	if declared != "" {
		return declared
	}
	if r.defaultContentType != "" {
		return r.defaultContentType
	}
	return "application/json"
}

// bindTarget resolves a @bind to the wrapper schema it names. An unknown
// wrapper leaves Wrapper nil, which falls back to emitting the plain body; the
// validator reports it.
func (r *resolver) bindTarget(bind *parser.BindTarget) *BindTarget {
	if bind == nil {
		return nil
	}
	return &BindTarget{
		Name:    bind.Wrapper,
		Field:   bind.Field,
		Wrapper: r.resolved[bind.Wrapper],
	}
}

// primitiveTypes maps the Go primitive names a @body value can use to their
// OpenAPI types. A @body naming anything else names a schema.
var primitiveTypes = map[string]string{
	"string":  "string",
	"bool":    "boolean",
	"int":     "integer",
	"int8":    "integer",
	"int16":   "integer",
	"int32":   "integer",
	"int64":   "integer",
	"uint":    "integer",
	"uint8":   "integer",
	"uint16":  "integer",
	"uint32":  "integer",
	"uint64":  "integer",
	"float32": "number",
	"float64": "number",
	// A []byte body is base64 text on the wire, not an array of numbers.
	"byte": "string",
}

// typeRef parses a @body value into a reference: "User", "[]User" or
// "map[string]User".
func typeRef(schema string) *TypeRef {
	ref := &TypeRef{}

	element := strings.TrimSpace(schema)
	switch {
	case strings.HasPrefix(element, "[]"):
		ref.IsArray = true
		element = strings.TrimPrefix(element, "[]")
	case strings.HasPrefix(element, "map[string]"):
		ref.IsMap = true
		element = strings.TrimPrefix(element, "map[string]")
	}

	if primitive, ok := primitiveTypes[element]; ok {
		ref.Primitive = primitive
	} else {
		ref.Schema = element
	}
	return ref
}
