package validator

import (
	"errors"
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/parser"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
)

// The validator is tested against hand-built IR: every rule gets the smallest
// package that breaks it. Assertions check the path and one stable keyword of
// the message, never its full prose.

func stringField(name string) *resolver.Field {
	return &resolver.Field{
		Name:     name,
		GoName:   strings.ToUpper(name[:1]) + name[1:],
		Required: true,
		Type:     resolver.TypeInfo{OpenAPI: "string"},
	}
}

// validPackage is a package that breaks no rule. Tests mutate a copy of it.
func validPackage() *resolver.Package {
	user := &resolver.Schema{Name: "User", Fields: []*resolver.Field{stringField("id")}}

	return &resolver.Package{
		Name:    "test",
		API:     &parser.APIInfo{Title: "Test", Version: "1.0.0"},
		Schemas: []*resolver.Schema{user},
		Endpoints: []*resolver.Endpoint{{
			Method: "GET",
			Path:   "/users/{id}",
			Parameters: []*resolver.Param{
				{In: parser.ParamPath, Field: stringField("id")},
			},
			Responses: []*resolver.Response{{
				Status:      "200",
				Description: "OK",
				ContentType: "application/json",
				Content:     &resolver.Content{Ref: &resolver.TypeRef{Schema: "User"}},
			}},
		}},
	}
}

// validationErrors returns the reported errors, failing when there are none.
func validationErrors(t *testing.T, err error) []*ValidationError {
	t.Helper()

	if err == nil {
		t.Fatal("expected validation to fail")
	}

	var multi *MultiError
	if !errors.As(err, &multi) {
		t.Fatalf("expected a *MultiError, got %T", err)
	}

	reported := make([]*ValidationError, 0, len(multi.Errors))
	for _, err := range multi.Errors {
		var single *ValidationError
		if !errors.As(err, &single) {
			t.Fatalf("expected a *ValidationError, got %T", err)
		}
		reported = append(reported, single)
	}
	return reported
}

// assertReported asserts that some error was reported at path with a message
// mentioning keyword.
func assertReported(t *testing.T, err error, path, keyword string) {
	t.Helper()

	for _, reported := range validationErrors(t, err) {
		if reported.Path == path && strings.Contains(reported.Message, keyword) {
			return
		}
	}
	t.Errorf("no error at %q mentioning %q; got %v", path, keyword, err)
}

func TestValidate_ValidPackage(t *testing.T) {
	if err := Validate(validPackage()); err != nil {
		t.Errorf("valid package should pass: %v", err)
	}
}

func TestValidate_MissingAPI(t *testing.T) {
	pkg := validPackage()
	pkg.API = nil

	// The endpoints are still checked; only the metadata rules are skipped.
	assertReported(t, Validate(pkg), "", "missing @api")
}

func TestValidate_APIRequiredFields(t *testing.T) {
	pkg := validPackage()
	pkg.API = &parser.APIInfo{}

	err := Validate(pkg)
	assertReported(t, err, "@api", "@title")
	assertReported(t, err, "@api", "@version")
}

func TestValidate_SecuritySchemes(t *testing.T) {
	tests := []struct {
		name    string
		scheme  *parser.SecurityScheme
		keyword string
	}{
		{"missing type", &parser.SecurityScheme{Name: "s"}, "@type"},
		{"http without scheme", &parser.SecurityScheme{Name: "s", Type: "http"}, "@scheme"},
		{"apiKey without in", &parser.SecurityScheme{Name: "s", Type: "apiKey", ParameterName: "X-Key"}, "@in"},
		{"apiKey without name", &parser.SecurityScheme{Name: "s", Type: "apiKey", In: "header"}, "@name"},
		{"unknown type", &parser.SecurityScheme{Name: "s", Type: "magic"}, "unknown security scheme type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := validPackage()
			pkg.API.SecuritySchemes = []*parser.SecurityScheme{tt.scheme}

			assertReported(t, Validate(pkg), "@api.@securityScheme[s]", tt.keyword)
		})
	}
}

