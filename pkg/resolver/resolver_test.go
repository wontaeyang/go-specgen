package resolver

import (
	"go/types"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/parser"
)

// newTestResolver parses a package and returns a resolver over it, which is
// the only way to build one now that the parser owns package loading.
func newTestResolver(t *testing.T, packagePath string) *Resolver {
	t.Helper()

	parsed, err := parser.Parse(packagePath)
	if err != nil {
		t.Fatalf("parse %s: %v", packagePath, err)
	}
	return NewResolver(parsed)
}

func TestNewResolver(t *testing.T) {
	resolver := newTestResolver(t, "../parser/testdata")

	if resolver == nil {
		t.Fatal("NewResolver() returned nil")
	}

	if resolver.pkg == nil {
		t.Error("Resolver.pkg is nil")
	}

	if resolver.schemaNames == nil {
		t.Error("Resolver.schemaNames is nil")
	}
}

func TestResolver_Resolve(t *testing.T) {
	// Parse the test package first
	// Create resolver
	resolver := newTestResolver(t, "../parser/testdata")

	// Resolve
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if resolved == nil {
		t.Fatal("Resolve() returned nil")
	}

	// Verify API
	if resolved.API == nil {
		t.Error("API not resolved")
	}

	// Verify schemas
	if len(resolved.Schemas) == 0 {
		t.Error("No schemas resolved")
	}

	// Verify parameters
	if len(resolved.Parameters) == 0 {
		t.Error("No parameters resolved")
	}

	// Verify endpoints
	if len(resolved.Endpoints) == 0 {
		t.Error("No endpoints resolved")
	}
}

func TestResolver_ResolveSchema(t *testing.T) {
	// Parse the test package first
	// Create resolver
	resolver := newTestResolver(t, "../parser/testdata")

	// Get User schema
	userSchema, ok := resolver.parsed.Schemas["User"]
	if !ok {
		t.Fatal("User schema not found in parsed package")
	}

	// Resolve it
	resolved, err := resolver.resolveSchema(userSchema)
	if err != nil {
		t.Fatalf("resolveSchema() error = %v", err)
	}

	if resolved == nil {
		t.Fatal("resolveSchema() returned nil")
	}

	if resolved.GoTypeName != "User" {
		t.Errorf("GoTypeName = %q, want %q", resolved.GoTypeName, "User")
	}

	// Verify fields were resolved
	if len(resolved.Fields) == 0 {
		t.Error("No fields resolved")
	}

	// Check specific fields
	var idField, emailField *Field
	for _, field := range resolved.Fields {
		switch field.GoName {
		case "ID":
			idField = field
		case "Email":
			emailField = field
		}
	}

	if idField == nil {
		t.Fatal("ID field not found")
	}

	if idField.Type.ScalarName() != "string" {
		t.Errorf("ID field OpenAPIType = %q, want %q", idField.Type.ScalarName(), "string")
	}

	if idField.Format != "uuid" {
		t.Errorf("ID field Format = %q, want %q", idField.Format, "uuid")
	}

	if emailField == nil {
		t.Fatal("Email field not found")
	}

	if emailField.Type.ScalarName() != "string" {
		t.Errorf("Email field OpenAPIType = %q, want %q", emailField.Type.ScalarName(), "string")
	}

	if emailField.Format != "email" {
		t.Errorf("Email field Format = %q, want %q", emailField.Format, "email")
	}
}

func TestResolver_ResolveTypeRef(t *testing.T) {
	resolver := newTestResolver(t, "../parser/testdata")

	// User is a @schema, so a field of that type is a reference rather than a
	// repeat of the definition.
	obj := resolver.pkg.Types.Scope().Lookup("User")
	if obj == nil {
		t.Fatal("User type not found")
	}

	ref := resolver.resolveTypeRef(obj.Type())
	if ref.Shape != ShapeRef {
		t.Fatalf("Shape = %v, want ShapeRef", ref.Shape)
	}
	if ref.Ref != "User" {
		t.Errorf("Ref = %q, want %q", ref.Ref, "User")
	}
}

func TestResolver_ResolveAPI(t *testing.T) {
	resolver := newTestResolver(t, "../parser/testdata")

	api := &parser.APIInfo{
		Title:   "Test API",
		Version: "1.0.0",
		Contact: &parser.Contact{
			Name:  "Test",
			Email: "test@example.com",
		},
		Servers: []*parser.Server{
			{URL: "https://api.example.com", Description: "Production"},
		},
	}

	resolved := resolver.resolveAPI(api)

	if resolved.Title != "Test API" {
		t.Errorf("Title = %q, want %q", resolved.Title, "Test API")
	}

	if resolved.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", resolved.Version, "1.0.0")
	}

	if resolved.Contact == nil {
		t.Fatal("Contact is nil")
	}

	if resolved.Contact.Name != "Test" {
		t.Errorf("Contact.Name = %q, want %q", resolved.Contact.Name, "Test")
	}

	if len(resolved.Servers) != 1 {
		t.Fatalf("Expected 1 server, got %d", len(resolved.Servers))
	}

	if resolved.Servers[0].URL != "https://api.example.com" {
		t.Errorf("Server URL = %q, want %q", resolved.Servers[0].URL, "https://api.example.com")
	}
}

