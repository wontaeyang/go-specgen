package resolver

import (
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/parser"
)

func TestNewResolver(t *testing.T) {
	resolver, err := NewResolver("../parser/testdata", nil)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	if resolver == nil {
		t.Fatal("NewResolver() returned nil")
	}

	if resolver.pkg == nil {
		t.Error("Resolver.pkg is nil")
	}

	if resolver.typeCache == nil {
		t.Error("Resolver.typeCache is nil")
	}
}

func TestNewResolver_InvalidPath(t *testing.T) {
	_, err := NewResolver("./nonexistent", nil)
	if err == nil {
		t.Error("NewResolver() should error for invalid path")
	}
}

func TestResolver_Resolve(t *testing.T) {
	// Parse the test package first
	p := parser.NewParser("../parser/testdata")
	parsed, err := p.Parse()
	if err != nil {
		t.Fatalf("Failed to parse package: %v", err)
	}

	// Create resolver
	resolver, err := NewResolver("../parser/testdata", nil)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	// Resolve
	resolved, err := resolver.Resolve(parsed)
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
	p := parser.NewParser("../parser/testdata")
	parsed, err := p.Parse()
	if err != nil {
		t.Fatalf("Failed to parse package: %v", err)
	}

	// Create resolver
	resolver, err := NewResolver("../parser/testdata", nil)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	// Get User schema
	userSchema, ok := parsed.Schemas["User"]
	if !ok {
		t.Fatal("User schema not found in parsed package")
	}

	// Build schema names map
	schemaNames := make(map[string]bool)
	for name := range parsed.Schemas {
		schemaNames[name] = true
	}

	// Resolve it
	resolved, err := resolver.resolveSchema(userSchema, schemaNames)
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
	var idField, emailField *ResolvedField
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

	if idField.OpenAPIType != "string" {
		t.Errorf("ID field OpenAPIType = %q, want %q", idField.OpenAPIType, "string")
	}

	if idField.Format != "uuid" {
		t.Errorf("ID field Format = %q, want %q", idField.Format, "uuid")
	}

	if emailField == nil {
		t.Fatal("Email field not found")
	}

	if emailField.OpenAPIType != "string" {
		t.Errorf("Email field OpenAPIType = %q, want %q", emailField.OpenAPIType, "string")
	}

	if emailField.Format != "email" {
		t.Errorf("Email field Format = %q, want %q", emailField.Format, "email")
	}
}

func TestResolver_ResolveType(t *testing.T) {
	resolver, err := NewResolver("../parser/testdata", nil)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	// Get User struct type
	obj := resolver.pkg.Types.Scope().Lookup("User")
	if obj == nil {
		t.Fatal("User type not found")
	}

	// Test resolving the struct type itself (should examine underlying fields)
	typeInfo := resolver.resolveType(obj.Type())
	if typeInfo == nil {
		t.Fatal("resolveType() returned nil")
	}

	// The type system should resolve string fields
	if typeInfo.OpenAPIType == "" {
		t.Error("OpenAPIType is empty")
	}
}

func TestResolver_ResolveAPI(t *testing.T) {
	resolver, err := NewResolver("../parser/testdata", nil)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

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

func TestExtractJSONName(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "simple json tag",
			tag:      `json:"id"`,
			expected: "id",
		},
		{
			name:     "json tag with omitempty",
			tag:      `json:"email,omitempty"`,
			expected: "email",
		},
		{
			name:     "json tag with dash",
			tag:      `json:"-"`,
			expected: "-",
		},
		{
			name:     "no json tag",
			tag:      `validate:"required"`,
			expected: "",
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: "",
		},
		{
			name:     "multiple tags",
			tag:      `json:"name" validate:"required"`,
			expected: "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONName(tt.tag)
			if got != tt.expected {
				t.Errorf("extractJSONName(%q) = %q, want %q", tt.tag, got, tt.expected)
			}
		})
	}
}