func TestValidate_SecuritySchemesWithoutExtraFields(t *testing.T) {
	pkg := validPackage()
	pkg.API.SecuritySchemes = []*parser.SecurityScheme{
		{Name: "oauth", Type: "oauth2"},
		{Name: "oidc", Type: "openIdConnect"},
		{Name: "bearer", Type: "http", Scheme: "bearer"},
		{Name: "key", Type: "apiKey", In: "header", ParameterName: "X-Key"},
	}

	if err := Validate(pkg); err != nil {
		t.Errorf("complete schemes should pass: %v", err)
	}
}

func TestValidate_SecurityRequirementReferences(t *testing.T) {
	pkg := validPackage()
	pkg.API.SecuritySchemes = []*parser.SecurityScheme{
		{Name: "bearerAuth", Type: "http", Scheme: "bearer"},
	}
	pkg.API.Security = [][]*parser.SecurityRequirement{{
		{SchemeName: "bearerAuth"},
		{SchemeName: "ghost"},
	}}

	assertReported(t, Validate(pkg), "@api.@security[0].@with[1]", "unknown security scheme")
}

func TestValidate_SchemaWithoutFields(t *testing.T) {
	pkg := validPackage()
	pkg.Schemas = append(pkg.Schemas, &resolver.Schema{Name: "Empty"})

	assertReported(t, Validate(pkg), "@schema[Empty]", "no fields")
}

func TestValidate_DuplicateFieldNames(t *testing.T) {
	pkg := validPackage()
	pkg.Schemas[0].Fields = append(pkg.Schemas[0].Fields, stringField("id"))

	assertReported(t, Validate(pkg), "@schema[User]", "duplicate field name")
}

func TestValidate_UnresolvedStructReference(t *testing.T) {
	pkg := validPackage()
	pkg.Schemas[0].Fields[0].Unresolved = "Address"

	assertReported(t, Validate(pkg), "@schema[User].Id", "not a @schema")
}

func TestValidate_FieldConstraints(t *testing.T) {
	one, two := 1, 2
	oneF, twoF := 1.0, 2.0

	tests := []struct {
		name    string
		field   *resolver.Field
		keyword string
	}{
		{
			name:    "enum on a boolean",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{OpenAPI: "boolean"}, Constraints: resolver.Constraints{Enum: []string{"yes"}}},
			keyword: "enum only supported",
		},
		{
			name:    "enum on an array of objects",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{IsArray: true, Items: "object"}, Constraints: resolver.Constraints{Enum: []string{"a"}}},
			keyword: "enum for arrays",
		},
		{
			name:    "minimum above maximum",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{OpenAPI: "integer"}, Constraints: resolver.Constraints{Minimum: &twoF, Maximum: &oneF}},
			keyword: "minimum cannot be greater",
		},
		{
			name:    "minLength above maxLength",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{OpenAPI: "string"}, Constraints: resolver.Constraints{MinLength: &two, MaxLength: &one}},
			keyword: "minLength cannot be greater",
		},
		{
			name:    "length on a non-string",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{OpenAPI: "integer"}, Constraints: resolver.Constraints{MinLength: &one}},
			keyword: "minLength/maxLength only valid",
		},
		{
			name:    "minItems above maxItems",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{IsArray: true, Items: "string"}, Constraints: resolver.Constraints{MinItems: &two, MaxItems: &one}},
			keyword: "minItems cannot be greater",
		},
		{
			name:    "items on a non-array",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{OpenAPI: "string"}, Constraints: resolver.Constraints{MaxItems: &one}},
			keyword: "minItems/maxItems only valid",
		},
		{
			name:    "uniqueItems on a non-array",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{OpenAPI: "string"}, Constraints: resolver.Constraints{UniqueItems: true}},
			keyword: "uniqueItems only valid",
		},
		{
			name:    "pattern on a non-string",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{OpenAPI: "integer"}, Constraints: resolver.Constraints{Pattern: "^a$"}},
			keyword: "pattern only valid",
		},
		{
			name:    "pattern that does not compile",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{OpenAPI: "string"}, Constraints: resolver.Constraints{Pattern: "[unterminated"}},
			keyword: "invalid pattern regex",
		},
		{
			name:    "readOnly and writeOnly together",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{OpenAPI: "string"}, Constraints: resolver.Constraints{ReadOnly: true, WriteOnly: true}},
			keyword: "readOnly and writeOnly",
		},
		{
			name:    "constraints on a map",
			field:   &resolver.Field{Name: "f", GoName: "F", Type: resolver.TypeInfo{IsMap: true, MapValue: "string"}, Constraints: resolver.Constraints{MinLength: &one}},
			keyword: "minLength/maxLength only valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := validPackage()
			pkg.Schemas[0].Fields = []*resolver.Field{tt.field}

			assertReported(t, Validate(pkg), "@schema[User].F", tt.keyword)
		})
	}
}