func TestResolver_ResolveParameter(t *testing.T) {
	// Parse the test package first
	// Create resolver
	resolver := newTestResolver(t, "../parser/testdata")

	// Find a parameter (UserIDPath)
	var param *parser.Parameter
	for _, p := range resolver.parsed.Parameters {
		if p.GoTypeName == "UserIDPath" {
			param = p
			break
		}
	}

	if param == nil {
		t.Skip("UserIDPath parameter not found in testdata")
	}

	// Resolve it
	resolved, err := resolver.resolveParameter(param)
	if err != nil {
		t.Fatalf("resolveParameter() error = %v", err)
	}

	if resolved == nil {
		t.Fatal("resolveParameter() returned nil")
	}

	if resolved.GoTypeName != "UserIDPath" {
		t.Errorf("GoTypeName = %q, want %q", resolved.GoTypeName, "UserIDPath")
	}

	// Verify fields
	if len(resolved.Fields) == 0 {
		t.Error("No fields resolved")
	}
}

func TestOmitsWhenEmpty(t *testing.T) {
	structType := types.NewStruct(nil, nil)

	tests := []struct {
		name      string
		tag       string
		fieldType types.Type
		want      bool
	}{
		{
			name:      "no omit options keeps field",
			tag:       `json:"name"`,
			fieldType: types.Typ[types.String],
			want:      false,
		},
		{
			name:      "omitempty on string omits",
			tag:       `json:"name,omitempty"`,
			fieldType: types.Typ[types.String],
			want:      true,
		},
		{
			name:      "omitempty on struct never omits",
			tag:       `json:"created_at,omitempty"`,
			fieldType: structType,
			want:      false,
		},
		{
			name:      "omitzero on struct omits",
			tag:       `json:"created_at,omitzero"`,
			fieldType: structType,
			want:      true,
		},
		{
			name:      "omitempty on pointer to struct omits",
			tag:       `json:"created_at,omitempty"`,
			fieldType: types.NewPointer(structType),
			want:      true,
		},
		{
			name:      "omitempty on slice omits",
			tag:       `json:"tags,omitempty"`,
			fieldType: types.NewSlice(types.Typ[types.String]),
			want:      true,
		},
		{
			name:      "omitempty on non-zero-length array never omits",
			tag:       `json:"coords,omitempty"`,
			fieldType: types.NewArray(types.Typ[types.Float64], 2),
			want:      false,
		},
		{
			name:      "omitempty on zero-length array omits",
			tag:       `json:"empty,omitempty"`,
			fieldType: types.NewArray(types.Typ[types.Float64], 0),
			want:      true,
		},

		// The options belong to a serialization tag. Another tag carrying the
		// same word means something of its own -- validator's omitempty is a
		// validation rule -- and used to mark the field optional from across
		// the raw tag string.
		{
			name:      "omitempty in a foreign tag does not omit",
			tag:       `json:"name" validate:"omitempty"`,
			fieldType: types.Typ[types.String],
			want:      false,
		},
		{
			name:      "omitzero in a foreign tag does not omit",
			tag:       `json:"name" db:"omitzero"`,
			fieldType: types.Typ[types.String],
			want:      false,
		},
		{
			name:      "omitempty in the json tag still omits alongside other tags",
			tag:       `json:"name,omitempty" validate:"required"`,
			fieldType: types.Typ[types.String],
			want:      true,
		},
		{
			name:      "a field literally named omitempty does not omit",
			tag:       `json:"omitempty"`,
			fieldType: types.Typ[types.String],
			want:      false,
		},
		{
			name:      "no serialization tag does not omit",
			tag:       `validate:"omitempty"`,
			fieldType: types.Typ[types.String],
			want:      false,
		},

		// encoding/xml names fields here too. Both options are honored under
		// every supported tag, so xml:",omitzero" omits even though
		// encoding/xml does not implement the option itself.
		{
			name:      "omitempty on an xml-only field omits",
			tag:       `xml:"name,omitempty"`,
			fieldType: types.Typ[types.String],
			want:      true,
		},
		{
			name:      "omitzero on an xml-only field omits",
			tag:       `xml:"name,omitzero"`,
			fieldType: types.Typ[types.String],
			want:      true,
		},
		{
			name:      "json wins over xml when both are present",
			tag:       `json:"name" xml:"name,omitempty"`,
			fieldType: types.Typ[types.String],
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := omitsWhenEmpty(tt.tag, tt.fieldType)
			if got != tt.want {
				t.Errorf("omitsWhenEmpty(%q, %s) = %v, want %v", tt.tag, tt.fieldType, got, tt.want)
			}
		})
	}
}

