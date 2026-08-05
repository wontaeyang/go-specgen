package resolver

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/parser"
)

// These tests go through the real pipeline — parser.Parse then Resolve — over
// two fixture packages: pkg/parser/testdata/valid, which carries the
// annotation surface, and testdata/shapes, which carries the Go type shapes.

const (
	validPkg  = "../parser/testdata/valid"
	shapesPkg = "./testdata/shapes"
)

func resolvePackage(t *testing.T, dir string) *Package {
	t.Helper()

	parsed, err := parser.Parse(dir)
	if err != nil {
		t.Fatalf("parse %s: %v", dir, err)
	}
	pkg, err := Resolve(parsed)
	if err != nil {
		t.Fatalf("resolve %s: %v", dir, err)
	}
	return pkg
}

func schema(t *testing.T, pkg *Package, name string) *Schema {
	t.Helper()

	for _, s := range pkg.Schemas {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("schema %s not resolved", name)
	return nil
}

func field(t *testing.T, s *Schema, name string) *Field {
	t.Helper()

	for _, f := range s.Fields {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("schema %s has no field %q", s.Name, name)
	return nil
}

func endpoint(t *testing.T, pkg *Package, method, path string) *Endpoint {
	t.Helper()

	for _, e := range pkg.Endpoints {
		if e.Method == method && e.Path == path {
			return e
		}
	}
	t.Fatalf("endpoint %s %s not resolved", method, path)
	return nil
}

func fieldNames(fields []*Field) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}

func TestResolve_SchemasSortedByName(t *testing.T) {
	pkg := resolvePackage(t, validPkg)

	var names []string
	for _, s := range pkg.Schemas {
		names = append(names, s.Name)
	}

	want := []string{
		"EmbeddedPtrTest", "EmbeddedTest", "Envelope", "Error", "Escapes",
		"FieldRequiredTest", "LegacyUser", "NestedEmbedTest", "Pair",
		"Response", "StringUserPair", "User", "UserResponse",
	}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("schema names = %v, want %v", names, want)
	}
}

func TestResolve_SchemaMetadata(t *testing.T) {
	pkg := resolvePackage(t, validPkg)

	user := schema(t, pkg, "User")
	if user.Description != "A user of the system.\nSecond line of the schema description." {
		t.Errorf("User description = %q", user.Description)
	}
	if user.Deprecated {
		t.Error("User should not be deprecated")
	}

	if legacy := schema(t, pkg, "LegacyUser"); !legacy.Deprecated {
		t.Error("LegacyUser should be deprecated")
	}
}

// TestResolve_RequiredAndNullable is the outcome table from
// examples/overrides/overrides.go: required and nullable describe what
// encoding/json puts on the wire, not what the tag text says.
func TestResolve_RequiredAndNullable(t *testing.T) {
	pkg := resolvePackage(t, validPkg)
	s := schema(t, pkg, "FieldRequiredTest")

	tests := []struct {
		field    string
		required bool
		nullable bool
	}{
		{"value", true, false},
		{"value_ptr", true, true},
		{"value_omit", false, false},
		{"value_ptr_omit", false, false},
		{"value_zero", false, false},
		{"value_ptr_zero", false, false},
		{"value_struct", true, false},
		{"value_struct_ptr", true, true},
		// A non-pointer struct is never empty, so omitempty cannot drop it.
		{"value_struct_omit", true, false},
		{"value_struct_zero", false, false},
		{"value_slice", true, false},
		{"value_slice_omit", false, false},
		{"value_slice_ptr", true, true},
		{"value_slice_ptr_omit", false, false},
		{"value_map", true, false},
		{"value_map_omit", false, false},
	}

	for _, tt := range tests {
		f := field(t, s, tt.field)
		if f.Required != tt.required {
			t.Errorf("%s: required = %v, want %v", tt.field, f.Required, tt.required)
		}
		if f.Nullable != tt.nullable {
			t.Errorf("%s: nullable = %v, want %v", tt.field, f.Nullable, tt.nullable)
		}
	}
}