func TestValidate_AllowedFieldConstraints(t *testing.T) {
	one, two := 1, 2

	fields := []*resolver.Field{
		{Name: "a", GoName: "A", Type: resolver.TypeInfo{OpenAPI: "string"}, Constraints: resolver.Constraints{Enum: []string{"x"}, MinLength: &one, MaxLength: &two, Pattern: "^x$"}},
		{Name: "b", GoName: "B", Type: resolver.TypeInfo{OpenAPI: "integer"}, Constraints: resolver.Constraints{Enum: []string{"1"}}},
		{Name: "c", GoName: "C", Type: resolver.TypeInfo{IsArray: true, Items: "string"}, Constraints: resolver.Constraints{Enum: []string{"x"}, MinItems: &one, MaxItems: &two, UniqueItems: true}},
		{Name: "d", GoName: "D", Type: resolver.TypeInfo{IsArray: true, Items: "integer"}, Constraints: resolver.Constraints{Enum: []string{"1"}}},
	}

	pkg := validPackage()
	pkg.Schemas[0].Fields = fields

	if err := Validate(pkg); err != nil {
		t.Errorf("constraints matching their types should pass: %v", err)
	}
}

func TestValidate_ParameterRules(t *testing.T) {
	array := resolver.TypeInfo{IsArray: true, Items: "string"}

	tests := []struct {
		name    string
		param   *resolver.Param
		keyword string
	}{
		{
			name:    "nullable path parameter",
			param:   &resolver.Param{In: parser.ParamPath, Field: &resolver.Field{Name: "id", GoName: "ID", Required: true, Nullable: true, Type: resolver.TypeInfo{OpenAPI: "string"}}},
			keyword: "cannot be nullable",
		},
		{
			name:    "optional path parameter",
			param:   &resolver.Param{In: parser.ParamPath, Field: &resolver.Field{Name: "id", GoName: "ID", Type: resolver.TypeInfo{OpenAPI: "string"}}},
			keyword: "always required",
		},
		{
			name:    "array path parameter",
			param:   &resolver.Param{In: parser.ParamPath, Field: &resolver.Field{Name: "id", GoName: "ID", Required: true, Type: array}},
			keyword: "path parameters cannot be arrays",
		},
		{
			name:    "array header parameter",
			param:   &resolver.Param{In: parser.ParamHeader, Field: &resolver.Field{Name: "X-Tag", GoName: "Tag", Type: array}},
			keyword: "header parameters cannot be arrays",
		},
		{
			name:    "array cookie parameter",
			param:   &resolver.Param{In: parser.ParamCookie, Field: &resolver.Field{Name: "session", GoName: "Session", Type: array}},
			keyword: "cookie parameters cannot be arrays",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := validPackage()
			endpoint := pkg.Endpoints[0]
			endpoint.Path = "/things"
			endpoint.Parameters = []*resolver.Param{tt.param}

			path := "@endpoint[GET /things].@" + tt.param.In + "." + tt.param.Field.GoName
			assertReported(t, Validate(pkg), path, tt.keyword)
		})
	}
}

func TestValidate_ArrayQueryParameterAllowed(t *testing.T) {
	pkg := validPackage()
	endpoint := pkg.Endpoints[0]
	endpoint.Path = "/things"
	endpoint.Parameters = []*resolver.Param{{
		In:    parser.ParamQuery,
		Field: &resolver.Field{Name: "tags", GoName: "Tags", Type: resolver.TypeInfo{IsArray: true, Items: "string"}},
	}}

	if err := Validate(pkg); err != nil {
		t.Errorf("query parameters may be arrays: %v", err)
	}
}