func TestResolver_ResolveEndpoint(t *testing.T) {
	// Parse the test package first
	// Create resolver
	resolver := newTestResolver(t, "../parser/testdata")

	// Resolve schemas first
	schemas := make(map[string]*Schema)
	for name, schema := range resolver.parsed.Schemas {
		resolved, err := resolver.resolveSchema(schema)
		if err != nil {
			t.Fatalf("Failed to resolve schema %s: %v", name, err)
		}
		schemas[name] = resolved
	}

	// Resolve parameters
	parameters := make(map[string]*ParameterStruct)
	for name, param := range resolver.parsed.Parameters {
		resolved, err := resolver.resolveParameter(param)
		if err != nil {
			t.Fatalf("Failed to resolve parameter %s: %v", name, err)
		}
		parameters[name] = resolved
	}

	// Get first endpoint
	if len(resolver.parsed.Endpoints) == 0 {
		t.Skip("No endpoints in testdata")
	}

	endpoint := resolver.parsed.Endpoints[0]

	// Resolve it
	resolved, err := resolver.resolveEndpoint(endpoint, parameters, schemas, "")
	if err != nil {
		t.Fatalf("resolveEndpoint() error = %v", err)
	}

	if resolved == nil {
		t.Fatal("resolveEndpoint() returned nil")
	}

	if resolved.Method != endpoint.Method {
		t.Errorf("Method = %q, want %q", resolved.Method, endpoint.Method)
	}

	if resolved.Path != endpoint.Path {
		t.Errorf("Path = %q, want %q", resolved.Path, endpoint.Path)
	}

	// Verify responses exist
	if len(resolved.Responses) == 0 && len(endpoint.Responses) > 0 {
		t.Error("Responses not resolved")
	}
}

func TestResolver_BasicTypeRefs(t *testing.T) {
	resolver := newTestResolver(t, "../parser/testdata")

	tests := []struct {
		kind       types.BasicKind
		wantType   string
		wantFormat string
	}{
		{types.String, "string", ""},
		{types.Bool, "boolean", ""},
		{types.Int, "integer", ""},
		{types.Int32, "integer", "int32"},
		{types.Int64, "integer", "int64"},
		{types.Uint64, "integer", ""},
		{types.Float32, "number", "float"},
		{types.Float64, "number", "double"},
	}

	for _, tt := range tests {
		t.Run(types.Typ[tt.kind].Name(), func(t *testing.T) {
			ref := resolver.resolveTypeRef(types.Typ[tt.kind])
			if ref.Shape != ShapeScalar {
				t.Fatalf("Shape = %v, want ShapeScalar", ref.Shape)
			}
			if ref.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", ref.Type, tt.wantType)
			}
			if ref.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", ref.Format, tt.wantFormat)
			}
		})
	}
}

// TestResolver_UnsupportedTypeRefs pins that types encoding/json refuses to
// marshal are marked rather than substituted for something plausible.
func TestResolver_UnsupportedTypeRefs(t *testing.T) {
	resolver := newTestResolver(t, "../parser/testdata")

	for _, kind := range []types.BasicKind{types.Complex64, types.Complex128, types.Uintptr} {
		t.Run(types.Typ[kind].Name(), func(t *testing.T) {
			if ref := resolver.resolveTypeRef(types.Typ[kind]); ref.Shape != ShapeUnsupported {
				t.Errorf("Shape = %v, want ShapeUnsupported", ref.Shape)
			}
		})
	}

	chanType := types.NewChan(types.SendRecv, types.Typ[types.Int])
	if ref := resolver.resolveTypeRef(chanType); ref.Shape != ShapeUnsupported {
		t.Errorf("chan Shape = %v, want ShapeUnsupported", ref.Shape)
	}
}

// TestResolver_NestedContainerTypeRefs pins that nesting survives, which the
// flat IsArray/ItemsType pair could not express.
func TestResolver_NestedContainerTypeRefs(t *testing.T) {
	resolver := newTestResolver(t, "../parser/testdata")

	// map[string][]int64
	nested := types.NewMap(types.Typ[types.String], types.NewSlice(types.Typ[types.Int64]))

	ref := resolver.resolveTypeRef(nested)
	if ref.Shape != ShapeMap {
		t.Fatalf("Shape = %v, want ShapeMap", ref.Shape)
	}
	if ref.Elem.Shape != ShapeArray {
		t.Fatalf("Elem.Shape = %v, want ShapeArray", ref.Elem.Shape)
	}
	if ref.Elem.Elem.Type != "integer" || ref.Elem.Elem.Format != "int64" {
		t.Errorf("innermost = %q/%q, want integer/int64", ref.Elem.Elem.Type, ref.Elem.Elem.Format)
	}
}

func TestResolver_ByteSliceResolvesToStringByte(t *testing.T) {
	resolver := newTestResolver(t, "../parser/testdata")

	// Construct a []byte type using go/types
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	typeInfo := resolver.resolveTypeRef(byteSlice)

	if typeInfo.ScalarName() != "string" {
		t.Errorf("[]byte OpenAPIType = %q, want %q", typeInfo.ScalarName(), "string")
	}
	if typeInfo.Format != "byte" {
		t.Errorf("[]byte Format = %q, want %q", typeInfo.Format, "byte")
	}
	if typeInfo.IsArray() {
		t.Error("[]byte should not be an array")
	}
}