func TestResolve_AnnotationOverrides(t *testing.T) {
	pkg := resolvePackage(t, validPkg)
	user := schema(t, pkg, "User")

	// @required false and @nullable true override what the tag and the
	// pointer implied.
	nickname := field(t, user, "nickname")
	if nickname.Required || !nickname.Nullable {
		t.Errorf("nickname: required = %v, nullable = %v; want false, true", nickname.Required, nickname.Nullable)
	}

	id := field(t, user, "id")
	if id.Description != "User ID" || id.Format != "uuid" || !id.ReadOnly {
		t.Errorf("id = %+v; want description, format uuid, readOnly", id)
	}

	role := field(t, user, "role")
	if !reflect.DeepEqual(role.Enum, []string{"admin", "user", "guest"}) {
		t.Errorf("role enum = %v", role.Enum)
	}
	if role.Default != "user" {
		t.Errorf("role default = %q", role.Default)
	}

	age := field(t, user, "age")
	if age.Minimum == nil || *age.Minimum != 0 || age.Maximum == nil || *age.Maximum != 130 {
		t.Errorf("age bounds = %v..%v", age.Minimum, age.Maximum)
	}
	if age.ExclusiveMinimum == nil || *age.ExclusiveMinimum != 0.5 {
		t.Errorf("age exclusiveMinimum = %v", age.ExclusiveMinimum)
	}

	labels := field(t, user, "labels")
	if labels.MinItems == nil || *labels.MinItems != 1 || !labels.UniqueItems {
		t.Errorf("labels constraints = %+v", labels.Constraints)
	}

	// A field with no @field annotation still resolves from the Go type.
	internal := field(t, user, "internal")
	if internal.Description != "" || !internal.Required {
		t.Errorf("internal = %+v; want no description and required", internal)
	}
}

func TestResolve_EmbeddedFlattening(t *testing.T) {
	pkg := resolvePackage(t, validPkg)

	tests := []struct {
		schema string
		want   []string
	}{
		{"EmbeddedTest", []string{"id", "created_at", "updated_at", "name"}},
		{"EmbeddedPtrTest", []string{"id", "created_at", "updated_at", "label"}},
		{"NestedEmbedTest", []string{"id", "created_at", "updated_at", "deleted_at", "deleted_by", "status"}},
	}

	for _, tt := range tests {
		got := fieldNames(schema(t, pkg, tt.schema).Fields)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s fields = %v, want %v", tt.schema, got, tt.want)
		}
	}
}

func TestResolve_EmbeddedCycleTerminates(t *testing.T) {
	pkg := resolvePackage(t, shapesPkg)

	// Ring embeds Loop, which embeds *Ring again. The second visit stops.
	got := fieldNames(schema(t, pkg, "Ring").Fields)
	want := []string{"name", "label", "name"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Ring fields = %v, want %v", got, want)
	}
}

func TestResolve_TypeShapes(t *testing.T) {
	pkg := resolvePackage(t, shapesPkg)
	s := schema(t, pkg, "Shapes")

	tests := []struct {
		field  string
		want   TypeInfo
		format string
	}{
		{field: "text", want: TypeInfo{OpenAPI: "string"}},
		{field: "count", want: TypeInfo{OpenAPI: "integer"}, format: "int32"},
		{field: "big", want: TypeInfo{OpenAPI: "integer"}, format: "int64"},
		{field: "ratio", want: TypeInfo{OpenAPI: "number"}, format: "float"},
		{field: "precise", want: TypeInfo{OpenAPI: "number"}, format: "double"},
		// OpenAPI has no unsigned formats.
		{field: "unsigned", want: TypeInfo{OpenAPI: "integer"}},
		{field: "flag", want: TypeInfo{OpenAPI: "boolean"}},
		{field: "created", want: TypeInfo{OpenAPI: "string"}, format: "date-time"},
		{field: "link", want: TypeInfo{OpenAPI: "string"}, format: "uri"},
		// []byte is base64 text, carried as a format on the array.
		{field: "blob", want: TypeInfo{IsArray: true, Items: "string"}, format: "byte"},
		{field: "names", want: TypeInfo{IsArray: true, Items: "string"}},
		{field: "fixed", want: TypeInfo{IsArray: true, Items: "integer"}},
		// Items keeps the element's scalar type alongside the reference,
		// because parameter schemas emit that instead of the reference.
		{field: "addresses", want: TypeInfo{IsArray: true, Items: "string", ItemsRef: "Address"}},
		{field: "ptr_addrs", want: TypeInfo{IsArray: true, Items: "string", ItemsRef: "Address"}},
		{field: "book", want: TypeInfo{IsMap: true, MapValueRef: "Address"}},
		{field: "counts", want: TypeInfo{IsMap: true, MapValue: "integer"}},
		{field: "times", want: TypeInfo{IsMap: true, MapValue: "string"}},
		{field: "anything", want: TypeInfo{IsAny: true}},
		{field: "home", want: TypeInfo{Ref: "Address"}},
		{field: "optional", want: TypeInfo{Ref: "Address"}},
		// A struct with no @schema has nothing to reference.
		{field: "stranger", want: TypeInfo{OpenAPI: "string"}},
	}

	for _, tt := range tests {
		f := field(t, s, tt.field)
		if f.Type != tt.want {
			t.Errorf("%s: type = %+v, want %+v", tt.field, f.Type, tt.want)
		}
		if f.Format != tt.format {
			t.Errorf("%s: format = %q, want %q", tt.field, f.Format, tt.format)
		}
	}
}