func TestValidate_InlineParameterFieldsAreChecked(t *testing.T) {
	// Inline declarations resolve into the same parameter list as referenced
	// structs, so their fields are checked the same way.
	pkg := validPackage()
	endpoint := pkg.Endpoints[0]
	endpoint.Path = "/things"
	endpoint.Parameters = []*resolver.Param{{
		In:    parser.ParamQuery,
		Field: &resolver.Field{Name: "limit", GoName: "Limit", Type: resolver.TypeInfo{OpenAPI: "integer"}, Constraints: resolver.Constraints{Pattern: "^1$"}},
	}}

	assertReported(t, Validate(pkg), "@endpoint[GET /things].@query.Limit", "pattern only valid")
}

func TestValidate_EndpointRules(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*resolver.Endpoint)
		path    string
		keyword string
	}{
		{
			name:    "unknown method",
			mutate:  func(e *resolver.Endpoint) { e.Method = "FETCH" },
			path:    "@endpoint[FETCH /users/{id}]",
			keyword: "invalid HTTP method",
		},
		{
			name:    "missing path",
			mutate:  func(e *resolver.Endpoint) { e.Path = ""; e.Parameters = nil },
			path:    "@endpoint[GET ]",
			keyword: "missing path",
		},
		{
			name:    "path without a leading slash",
			mutate:  func(e *resolver.Endpoint) { e.Path = "users/{id}" },
			path:    "@endpoint[GET users/{id}]",
			keyword: "must start with /",
		},
		{
			name:    "invalid path variable name",
			mutate:  func(e *resolver.Endpoint) { e.Path = "/users/{user id}"; e.Parameters = nil },
			path:    "@endpoint[GET /users/{user id}]",
			keyword: "invalid path variable name",
		},
		{
			name:    "path variable without a parameter",
			mutate:  func(e *resolver.Endpoint) { e.Parameters = nil },
			path:    "@endpoint[GET /users/{id}]",
			keyword: "has no corresponding @path parameter",
		},
		{
			name:    "path parameter not in the path",
			mutate:  func(e *resolver.Endpoint) { e.Path = "/users"; e.Parameters[0].Field.Name = "other" },
			path:    "@endpoint[GET /users]",
			keyword: "not used in path",
		},
		{
			name:    "no responses",
			mutate:  func(e *resolver.Endpoint) { e.Responses = nil },
			path:    "@endpoint[GET /users/{id}]",
			keyword: "at least one response",
		},
		{
			name: "parameter name conflict across kinds",
			mutate: func(e *resolver.Endpoint) {
				e.Parameters = append(e.Parameters, &resolver.Param{
					In:    parser.ParamQuery,
					Field: &resolver.Field{Name: "id", GoName: "ID", Type: resolver.TypeInfo{OpenAPI: "string"}},
				})
			},
			path:    "@endpoint[GET /users/{id}]",
			keyword: "parameter name conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := validPackage()
			tt.mutate(pkg.Endpoints[0])

			assertReported(t, Validate(pkg), tt.path, tt.keyword)
		})
	}
}

func TestValidate_DuplicateRoutes(t *testing.T) {
	pkg := validPackage()
	duplicate := *pkg.Endpoints[0]
	pkg.Endpoints = append(pkg.Endpoints, &duplicate)

	assertReported(t, Validate(pkg), "@endpoint[GET /users/{id}]", "duplicate operation")
}

func TestValidate_SamePathDifferentMethods(t *testing.T) {
	pkg := validPackage()
	other := *pkg.Endpoints[0]
	other.Method = "DELETE"
	pkg.Endpoints = append(pkg.Endpoints, &other)

	if err := Validate(pkg); err != nil {
		t.Errorf("one path may carry several methods: %v", err)
	}
}

func TestValidate_DuplicateOperationIDs(t *testing.T) {
	pkg := validPackage()
	pkg.Endpoints[0].OperationID = "getUser"

	other := *pkg.Endpoints[0]
	other.Path = "/people/{id}"
	pkg.Endpoints = append(pkg.Endpoints, &other)

	assertReported(t, Validate(pkg), "@endpoint[GET /people/{id}]", "duplicate @operationID: getUser")
}