func TestResolver_InlineDeclarations(t *testing.T) {
	// Parse the inline example package
	// Create resolver with comments (for inline resolution)
	resolver := newTestResolver(t, "../../examples/inline")

	// Resolve
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Find GetUser endpoint
	var getUserEndpoint *Endpoint
	for _, ep := range resolved.Endpoints {
		if ep.FuncName == "GetUser" {
			getUserEndpoint = ep
			break
		}
	}

	if getUserEndpoint == nil {
		t.Fatal("GetUser endpoint not found")
	}

	// Inline path parameters land in the merged list, tagged with their location.
	var idField *Field
	for _, param := range getUserEndpoint.Parameters {
		if param.In == "path" && param.Field.Name == "id" {
			idField = param.Field
			break
		}
	}

	if idField == nil {
		t.Fatal("GetUser should have an inline path parameter named id")
	}
	if idField.Type.ScalarName() != "string" {
		t.Errorf("Path field type = %q, want %q", idField.Type.ScalarName(), "string")
	}
	if idField.Description != "User ID" {
		t.Errorf("Path field description = %q, want %q", idField.Description, "User ID")
	}

	// Inline responses land in the merged, sorted response list.
	var resp200 *InlineBody
	for _, resp := range getUserEndpoint.Responses {
		if resp.StatusCode == "200" {
			resp200 = resp.Inline
			break
		}
	}

	if resp200 == nil {
		t.Fatal("GetUser should have an inline 200 response")
	}
	if len(resp200.Fields) != 3 {
		t.Errorf("Response 200 has %d fields, want 3", len(resp200.Fields))
	}
	fieldNames := make(map[string]bool)
	for _, f := range resp200.Fields {
		fieldNames[f.Name] = true
	}
	for _, want := range []string{"id", "email", "name"} {
		if !fieldNames[want] {
			t.Errorf("Response 200 should have %q field", want)
		}
	}

	// Find ListUsers endpoint
	var listUsersEndpoint *Endpoint
	for _, ep := range resolved.Endpoints {
		if ep.FuncName == "ListUsers" {
			listUsersEndpoint = ep
			break
		}
	}

	if listUsersEndpoint == nil {
		t.Fatal("ListUsers endpoint not found")
	}

	// Inline query parameters, likewise.
	var queryFields []*Field
	for _, param := range listUsersEndpoint.Parameters {
		if param.In == "query" {
			queryFields = append(queryFields, param.Field)
		}
	}

	if len(queryFields) != 3 {
		t.Errorf("ListUsers has %d query parameters, want 3", len(queryFields))
	}
	for _, f := range queryFields {
		if f.Name != "limit" {
			continue
		}
		if f.Type.ScalarName() != "integer" {
			t.Errorf("limit field type = %q, want %q", f.Type.ScalarName(), "integer")
		}
		if f.Minimum == nil || *f.Minimum != 1 {
			t.Error("limit field should have minimum=1")
		}
		if f.Maximum == nil || *f.Maximum != 100 {
			t.Error("limit field should have maximum=100")
		}
	}
}