func TestResolver_ResolveParameter(t *testing.T) {
	// Parse the test package first
	p := parser.NewParser("../parser/testdata")
	parsed, err := p.Parse()
	if err != nil {
		t.Fatalf("Failed to parse package: %v", err)
	}

	// Create resolver
	resolver, err := NewResolver("../parser/testdata", nil)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	// Find a parameter (UserIDPath)
	var param *parser.Parameter
	for _, p := range parsed.Parameters {
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

func TestResolver_ResolveEndpoint(t *testing.T) {
	// Parse the test package first
	p := parser.NewParser("../parser/testdata")
	parsed, err := p.Parse()
	if err != nil {
		t.Fatalf("Failed to parse package: %v", err)
	}

	// Create resolver
	resolver, err := NewResolver("../parser/testdata", nil)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	// Build schema names map
	schemaNames := make(map[string]bool)
	for name := range parsed.Schemas {
		schemaNames[name] = true
	}

	// Resolve schemas first
	schemas := make(map[string]*ResolvedSchema)
	for name, schema := range parsed.Schemas {
		resolved, err := resolver.resolveSchema(schema, schemaNames)
		if err != nil {
			t.Fatalf("Failed to resolve schema %s: %v", name, err)
		}
		schemas[name] = resolved
	}

	// Resolve parameters
	parameters := make(map[string]*ResolvedParameter)
	for name, param := range parsed.Parameters {
		resolved, err := resolver.resolveParameter(param)
		if err != nil {
			t.Fatalf("Failed to resolve parameter %s: %v", name, err)
		}
		parameters[name] = resolved
	}

	// Get first endpoint
	if len(parsed.Endpoints) == 0 {
		t.Skip("No endpoints in testdata")
	}

	endpoint := parsed.Endpoints[0]

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

func TestTypeInfo_BasicTypes(t *testing.T) {
	resolver, err := NewResolver("../parser/testdata", nil)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	// We would need to construct Go types for testing
	// For now, just verify resolver was created
	if resolver.typeCache == nil {
		t.Error("typeCache not initialized")
	}
}

func TestResolver_InlineDeclarations(t *testing.T) {
	// Parse the inline example package
	p := parser.NewParser("../../examples/inline")
	parsed, err := p.Parse()
	if err != nil {
		t.Fatalf("Failed to parse package: %v", err)
	}

	// Create resolver with comments (for inline resolution)
	resolver, err := NewResolver("../../examples/inline", p.Comments())
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	// Resolve
	resolved, err := resolver.Resolve(parsed)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Find GetUser endpoint
	var getUserEndpoint *ResolvedEndpoint
	for _, ep := range resolved.Endpoints {
		if ep.FuncName == "GetUser" {
			getUserEndpoint = ep
			break
		}
	}

	if getUserEndpoint == nil {
		t.Fatal("GetUser endpoint not found")
	}

	// Check inline path params
	if getUserEndpoint.InlinePathParams == nil {
		t.Error("GetUser should have inline path params")
	} else {
		if len(getUserEndpoint.InlinePathParams.Fields) == 0 {
			t.Error("GetUser inline path should have fields")
		} else {
			idField := getUserEndpoint.InlinePathParams.Fields[0]
			if idField.Name != "id" {
				t.Errorf("Path field name = %q, want %q", idField.Name, "id")
			}
			if idField.OpenAPIType != "string" {
				t.Errorf("Path field type = %q, want %q", idField.OpenAPIType, "string")
			}
			if idField.Description != "User ID" {
				t.Errorf("Path field description = %q, want %q", idField.Description, "User ID")
			}
		}
	}

	// Check inline responses
	if getUserEndpoint.InlineResponses == nil {
		t.Error("GetUser should have inline responses")
	} else {
		resp200 := getUserEndpoint.InlineResponses["200"]
		if resp200 == nil {
			t.Error("GetUser should have 200 response")
		} else {
			if len(resp200.Fields) != 3 {
				t.Errorf("Response 200 has %d fields, want 3", len(resp200.Fields))
			}
			// Check field names
			fieldNames := make(map[string]bool)
			for _, f := range resp200.Fields {
				fieldNames[f.Name] = true
			}
			if !fieldNames["id"] {
				t.Error("Response 200 should have 'id' field")
			}
			if !fieldNames["email"] {
				t.Error("Response 200 should have 'email' field")
			}
			if !fieldNames["name"] {
				t.Error("Response 200 should have 'name' field")
			}
		}
	}

	// Find ListUsers endpoint
	var listUsersEndpoint *ResolvedEndpoint
	for _, ep := range resolved.Endpoints {
		if ep.FuncName == "ListUsers" {
			listUsersEndpoint = ep
			break
		}
	}

	if listUsersEndpoint == nil {
		t.Fatal("ListUsers endpoint not found")
	}

	// Check inline query params
	if listUsersEndpoint.InlineQueryParams == nil {
		t.Error("ListUsers should have inline query params")
	} else {
		if len(listUsersEndpoint.InlineQueryParams.Fields) != 3 {
			t.Errorf("ListUsers inline query has %d fields, want 3", len(listUsersEndpoint.InlineQueryParams.Fields))
		}
		// Find limit field and check annotations
		for _, f := range listUsersEndpoint.InlineQueryParams.Fields {
			if f.Name == "limit" {
				if f.OpenAPIType != "integer" {
					t.Errorf("limit field type = %q, want %q", f.OpenAPIType, "integer")
				}
				if f.Minimum == nil || *f.Minimum != 1 {
					t.Error("limit field should have minimum=1")
				}
				if f.Maximum == nil || *f.Maximum != 100 {
					t.Error("limit field should have maximum=100")
				}
			}
		}
	}
}

func TestParseInlineAnnotation(t *testing.T) {
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

			result, err := ParseInlineAnnotation(comment, tt.annotationType)
			if err != nil {
				t.Fatalf("ParseInlineAnnotation() error = %v", err)
			}

			if tt.expectNil {
				if result != nil {
					t.Errorf("ParseInlineAnnotation() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("ParseInlineAnnotation() = nil, want non-nil")
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
			resolver, err := NewResolver("../parser/testdata", nil)
			if err != nil {
				t.Fatalf("NewResolver() error = %v", err)
			}

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

			resolved, err := resolver.resolveEndpoint(endpoint, map[string]*ResolvedParameter{}, map[string]*ResolvedSchema{}, tt.defaultType)
			if err != nil {
				t.Fatalf("resolveEndpoint() error = %v", err)
			}

			if resolved.Request != nil && resolved.Request.ContentType != tt.expectedRequest {
				t.Errorf("Request.ContentType = %q, want %q", resolved.Request.ContentType, tt.expectedRequest)
			}

			if resp, ok := resolved.Responses["200"]; ok {
				if resp.ContentType != tt.expectedResponse {
					t.Errorf("Response.ContentType = %q, want %q", resp.ContentType, tt.expectedResponse)
				}
			}
		})
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