func TestValidate_MissingOperationIDsAreNotDuplicates(t *testing.T) {
	pkg := validPackage()
	other := *pkg.Endpoints[0]
	other.Path = "/people/{id}"
	pkg.Endpoints = append(pkg.Endpoints, &other)

	if err := Validate(pkg); err != nil {
		t.Errorf("endpoints without an @operationID are unnamed, not duplicates: %v", err)
	}
}

func TestValidate_EndpointAuth(t *testing.T) {
	pkg := validPackage()
	pkg.API.SecuritySchemes = []*parser.SecurityScheme{
		{Name: "bearerAuth", Type: "http", Scheme: "bearer"},
	}

	t.Run("declared scheme", func(t *testing.T) {
		pkg.Endpoints[0].Auth = "bearerAuth"

		if err := Validate(pkg); err != nil {
			t.Errorf("@auth naming a declared scheme should pass: %v", err)
		}
	})

	t.Run("unknown scheme", func(t *testing.T) {
		pkg.Endpoints[0].Auth = "ghostAuth"

		assertReported(t, Validate(pkg), "@endpoint[GET /users/{id}]", "@auth references unknown security scheme: ghostAuth")
	})
}

func TestValidate_EndpointTags(t *testing.T) {
	pkg := validPackage()
	pkg.API.Tags = []*parser.Tag{{Name: "users"}}
	pkg.Endpoints[0].Tags = []string{"users", "ghosts"}

	assertReported(t, Validate(pkg), "@endpoint[GET /users/{id}]", "undefined tag: ghosts")
}

func TestValidate_TagsUncheckedWithoutAPITags(t *testing.T) {
	pkg := validPackage()
	pkg.Endpoints[0].Tags = []string{"anything"}

	if err := Validate(pkg); err != nil {
		t.Errorf("tags are only checked when the API defines some: %v", err)
	}
}

func TestValidate_RequestBody(t *testing.T) {
	t.Run("missing content type", func(t *testing.T) {
		pkg := validPackage()
		pkg.Endpoints[0].Request = &resolver.Request{
			Content: &resolver.Content{Ref: &resolver.TypeRef{Schema: "User"}},
		}

		assertReported(t, Validate(pkg), "@endpoint[GET /users/{id}].@request", "@contentType")
	})

	t.Run("unknown schema", func(t *testing.T) {
		pkg := validPackage()
		pkg.Endpoints[0].Request = &resolver.Request{
			ContentType: "application/json",
			Content:     &resolver.Content{Ref: &resolver.TypeRef{Schema: "Ghost"}},
		}

		assertReported(t, Validate(pkg), "@endpoint[GET /users/{id}].@request", "unknown schema: Ghost")
	})

	t.Run("primitive body needs no schema", func(t *testing.T) {
		pkg := validPackage()
		pkg.Endpoints[0].Request = &resolver.Request{
			ContentType: "application/json",
			Content:     &resolver.Content{Ref: &resolver.TypeRef{IsMap: true, Primitive: "integer"}},
		}

		if err := Validate(pkg); err != nil {
			t.Errorf("a primitive body should pass: %v", err)
		}
	})
}

func TestValidate_ResponseStatusCodes(t *testing.T) {
	valid := []string{"200", "204", "404", "2XX", "5XX", "default"}
	for _, status := range valid {
		pkg := validPackage()
		pkg.Endpoints[0].Responses[0].Status = status

		if err := Validate(pkg); err != nil {
			t.Errorf("status %q should pass: %v", status, err)
		}
	}

	invalid := []string{"", "20", "2000", "600", "XXX", "abc"}
	for _, status := range invalid {
		pkg := validPackage()
		pkg.Endpoints[0].Responses[0].Status = status

		path := "@endpoint[GET /users/{id}].@response[" + status + "]"
		assertReported(t, Validate(pkg), path, "invalid status code")
	}
}

func TestValidate_ResponseBody(t *testing.T) {
	pkg := validPackage()
	pkg.Endpoints[0].Responses[0].Content.Ref.Schema = "Ghost"

	assertReported(t, Validate(pkg), "@endpoint[GET /users/{id}].@response[200]", "unknown schema: Ghost")
}