func TestResolve_FieldNamesFromTags(t *testing.T) {
	pkg := resolvePackage(t, shapesPkg)

	got := fieldNames(schema(t, pkg, "Shapes").Fields)
	for _, name := range []string{"from_xml", "NoTag"} {
		if !slices.Contains(got, name) {
			t.Errorf("fields %v should contain %q", got, name)
		}
	}
	// json:"-" is skipped, and so are unexported fields.
	for _, name := range []string{"Skipped", "hidden"} {
		if slices.Contains(got, name) {
			t.Errorf("fields %v should not contain %q", got, name)
		}
	}
}

func TestResolve_UnresolvedStructs(t *testing.T) {
	pkg := resolvePackage(t, shapesPkg)
	s := schema(t, pkg, "Shapes")

	// Reported through the type itself and through a slice of it.
	for _, name := range []string{"stranger", "strangers"} {
		if got := field(t, s, name).Unresolved; got != "Untagged" {
			t.Errorf("%s: unresolved = %q, want Untagged", name, got)
		}
	}
	// A @schema struct, a special type and a primitive are all resolvable.
	for _, name := range []string{"home", "created", "text", "book"} {
		if got := field(t, s, name).Unresolved; got != "" {
			t.Errorf("%s: unresolved = %q, want empty", name, got)
		}
	}
}

func TestResolve_AnonymousStructs(t *testing.T) {
	pkg := resolvePackage(t, shapesPkg)
	s := schema(t, pkg, "Nested")

	inline := field(t, s, "inline")
	if inline.Type != (TypeInfo{OpenAPI: "object"}) || len(inline.Inline) != 1 {
		t.Fatalf("inline = %+v with %d fields", inline.Type, len(inline.Inline))
	}
	if inline.Description != "An inline object" {
		t.Errorf("inline description = %q", inline.Description)
	}

	items := field(t, s, "items")
	if items.Type != (TypeInfo{IsArray: true, Items: "object"}) || len(items.ItemsInline) != 1 {
		t.Errorf("items = %+v with %d item fields", items.Type, len(items.ItemsInline))
	}

	values := field(t, s, "values")
	if values.Type != (TypeInfo{IsMap: true, MapValue: "string"}) || len(values.MapValueInline) != 1 {
		t.Errorf("values = %+v with %d value fields", values.Type, len(values.MapValueInline))
	}

	// An empty anonymous struct emits as a bare object.
	empty := field(t, s, "empty")
	if empty.Type != (TypeInfo{OpenAPI: "object"}) || len(empty.Inline) != 0 {
		t.Errorf("empty = %+v with %d fields", empty.Type, len(empty.Inline))
	}
}

// TestResolve_AnonymousStructsDropAnnotations pins today's behavior: a @field
// annotation inside an anonymous struct is not applied. Phase 4 flips this and
// regenerates examples/inline/inline.yaml.
func TestResolve_AnonymousStructsDropAnnotations(t *testing.T) {
	pkg := resolvePackage(t, shapesPkg)

	nested := field(t, schema(t, pkg, "Nested"), "inline")
	if len(nested.Inline) != 1 {
		t.Fatalf("expected one inlined field, got %d", len(nested.Inline))
	}
	if description := nested.Inline[0].Description; description != "" {
		t.Errorf("nested field description = %q, want it dropped", description)
	}
}