func TestParseInlineDeclaration(t *testing.T) {
	tests := []struct {
		name            string
		annotationType  string
		lines           []string
		expectedCT      string
		expectedBind    string
		expectedDesc    string
		expectedHeaders []string
		expectNil       bool
	}{
		{
			name:           "request with short name json",
			annotationType: "request",
			lines:          []string{"@request { @contentType json }"},
			expectedCT:     "json",
		},
		{
			name:           "request with full MIME type",
			annotationType: "request",
			lines:          []string{"@request { @contentType application/vnd.api+json }"},
			expectedCT:     "application/vnd.api+json",
		},
		{
			name:           "request with no content type",
			annotationType: "request",
			lines:          []string{"@request"},
			expectedCT:     "",
		},
		{
			name:           "nil comment",
			annotationType: "request",
			lines:          nil,
			expectNil:      true,
		},
		{
			name:           "response multiline annotation",
			annotationType: "response",
			lines:          []string{"@response 201 {", "  @contentType json", "}"},
			expectedCT:     "json",
		},
		{
			name:           "response with bind on same line",
			annotationType: "response",
			lines:          []string{"@response 200 { @bind DataResponse.Data }"},
			expectedBind:   "DataResponse.Data",
		},
		{
			name:           "response with content type and bind",
			annotationType: "response",
			lines:          []string{"@response 200 { @contentType json @bind DataResponse.Data }"},
			expectedCT:     "json",
			expectedBind:   "DataResponse.Data",
		},
		{
			name:           "response multiline with bind",
			annotationType: "response",
			lines:          []string{"@response 200 {", "  @bind APIResponse.Payload", "}"},
			expectedBind:   "APIResponse.Payload",
		},
		{
			name:           "request with bind",
			annotationType: "request",
			lines:          []string{"@request { @bind RequestWrapper.Body }"},
			expectedBind:   "RequestWrapper.Body",
		},
		{
			name:           "response with description",
			annotationType: "response",
			lines:          []string{"@response 200 { @description User found }"},
			expectedDesc:   "User found",
		},
		{
			name:            "response with header",
			annotationType:  "response",
			lines:           []string{"@response 200 { @header RateLimitHeaders }"},
			expectedHeaders: []string{"RateLimitHeaders"},
		},
		{
			name:            "response with multiple headers",
			annotationType:  "response",
			lines:           []string{"@response 200 {", "  @header RateLimitHeaders", "  @header CacheHeaders", "}"},
			expectedHeaders: []string{"RateLimitHeaders", "CacheHeaders"},
		},
		{
			name:            "response with all annotations",
			annotationType:  "response",
			lines:           []string{"@response 200 { @contentType json @header RateLimitHeaders @description Users found @bind APIResponse.Data }"},
			expectedCT:      "json",
			expectedBind:    "APIResponse.Data",
			expectedDesc:    "Users found",
			expectedHeaders: []string{"RateLimitHeaders"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var comment *parser.CommentBlock
			if tt.lines != nil {
				comment = &parser.CommentBlock{Lines: tt.lines}
			}

			result, err := ParseInlineDeclaration(comment, tt.annotationType)
			if err != nil {
				t.Fatalf("ParseInlineDeclaration() error = %v", err)
			}

			if tt.expectNil {
				if result != nil {
					t.Errorf("ParseInlineDeclaration() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("ParseInlineDeclaration() = nil, want non-nil")
			}

			// Check content type
			ct := result.GetChildValue("@contentType")
			if ct != tt.expectedCT {
				t.Errorf("contentType = %q, want %q", ct, tt.expectedCT)
			}

			// Check bind
			bind := result.GetChildValue("@bind")
			if bind != tt.expectedBind {
				t.Errorf("bind = %q, want %q", bind, tt.expectedBind)
			}

			// Check description
			desc := result.GetChildValue("@description")
			if desc != tt.expectedDesc {
				t.Errorf("description = %q, want %q", desc, tt.expectedDesc)
			}

			// Check headers
			headers := result.GetRepeatedChildren("@header")
			if len(headers) != len(tt.expectedHeaders) {
				t.Errorf("headers count = %d, want %d", len(headers), len(tt.expectedHeaders))
			} else {
				for i, h := range headers {
					if h.Value != tt.expectedHeaders[i] {
						t.Errorf("header[%d] = %q, want %q", i, h.Value, tt.expectedHeaders[i])
					}
				}
			}
		})
	}
}

func TestContentTypePrecedence(t *testing.T) {
	tests := []struct {
		name             string
		explicitType     string
		defaultType      string
		expectedRequest  string
		expectedResponse string
	}{
		{
			name:             "explicit overrides default",
			explicitType:     "application/xml",
			defaultType:      "application/json",
			expectedRequest:  "application/xml",
			expectedResponse: "application/xml",
		},
		{
			name:             "default used when no explicit",
			explicitType:     "",
			defaultType:      "application/xml",
			expectedRequest:  "application/xml",
			expectedResponse: "application/xml",
		},
		{
			name:             "fallback to json when no explicit or default",
			explicitType:     "",
			defaultType:      "",
			expectedRequest:  "application/json",
			expectedResponse: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := newTestResolver(t, "../parser/testdata")

			endpoint := &parser.Endpoint{
				Method: "POST",
				Path:   "/test",
				Request: &parser.RequestBody{
					ContentType: tt.explicitType,
					Body: &parser.Body{
						Schema: "TestBody",
					},
				},
				Responses: map[string]*parser.Response{
					"200": {
						StatusCode:  "200",
						ContentType: tt.explicitType,
						Body: &parser.Body{
							Schema: "TestBody",
						},
					},
				},
			}

			resolved, err := resolver.resolveEndpoint(endpoint, map[string]*ParameterStruct{}, map[string]*Schema{}, tt.defaultType)
			if err != nil {
				t.Fatalf("resolveEndpoint() error = %v", err)
			}

			if resolved.Request != nil && resolved.Request.ContentType != tt.expectedRequest {
				t.Errorf("Request.ContentType = %q, want %q", resolved.Request.ContentType, tt.expectedRequest)
			}

			for _, resp := range resolved.Responses {
				if resp.StatusCode == "200" && resp.ContentType != tt.expectedResponse {
					t.Errorf("Response.ContentType = %q, want %q", resp.ContentType, tt.expectedResponse)
				}
			}
		})
	}
}

func TestResolver_PointerImpliesNotRequired(t *testing.T) {
	// Parse the test package
	// Create resolver
	resolver := newTestResolver(t, "../parser/testdata")

	// Get FieldRequiredTest schema
	schema, ok := resolver.parsed.Schemas["FieldRequiredTest"]
	if !ok {
		t.Fatal("FieldRequiredTest schema not found in parsed package")
	}

	// Resolve it
	resolved, err := resolver.resolveSchema(schema)
	if err != nil {
		t.Fatalf("resolveSchema() error = %v", err)
	}

	// Build field lookup
	fields := make(map[string]*Field)
	for _, f := range resolved.Fields {
		fields[f.Name] = f
	}

	tests := []struct {
		fieldName    string
		wantRequired bool
		wantNullable bool
	}{
		{"value", true, false},           // string: required, not nullable
		{"value_ptr", true, true},        // *string: required, nullable
		{"value_omit", false, false},     // string,omitempty: not required, not nullable
		{"value_ptr_omit", false, false}, // *string,omitempty: omitted when nil, never null on the wire
		{"value_zero", false, false},     // string,omitzero: not required, not nullable
		{"value_ptr_zero", false, false}, // *string,omitzero: omitted when nil, never null on the wire
		{"value_struct", true, false},    // User: required, not nullable
		{"value_struct_ptr", true, true}, // *User: required, nullable

		// Slices and maps follow the same tag-based rule. Note: a nil slice/map
		// without omitempty marshals as null, but non-nullable is the deliberate
		// default (most handlers guarantee non-nil); use @nullable true to opt in.
		{"value_slice", true, false},           // []string: required, not nullable (documented gap)
		{"value_slice_omit", false, false},     // []string,omitempty: optional, never null on the wire
		{"value_slice_ptr", true, true},        // *[]string: required, nullable
		{"value_slice_ptr_omit", false, false}, // *[]string,omitempty: optional, nil ptr omitted
		{"value_map", true, false},             // map: required, not nullable (documented gap)
		{"value_map_omit", false, false},       // map,omitempty: optional, never null on the wire
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			f, ok := fields[tt.fieldName]
			if !ok {
				t.Fatalf("field %q not found", tt.fieldName)
			}
			if f.Required != tt.wantRequired {
				t.Errorf("field %q Required = %v, want %v", tt.fieldName, f.Required, tt.wantRequired)
			}
			if f.Nullable != tt.wantNullable {
				t.Errorf("field %q Nullable = %v, want %v", tt.fieldName, f.Nullable, tt.wantNullable)
			}
		})
	}
}