func TestValidate_InlineResponseFieldsAreChecked(t *testing.T) {
	one := 1

	pkg := validPackage()
	pkg.Endpoints[0].Responses[0].Content = &resolver.Content{Fields: []*resolver.Field{
		{Name: "count", GoName: "Count", Type: resolver.TypeInfo{OpenAPI: "integer"}, Constraints: resolver.Constraints{MinLength: &one}},
	}}

	assertReported(t, Validate(pkg), "@endpoint[GET /users/{id}].@response[200].Count", "minLength/maxLength only valid")
}

func TestValidate_BindTarget(t *testing.T) {
	wrapper := &resolver.Schema{Name: "Envelope", Fields: []*resolver.Field{
		{Name: "data", GoName: "Data", Type: resolver.TypeInfo{IsAny: true}},
	}}

	t.Run("resolved wrapper and field", func(t *testing.T) {
		pkg := validPackage()
		pkg.Schemas = append(pkg.Schemas, wrapper)
		pkg.Endpoints[0].Responses[0].Content.Bind = &resolver.BindTarget{
			Name: "Envelope", Field: "Data", Wrapper: wrapper,
		}

		if err := Validate(pkg); err != nil {
			t.Errorf("a resolved bind should pass: %v", err)
		}
	})

	t.Run("unknown wrapper", func(t *testing.T) {
		pkg := validPackage()
		pkg.Endpoints[0].Responses[0].Content.Bind = &resolver.BindTarget{Name: "Ghost", Field: "Data"}

		assertReported(t, Validate(pkg), "@endpoint[GET /users/{id}].@response[200].@bind", "unknown wrapper schema: Ghost")
	})

	t.Run("unknown field", func(t *testing.T) {
		pkg := validPackage()
		pkg.Schemas = append(pkg.Schemas, wrapper)
		pkg.Endpoints[0].Responses[0].Content.Bind = &resolver.BindTarget{
			Name: "Envelope", Field: "Missing", Wrapper: wrapper,
		}

		assertReported(t, Validate(pkg), "@endpoint[GET /users/{id}].@response[200].@bind", "no field")
	})
}

func TestMultiError(t *testing.T) {
	single := &MultiError{Errors: []error{&ValidationError{Path: "@api", Message: "boom"}}}
	if got := single.Error(); got != "@api: boom" {
		t.Errorf("single error = %q", got)
	}

	several := &MultiError{Errors: []error{
		&ValidationError{Path: "@api", Message: "boom"},
		&ValidationError{Message: "bang"},
	}}
	message := several.Error()
	for _, want := range []string{"2 validation errors", "1. @api: boom", "2. bang"} {
		if !strings.Contains(message, want) {
			t.Errorf("message %q should contain %q", message, want)
		}
	}
}

func TestMultiError_Unwrap(t *testing.T) {
	pkg := validPackage()
	pkg.API = &parser.APIInfo{}

	err := Validate(pkg)

	var reported *ValidationError
	if !errors.As(err, &reported) {
		t.Fatalf("errors.As should reach a *ValidationError through Unwrap: %v", err)
	}
	if reported.Path != "@api" {
		t.Errorf("first unwrapped error = %+v", reported)
	}
}

func TestValidate_ErrorOrderIsStable(t *testing.T) {
	build := func() *resolver.Package {
		pkg := validPackage()
		pkg.API.Title = ""
		pkg.API.SecuritySchemes = []*parser.SecurityScheme{
			{Name: "zeta"}, {Name: "alpha"},
		}
		pkg.Schemas = append(pkg.Schemas,
			&resolver.Schema{Name: "Zed"},
			&resolver.Schema{Name: "Ann"},
		)
		pkg.Endpoints[0].Method = "FETCH"
		return pkg
	}

	first := Validate(build()).Error()
	second := Validate(build()).Error()
	if first != second {
		t.Errorf("error order is not stable:\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	// Source order, not map order: the schemes and schemas report in the
	// order the package carries them.
	if !strings.Contains(first, "zeta") || strings.Index(first, "zeta") > strings.Index(first, "alpha") {
		t.Errorf("security schemes should report in declaration order:\n%s", first)
	}
	if strings.Index(first, "@schema[Zed]") > strings.Index(first, "@schema[Ann]") {
		t.Errorf("schemas should report in package order:\n%s", first)
	}
}