func TestResolve_Generics(t *testing.T) {
	pkg := resolvePackage(t, validPkg)

	// Templates stay in the list, flagged, so the generator can skip them.
	for _, name := range []string{"Response", "Pair"} {
		if !schema(t, pkg, name).IsGeneric {
			t.Errorf("%s should be marked generic", name)
		}
	}

	// An alias instantiating a generic resolves to the substituted fields.
	userResponse := schema(t, pkg, "UserResponse")
	if userResponse.IsGeneric {
		t.Error("UserResponse should not be marked generic")
	}
	if got := field(t, userResponse, "data").Type; got != (TypeInfo{Ref: "User"}) {
		t.Errorf("UserResponse.data type = %+v, want a User reference", got)
	}
	if got := field(t, userResponse, "success").Type; got != (TypeInfo{OpenAPI: "boolean"}) {
		t.Errorf("UserResponse.success type = %+v", got)
	}

	// Two type arguments substitute independently.
	pair := schema(t, pkg, "StringUserPair")
	if got := field(t, pair, "key").Type; got != (TypeInfo{OpenAPI: "string"}) {
		t.Errorf("StringUserPair.key type = %+v", got)
	}
	if got := field(t, pair, "value").Type; got != (TypeInfo{Ref: "User"}) {
		t.Errorf("StringUserPair.value type = %+v", got)
	}

	// A field typed with the template itself is not referenceable.
	if got := field(t, schema(t, pkg, "Response"), "data").Type; got != (TypeInfo{OpenAPI: "string"}) {
		t.Errorf("Response.data type = %+v, want the type-parameter fallback", got)
	}
}

func TestResolve_ParameterOrder(t *testing.T) {
	pkg := resolvePackage(t, validPkg)
	e := endpoint(t, pkg, "GET", "/users/{id}")

	var got []string
	for _, param := range e.Parameters {
		got = append(got, param.In+":"+param.Field.Name)
	}
	want := []string{
		"path:id", "query:limit", "query:cursor",
		"header:X-RateLimit-Limit", "cookie:session",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parameters = %v, want %v", got, want)
	}
}

func TestResolve_ParameterRules(t *testing.T) {
	pkg := resolvePackage(t, shapesPkg)
	e := endpoint(t, pkg, "GET", "/shapes/{id}")

	byName := make(map[string]*Param, len(e.Parameters))
	var order []string
	for _, param := range e.Parameters {
		byName[param.Field.Name] = param
		order = append(order, param.Field.Name)
	}

	// Embedded parameter fields flatten ahead of the embedding struct's own.
	want := []string{"id", "offset", "q", "limit", "Ignored", "Untagged", "list"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("parameters = %v, want %v", order, want)
	}

	// A path parameter is always required; the rest opt in with ",required".
	if !byName["id"].Field.Required {
		t.Error("path parameter id should be required")
	}
	if !byName["q"].Field.Required {
		t.Error("query parameter q should be required via the tag option")
	}
	if byName["limit"].Field.Required {
		t.Error("query parameter limit should be optional")
	}

	// A pointer parameter is not nullable: a query string cannot carry null.
	if byName["limit"].Field.Nullable {
		t.Error("query parameter limit should not be nullable")
	}
}

func TestResolve_InlineDeclarations(t *testing.T) {
	pkg := resolvePackage(t, validPkg)
	e := endpoint(t, pkg, "POST", "/orders")

	var got []string
	for _, param := range e.Parameters {
		got = append(got, param.In+":"+param.Field.Name)
	}
	// Several inline @query structs in one handler flatten into one list, in
	// declaration order.
	want := []string{
		"path:id", "query:status", "query:limit",
		"header:X-Idempotency-Key", "cookie:session",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parameters = %v, want %v", got, want)
	}

	if e.Request == nil || e.Request.Content == nil {
		t.Fatal("inline request not resolved")
	}
	if e.Request.ContentType != "application/xml" {
		t.Errorf("request content type = %q", e.Request.ContentType)
	}
	if !e.Request.Required {
		t.Error("request bodies are always required")
	}
	if names := fieldNames(e.Request.Content.Fields); !reflect.DeepEqual(names, []string{"customer_id"}) {
		t.Errorf("request fields = %v", names)
	}
	if e.Request.Content.Bind == nil || e.Request.Content.Bind.Wrapper == nil {
		t.Error("request @bind should resolve to the Envelope schema")
	}
}