func TestResolver_EmbeddedStructFlattening(t *testing.T) {
	// Parse the test package
	// Create resolver
	resolver := newTestResolver(t, "../parser/testdata")

	// Get EmbeddedTest schema
	schema, ok := resolver.parsed.Schemas["EmbeddedTest"]
	if !ok {
		t.Fatal("EmbeddedTest schema not found in parsed package")
	}

	// Resolve it
	resolved, err := resolver.resolveSchema(schema)
	if err != nil {
		t.Fatalf("resolveSchema() error = %v", err)
	}

	// Build field lookup
	fields := make(map[string]*Field)
	for _, f := range resolved.Fields {
		fields[f.Name] = f
	}

	// Should have 5 fields: id, created_at, updated_at (from BaseModel), name, email
	expectedFields := []string{"id", "created_at", "updated_at", "name", "email"}
	for _, name := range expectedFields {
		if _, ok := fields[name]; !ok {
			t.Errorf("expected field %q not found in resolved fields", name)
		}
	}

	if len(resolved.Fields) != len(expectedFields) {
		t.Errorf("expected %d fields, got %d", len(expectedFields), len(resolved.Fields))
		for _, f := range resolved.Fields {
			t.Logf("  field: %s (GoName: %s)", f.Name, f.GoName)
		}
	}

	// Verify embedded fields have correct types
	if f, ok := fields["id"]; ok {
		if f.Type.ScalarName() != "string" {
			t.Errorf("id field OpenAPIType = %q, want %q", f.Type.ScalarName(), "string")
		}
	}
}

func TestResolver_EmbeddedPtrFlattening(t *testing.T) {
	resolver := newTestResolver(t, "../parser/testdata")

	schema, ok := resolver.parsed.Schemas["EmbeddedPtrTest"]
	if !ok {
		t.Fatal("EmbeddedPtrTest schema not found in parsed package")
	}

	resolved, err := resolver.resolveSchema(schema)
	if err != nil {
		t.Fatalf("resolveSchema() error = %v", err)
	}

	fields := make(map[string]*Field)
	for _, f := range resolved.Fields {
		fields[f.Name] = f
	}

	// Should have 4 fields: id, created_at, updated_at (from *BaseModel), label
	expectedFields := []string{"id", "created_at", "updated_at", "label"}
	for _, name := range expectedFields {
		if _, ok := fields[name]; !ok {
			t.Errorf("expected field %q not found in resolved fields", name)
		}
	}

	if len(resolved.Fields) != len(expectedFields) {
		t.Errorf("expected %d fields, got %d", len(expectedFields), len(resolved.Fields))
		for _, f := range resolved.Fields {
			t.Logf("  field: %s (GoName: %s)", f.Name, f.GoName)
		}
	}
}

func TestResolver_NestedEmbedFlattening(t *testing.T) {
	resolver := newTestResolver(t, "../parser/testdata")

	schema, ok := resolver.parsed.Schemas["NestedEmbedTest"]
	if !ok {
		t.Fatal("NestedEmbedTest schema not found in parsed package")
	}

	resolved, err := resolver.resolveSchema(schema)
	if err != nil {
		t.Fatalf("resolveSchema() error = %v", err)
	}

	fields := make(map[string]*Field)
	for _, f := range resolved.Fields {
		fields[f.Name] = f
	}

	// Should have fields from BaseModel (id, created_at, updated_at),
	// Auditable (deleted_at, deleted_by), and status
	expectedFields := []string{"id", "created_at", "updated_at", "deleted_at", "deleted_by", "status"}
	for _, name := range expectedFields {
		if _, ok := fields[name]; !ok {
			t.Errorf("expected field %q not found in resolved fields", name)
		}
	}

	if len(resolved.Fields) != len(expectedFields) {
		t.Errorf("expected %d fields, got %d", len(expectedFields), len(resolved.Fields))
		for _, f := range resolved.Fields {
			t.Logf("  field: %s (GoName: %s)", f.Name, f.GoName)
		}
	}

	// Verify pointer+omitempty fields from Auditable are optional but not nullable
	if f, ok := fields["deleted_at"]; ok {
		if f.Nullable {
			t.Error("deleted_at should not be nullable (omitempty omits nil instead of encoding null)")
		}
		if f.Required {
			t.Error("deleted_at should not be required (omitempty)")
		}
	}
}