func TestResolve_InlineDeclarationsInClosure(t *testing.T) {
	pkg := resolvePackage(t, validPkg)
	e := endpoint(t, pkg, "POST", "/greet")

	if e.Request == nil || e.Request.Content == nil {
		t.Fatal("request declared in the factory not resolved")
	}
	if len(e.Responses) != 1 || e.Responses[0].Content == nil {
		t.Fatalf("response declared in the returned closure not resolved: %+v", e.Responses)
	}
}

func TestResolve_ResponseOrder(t *testing.T) {
	pkg := resolvePackage(t, validPkg)

	var got []string
	for _, response := range endpoint(t, pkg, "POST", "/orders").Responses {
		got = append(got, response.Status)
	}
	if want := []string{"200", "201", "4XX", "default"}; !reflect.DeepEqual(got, want) {
		t.Errorf("statuses = %v, want %v", got, want)
	}
}

func TestResolve_ResponseMerge(t *testing.T) {
	pkg := resolvePackage(t, shapesPkg)
	e := endpoint(t, pkg, "GET", "/mixed")

	var got []string
	for _, response := range e.Responses {
		got = append(got, response.Status)
	}
	// Declared statuses sorted first, then the inline ones sorted.
	if want := []string{"200", "500", "404"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("statuses = %v, want %v", got, want)
	}

	// The declared 200 wins over the inline one of the same status.
	if e.Responses[0].Description != "From the block" {
		t.Errorf("200 description = %q, want the declared one", e.Responses[0].Description)
	}
	if e.Responses[0].Content == nil || e.Responses[0].Content.Ref == nil {
		t.Error("200 should carry the declared body reference")
	}

	// @contentType empty leaves a response without content.
	if e.Responses[1].ContentType != "" || e.Responses[1].Content != nil {
		t.Errorf("500 = %+v, want no content", e.Responses[1])
	}

	// An inline response with no @description gets a generated one.
	if e.Responses[2].Description != "Response for status 404" {
		t.Errorf("404 description = %q", e.Responses[2].Description)
	}
}

func TestResolve_ResponseHeaders(t *testing.T) {
	pkg := resolvePackage(t, validPkg)

	response := endpoint(t, pkg, "GET", "/users/{id}").Responses[0]
	if names := fieldNames(response.Headers); !reflect.DeepEqual(names, []string{"X-RateLimit-Limit"}) {
		t.Errorf("headers = %v", names)
	}
}

func TestResolve_RepeatedStatusLastWins(t *testing.T) {
	pkg := resolvePackage(t, validPkg)

	responses := endpoint(t, pkg, "GET", "/repeat").Responses
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	if responses[0].Description != "second" {
		t.Errorf("description = %q, want the last declaration", responses[0].Description)
	}
}

func TestResolve_ContentTypeDefaulting(t *testing.T) {
	valid := resolvePackage(t, validPkg)
	create := endpoint(t, valid, "POST", "/users")

	// An explicit @contentType wins.
	if create.Request.ContentType != "application/xml" {
		t.Errorf("request content type = %q, want the declared one", create.Request.ContentType)
	}
	// Otherwise the API default applies; that package declares json.
	if create.Responses[0].ContentType != "application/json" {
		t.Errorf("201 content type = %q, want the API default", create.Responses[0].ContentType)
	}
	// A bodyless response keeps whatever it declared, without defaulting.
	if create.Responses[1].ContentType != "" {
		t.Errorf("204 content type = %q, want empty", create.Responses[1].ContentType)
	}

	// With no API default, JSON is the fallback.
	shapes := resolvePackage(t, shapesPkg)
	if got := endpoint(t, shapes, "GET", "/shapes/{id}").Responses[0].ContentType; got != "application/json" {
		t.Errorf("content type = %q, want the JSON fallback", got)
	}
}

func TestResolve_BodyReferences(t *testing.T) {
	pkg := resolvePackage(t, validPkg)

	created := endpoint(t, pkg, "POST", "/users").Responses[0]
	if created.Content.Ref == nil || !created.Content.Ref.IsArray || created.Content.Ref.Schema != "User" {
		t.Errorf("201 body ref = %+v, want an array of User", created.Content.Ref)
	}
}

func TestResolve_Bind(t *testing.T) {
	pkg := resolvePackage(t, shapesPkg)
	e := endpoint(t, pkg, "POST", "/shapes")

	// A known wrapper resolves to the schema the generator inlines.
	bind := e.Responses[0].Content.Bind
	if bind == nil || bind.Wrapper == nil || bind.Wrapper.Name != "Address" || bind.Field != "Street" {
		t.Errorf("response bind = %+v, want the Address wrapper", bind)
	}

	// An unknown wrapper keeps its name for the validator and falls back to
	// emitting the plain body.
	bind = e.Request.Content.Bind
	if bind == nil || bind.Wrapper != nil || bind.Name != "Missing" {
		t.Errorf("request bind = %+v, want an unresolved wrapper named Missing", bind)
	}
}

func TestResolve_EndpointMetadata(t *testing.T) {
	pkg := resolvePackage(t, validPkg)

	e := endpoint(t, pkg, "GET", "/users/{id}")
	if e.OperationID != "getUser" || e.Summary != "Get a user" || e.Auth != "bearerAuth" {
		t.Errorf("endpoint metadata = %+v", e)
	}
	if !reflect.DeepEqual(e.Tags, []string{"users", "admin"}) {
		t.Errorf("tags = %v", e.Tags)
	}
	if e.Deprecated {
		t.Error("GET /users/{id} should not be deprecated")
	}
	if !endpoint(t, pkg, "POST", "/users").Deprecated {
		t.Error("POST /users should be deprecated")
	}
}

func TestResolve_APIPassThrough(t *testing.T) {
	parsed, err := parser.Parse(validPkg)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := Resolve(parsed)
	if err != nil {
		t.Fatal(err)
	}

	if pkg.API != parsed.API {
		t.Error("API metadata should be passed through untouched")
	}
	if pkg.Name != "valid" {
		t.Errorf("package name = %q", pkg.Name)
	}
}

func TestResolve_Deterministic(t *testing.T) {
	for _, dir := range []string{validPkg, shapesPkg} {
		first := resolvePackage(t, dir)
		second := resolvePackage(t, dir)

		if !reflect.DeepEqual(first, second) {
			t.Errorf("%s: two resolutions differ", dir)
		}
	}
}

func TestResolve_NonStructSchema(t *testing.T) {
	parsed, err := parser.Parse("./testdata/badschema")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, err = Resolve(parsed)
	if err == nil {
		t.Fatal("a @schema on a non-struct alias should fail to resolve")
	}

	var positioned *parser.Error
	if !errors.As(err, &positioned) {
		t.Fatalf("error should carry a position: %v", err)
	}
	if positioned.Pos.Filename == "" {
		t.Errorf("error position is empty: %v", err)
	}
}

func TestResolve_UnknownReferences(t *testing.T) {
	parsed, err := parser.Parse("./testdata/badrefs")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, err = Resolve(parsed)
	if err == nil {
		t.Fatal("references to undeclared structs should fail to resolve")
	}

	// Both endpoints report: resolution accumulates per endpoint.
	message := err.Error()
	for _, want := range []string{"@query references unknown parameter struct: Ghost",
		"@header references unknown parameter struct: GhostHeaders"} {
		if !strings.Contains(message, want) {
			t.Errorf("error %q should mention %q", message, want)
		}
	}

	var positioned *parser.Error
	if !errors.As(err, &positioned) {
		t.Fatalf("error should carry a position: %v", err)
	}
	if positioned.Pos.Filename == "" {
		t.Errorf("error position is empty: %v", err)
	}
}

func TestResolve_WithoutGoTypes(t *testing.T) {
	if _, err := Resolve(&parser.Package{Name: "empty"}); err == nil {
		t.Error("resolving without loaded Go types should fail")
	}
}