func TestResolver_EmbeddedParameterFlattening(t *testing.T) {
	resolver := newTestResolver(t, "../parser/testdata")

	// Find EmbeddedQueryParams parameter
	var param *parser.Parameter
	for _, p := range resolver.parsed.Parameters {
		if p.GoTypeName == "EmbeddedQueryParams" {
			param = p
			break
		}
	}

	if param == nil {
		t.Fatal("EmbeddedQueryParams parameter not found in parsed package")
	}

	resolved, err := resolver.resolveParameter(param)
	if err != nil {
		t.Fatalf("resolveParameter() error = %v", err)
	}

	fields := make(map[string]*Field)
	for _, f := range resolved.Fields {
		fields[f.Name] = f
	}

	// Should have 3 fields: limit, offset (from CommonQueryParams), search
	expectedFields := []string{"limit", "offset", "search"}
	for _, name := range expectedFields {
		if _, ok := fields[name]; !ok {
			t.Errorf("expected field %q not found in resolved fields", name)
		}
	}

	if len(resolved.Fields) != len(expectedFields) {
		t.Errorf("expected %d fields, got %d", len(expectedFields), len(resolved.Fields))
		for _, f := range resolved.Fields {
			t.Logf("  field: %s (GoName: %s)", f.Name, f.GoName)
		}
	}

	// Verify embedded fields are resolved as query parameters (not required by default)
	if f, ok := fields["limit"]; ok {
		if f.Required {
			t.Error("limit should not be required (omitempty)")
		}
		if f.Type.ScalarName() != "integer" {
			t.Errorf("limit OpenAPIType = %q, want %q", f.Type.ScalarName(), "integer")
		}
	}
}

func TestResolveFieldNameFromTag(t *testing.T) {
	tests := []struct {
		name        string
		tag         string
		goFieldName string
		want        string
	}{
		{
			name:        "json tag only",
			tag:         `json:"user_id"`,
			goFieldName: "UserID",
			want:        "user_id",
		},
		{
			name:        "xml tag only (no json)",
			tag:         `xml:"UserName"`,
			goFieldName: "Name",
			want:        "UserName",
		},
		{
			name:        "both tags - json takes priority",
			tag:         `json:"email" xml:"EmailAddress"`,
			goFieldName: "Email",
			want:        "email",
		},
		{
			name:        "no tags - uses Go field name",
			tag:         ``,
			goFieldName: "Age",
			want:        "Age",
		},
		{
			name:        "json skip - returns empty (skip field)",
			tag:         `json:"-"`,
			goFieldName: "Hidden",
			want:        "",
		},
		{
			name:        "json skip with xml - still skips (respects -)",
			tag:         `json:"-" xml:"ShouldNotUse"`,
			goFieldName: "Hidden",
			want:        "",
		},
		{
			name:        "xml skip only (no json) - returns empty (skip field)",
			tag:         `xml:"-"`,
			goFieldName: "Hidden",
			want:        "",
		},
		{
			name:        "json empty - falls back to xml",
			tag:         `json:"" xml:"from_xml"`,
			goFieldName: "Field",
			want:        "from_xml",
		},
		{
			name:        "json omitempty only - falls back to xml",
			tag:         `json:",omitempty" xml:"from_xml"`,
			goFieldName: "Field",
			want:        "from_xml",
		},
		{
			name:        "json omitempty only (no xml) - uses Go field name",
			tag:         `json:",omitempty"`,
			goFieldName: "OptionalField",
			want:        "OptionalField",
		},
		{
			name:        "json with omitempty - uses json name",
			tag:         `json:"field_name,omitempty"`,
			goFieldName: "FieldName",
			want:        "field_name",
		},
		{
			name:        "xml with attr option - uses xml name",
			tag:         `xml:"FieldName,attr"`,
			goFieldName: "Field",
			want:        "FieldName",
		},
		{
			name:        "json name with xml takes priority",
			tag:         `json:"json_name" xml:"XmlName"`,
			goFieldName: "Field",
			want:        "json_name",
		},
		{
			name:        "both empty - uses Go field name",
			tag:         `json:"" xml:""`,
			goFieldName: "MyField",
			want:        "MyField",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveFieldNameFromTag(tt.tag, tt.goFieldName)
			if got != tt.want {
				t.Errorf("resolveFieldNameFromTag(%q, %q) = %q, want %q", tt.tag, tt.goFieldName, got, tt.want)
			}
		})
	}
}

// applyFieldAnnotations has been removed — @field parsing now happens in the
// parser via parseFieldComments and is validated by convertParsedField. Equivalent
// invalid-numeric coverage lives in pkg/parser/parser_test.go:
//   - TestParser_ConvertParsedField_InvalidFloat
//   - TestParser_ConvertParsedField_InvalidInt

func TestApplyAnnotationOverrides_RequiredNullable(t *testing.T) {
	bptr := func(b bool) *bool { return &b }

	tests := []struct {
		name          string
		startRequired bool
		startNullable bool
		annotation    *parser.Field
		wantRequired  bool
		wantNullable  bool
	}{
		{
			name:          "no override leaves values alone",
			startRequired: true,
			startNullable: false,
			annotation:    &parser.Field{},
			wantRequired:  true,
			wantNullable:  false,
		},
		{
			name:          "@required false overrides required=true (non-pointer optional)",
			startRequired: true,
			startNullable: false,
			annotation:    &parser.Field{Required: bptr(false)},
			wantRequired:  false,
			wantNullable:  false,
		},
		{
			name:          "@required true overrides required=false (pointer required)",
			startRequired: false,
			startNullable: true,
			annotation:    &parser.Field{Required: bptr(true)},
			wantRequired:  true,
			wantNullable:  true,
		},
		{
			name:          "@nullable true on non-pointer",
			startRequired: true,
			startNullable: false,
			annotation:    &parser.Field{Nullable: bptr(true)},
			wantRequired:  true,
			wantNullable:  true,
		},
		{
			name:          "@nullable false on pointer",
			startRequired: false,
			startNullable: true,
			annotation:    &parser.Field{Nullable: bptr(false)},
			wantRequired:  false,
			wantNullable:  false,
		},
		{
			name:          "both overrides apply independently",
			startRequired: true,
			startNullable: false,
			annotation:    &parser.Field{Required: bptr(false), Nullable: bptr(true)},
			wantRequired:  false,
			wantNullable:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := &Field{
				Required: tt.startRequired,
				Nullable: tt.startNullable,
			}
			applyAnnotationOverrides(resolved, tt.annotation)
			if resolved.Required != tt.wantRequired {
				t.Errorf("Required = %v, want %v", resolved.Required, tt.wantRequired)
			}
			if resolved.Nullable != tt.wantNullable {
				t.Errorf("Nullable = %v, want %v", resolved.Nullable, tt.wantNullable)
			}
		})
	}
}

// TestResponseDescription covers the fallback both response forms share. A
// named @response used to render an empty description here while an in-function
// one rendered "Response for status N".
func TestResponseDescription(t *testing.T) {
	tests := []struct {
		name       string
		declared   string
		statusCode string
		want       string
	}{
		{"@description wins", "User found", "200", "User found"},
		{"@description wins over a phrase that exists", "Created the widget", "201", "Created the widget"},
		{"200", "", "200", "OK"},
		{"201", "", "201", "Created"},
		{"204", "", "204", "No Content"},
		{"404", "", "404", "Not Found"},
		{"422", "", "422", "Unprocessable Entity"},
		{"default has no phrase", "", "default", "Default response"},
		{"a range has no phrase", "", "4XX", "Response for status 4XX"},
		{"an unassigned code has no phrase", "", "299", "Response for status 299"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := responseDescription(tt.declared, tt.statusCode); got != tt.want {
				t.Errorf("responseDescription(%q, %q) = %q, want %q", tt.declared, tt.statusCode, got, tt.want)
			}
		})
	}
}

// TestResolver_InlineRequestDecodeRule covers applyDecodeRule end to end: the
// raw facts resolveField records, the recursion into anonymous objects and
// container elements, override re-application, and the two places the rule
// must not reach (a referenced named schema, an inline response).
func TestResolver_InlineRequestDecodeRule(t *testing.T) {
	resolver := newTestResolver(t, "testdata/decode")
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(resolved.Endpoints) != 1 {
		t.Fatalf("got %d endpoints, want 1", len(resolved.Endpoints))
	}
	endpoint := resolved.Endpoints[0]
	if endpoint.Request == nil || endpoint.Request.Inline == nil {
		t.Fatal("Create should have an inline request")
	}

	byName := func(fields []*Field) map[string]*Field {
		m := make(map[string]*Field, len(fields))
		for _, f := range fields {
			m[f.Name] = f
		}
		return m
	}
	check := func(t *testing.T, fields map[string]*Field, name string, wantRequired, wantNullable bool) {
		t.Helper()
		f, ok := fields[name]
		if !ok {
			t.Fatalf("field %q not found", name)
		}
		if f.Required != wantRequired || f.Nullable != wantNullable {
			t.Errorf("field %q Required/Nullable = %v/%v, want %v/%v",
				name, f.Required, f.Nullable, wantRequired, wantNullable)
		}
	}

	req := byName(endpoint.Request.Inline.Fields)

	t.Run("top level", func(t *testing.T) {
		check(t, req, "plain", true, false)
		check(t, req, "ptr", false, false)
		check(t, req, "omit", true, false) // omitempty is an encoding fact
		check(t, req, "ptr_omit", false, false)
		check(t, req, "slice", true, false) // nil-able, but not a pointer
		check(t, req, "slice_ptr", false, false)
		check(t, req, "tag", true, false)
	})

	t.Run("overrides win", func(t *testing.T) {
		check(t, req, "optional", false, false)
		check(t, req, "nullable", false, true)
		check(t, req, "forced", true, true)
	})

	t.Run("nested object", func(t *testing.T) {
		nested := byName(req["nested"].Type.Fields)
		check(t, nested, "inner", true, false)
		check(t, nested, "inner_ptr", false, false)
	})

	t.Run("array element", func(t *testing.T) {
		items := byName(req["items"].Type.Elem.Fields)
		check(t, items, "item", true, false)
		check(t, items, "item_ptr", false, false)
	})

	t.Run("map value", func(t *testing.T) {
		vals := byName(req["by_key"].Type.Elem.Fields)
		check(t, vals, "val", true, false)
		check(t, vals, "val_ptr", false, false)
	})

	t.Run("referenced schema keeps marshal rule", func(t *testing.T) {
		if req["tag"].Type.Shape != ShapeRef {
			t.Fatalf("tag shape = %v, want ShapeRef", req["tag"].Type.Shape)
		}
		tag := byName(resolved.Schemas["Tag"].Fields)
		check(t, tag, "label", true, true)
	})

	t.Run("inline response keeps marshal rule", func(t *testing.T) {
		var body *InlineBody
		for _, resp := range endpoint.Responses {
			if resp.StatusCode == "201" {
				body = resp.Inline
			}
		}
		if body == nil {
			t.Fatal("Create should have an inline 201 response")
		}
		created := byName(body.Fields)
		check(t, created, "id", true, false)
		check(t, created, "alias", true, true)
	})
}
