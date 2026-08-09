package parser

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/annotation"
)

func TestParser_Parse(t *testing.T) {
	// Test full parse of testdata/sample.go
	parsed, err := Parse("./testdata")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	// Verify API info
	if parsed.API == nil {
		t.Fatal("API not parsed")
	}

	if parsed.API.Title != "Test API" {
		t.Errorf("API.Title = %q, want %q", parsed.API.Title, "Test API")
	}

	if parsed.API.Version != "1.0.0" {
		t.Errorf("API.Version = %q, want %q", parsed.API.Version, "1.0.0")
	}

	// Verify schemas
	if len(parsed.Schemas) == 0 {
		t.Error("No schemas parsed")
	}

	// Verify User schema exists
	userSchema, ok := parsed.Schemas["User"]
	if !ok {
		t.Fatal("User schema not found")
	}

	if userSchema.GoTypeName != "User" {
		t.Errorf("User.GoTypeName = %q, want %q", userSchema.GoTypeName, "User")
	}

	if len(userSchema.Fields) == 0 {
		t.Error("User schema has no fields")
	}

	// Verify parameters
	if len(parsed.Parameters) == 0 {
		t.Error("No parameters parsed")
	}

	// Verify endpoints
	if len(parsed.Endpoints) == 0 {
		t.Error("No endpoints parsed")
	}
}

func TestParser_ParseAPI(t *testing.T) {

	// Extract comments first
	comments, err := ExtractComments("./testdata")
	if err != nil {
		t.Fatalf("ExtractComments() error = %v", err)
	}
	p := &parser{comments: comments}

	result := &Package{
		PackageName: "testdata",
		Schemas:     make(map[string]*Schema),
		Parameters:  make(map[string]*Parameter),
		Endpoints:   make([]*Endpoint, 0),
	}

	p.parseAPI(result)
	err = p.errs.Err()
	if err != nil {
		t.Fatalf("parseAPI() error = %v", err)
	}

	if result.API == nil {
		t.Fatal("API not parsed")
	}

	// Check required fields
	if result.API.Title == "" {
		t.Error("API.Title is empty")
	}

	if result.API.Version == "" {
		t.Error("API.Version is empty")
	}
}

func TestParser_ParseAPI_MissingRequired(t *testing.T) {
	// Create a parser with no package comments
	p := &parser{
		comments: &PackageComments{
			PackageComments: nil, // No package-level comments
		},
	}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseAPI(result)
	err := p.errs.Err()
	if err == nil {
		t.Error("parseAPI() should error when @api is missing")
	}
}

func TestParser_ParseSchemas(t *testing.T) {

	// Extract comments first
	comments, err := ExtractComments("./testdata")
	if err != nil {
		t.Fatalf("ExtractComments() error = %v", err)
	}
	p := &parser{comments: comments}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseSchemas(result)
	err = p.errs.Err()
	if err != nil {
		t.Fatalf("parseSchemas() error = %v", err)
	}

	// parseStructFields is the single place that parses @field annotations, so
	// it must be invoked to populate Schema.Fields — identical path to inline
	// var structs.
	p.parseStructFields(result)
	if err := p.errs.Err(); err != nil {
		t.Fatalf("parseStructFields() error = %v", err)
	}

	// Check if User schema was parsed
	userSchema, ok := result.Schemas["User"]
	if !ok {
		t.Fatal("User schema not found")
	}

	// Verify schema has fields
	if len(userSchema.Fields) == 0 {
		t.Error("User schema should have fields")
	}
}

func TestParser_ParseParameters(t *testing.T) {

	// Extract comments first
	comments, err := ExtractComments("./testdata")
	if err != nil {
		t.Fatalf("ExtractComments() error = %v", err)
	}
	p := &parser{comments: comments}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseParameters(result)
	err = p.errs.Err()
	if err != nil {
		t.Fatalf("parseParameters() error = %v", err)
	}

	// Check parameter types
	for name, param := range result.Parameters {
		if param.GoTypeName != name {
			t.Errorf("Parameter %s has GoTypeName = %q, want %q", name, param.GoTypeName, name)
		}

		// Verify type is set correctly
		if param.Type == "" {
			t.Errorf("Parameter %s has empty Type", name)
		}
	}
}

func TestParser_ParseStructFields_InlineStructs(t *testing.T) {
	// Verifies that parseStructFields populates Fields for inline var structs
	// declared inside handler bodies via the same path as @schema structs.
	result, err := Parse("../../examples/inline")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	inlines := result.FuncInlines["ListUsers"]
	if inlines == nil || len(inlines.Query) == 0 {
		t.Fatal("ListUsers should have an inline @query struct")
	}
	query := inlines.Query[0]
	if len(query.Fields) == 0 {
		t.Fatal("inline @query Fields should be populated by parseStructFields")
	}

	want := map[string]struct {
		description string
		hasMinimum  bool
		hasEnum     bool
	}{
		"Limit":  {description: "Maximum results", hasMinimum: true},
		"Offset": {description: "Page offset", hasMinimum: true},
		"Status": {description: "Filter by status", hasEnum: true},
	}

	got := make(map[string]*Field, len(query.Fields))
	for _, f := range query.Fields {
		got[f.GoName] = f
	}

	for name, expected := range want {
		f, ok := got[name]
		if !ok {
			t.Errorf("Field %q not parsed onto info.Fields", name)
			continue
		}
		if f.Description != expected.description {
			t.Errorf("%s.Description = %q, want %q", name, f.Description, expected.description)
		}
		if expected.hasMinimum && f.Minimum == nil {
			t.Errorf("%s should have @minimum populated", name)
		}
		if expected.hasEnum && len(f.Enum) == 0 {
			t.Errorf("%s should have @enum populated", name)
		}
	}
}

func TestParser_ParseEndpoints(t *testing.T) {

	// Extract comments first
	comments, err := ExtractComments("./testdata")
	if err != nil {
		t.Fatalf("ExtractComments() error = %v", err)
	}
	p := &parser{comments: comments}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseEndpoints(result)
	err = p.errs.Err()
	if err != nil {
		t.Fatalf("parseEndpoints() error = %v", err)
	}

	if len(result.Endpoints) == 0 {
		t.Fatal("No endpoints parsed")
	}

	// Check first endpoint
	endpoint := result.Endpoints[0]

	if endpoint.Method == "" {
		t.Error("Endpoint.Method is empty")
	}

	if endpoint.Path == "" {
		t.Error("Endpoint.Path is empty")
	}

	// Verify responses exist
	if len(endpoint.Responses) == 0 {
		t.Error("Endpoint should have responses")
	}
}

func TestParser_ConvertParsedField(t *testing.T) {

	tests := []struct {
		name       string
		fieldName  string
		annotation *ParsedAnnotation
		wantDesc   string
		wantFormat string
	}{
		{
			name:      "simple field",
			fieldName: "Email",
			annotation: &ParsedAnnotation{
				Children: map[string]*ParsedAnnotation{
					"@description": {Value: "User email"},
					"@format":      {Value: "email"},
				},
			},
			wantDesc:   "User email",
			wantFormat: "email",
		},
		{
			name:      "field with example",
			fieldName: "Name",
			annotation: &ParsedAnnotation{
				Children: map[string]*ParsedAnnotation{
					"@description": {Value: "User name"},
					"@example":     {Value: "John Doe"},
				},
			},
			wantDesc:   "User name",
			wantFormat: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, err := convertParsedField(tt.fieldName, tt.annotation)
			if err != nil {
				t.Fatalf("convertParsedField() error = %v", err)
			}

			if field.GoName != tt.fieldName {
				t.Errorf("GoName = %q, want %q", field.GoName, tt.fieldName)
			}

			if field.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", field.Description, tt.wantDesc)
			}

			if field.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", field.Format, tt.wantFormat)
			}
		})
	}
}

func TestParser_ConvertParsedField_Numeric(t *testing.T) {
	annotation := &ParsedAnnotation{
		Children: map[string]*ParsedAnnotation{
			"@minimum":   {Value: "0"},
			"@maximum":   {Value: "100"},
			"@minLength": {Value: "5"},
			"@maxLength": {Value: "50"},
		},
	}

	field, err := convertParsedField("Age", annotation)
	if err != nil {
		t.Fatalf("convertParsedField() error = %v", err)
	}

	if field.Minimum == nil || *field.Minimum != 0 {
		t.Error("Minimum not parsed correctly")
	}

	if field.Maximum == nil || *field.Maximum != 100 {
		t.Error("Maximum not parsed correctly")
	}

	if field.MinLength == nil || *field.MinLength != 5 {
		t.Error("MinLength not parsed correctly")
	}

	if field.MaxLength == nil || *field.MaxLength != 50 {
		t.Error("MaxLength not parsed correctly")
	}
}

func TestParser_ConvertParsedField_Enum(t *testing.T) {
	annotation := &ParsedAnnotation{
		Children: map[string]*ParsedAnnotation{
			"@enum": {Value: "active, inactive, pending"},
		},
	}

	field, err := convertParsedField("Status", annotation)
	if err != nil {
		t.Fatalf("convertParsedField() error = %v", err)
	}

	if len(field.Enum) != 3 {
		t.Fatalf("Enum has %d values, want 3", len(field.Enum))
	}

	expectedEnum := []string{"active", "inactive", "pending"}
	for i, val := range expectedEnum {
		if field.Enum[i] != val {
			t.Errorf("Enum[%d] = %q, want %q", i, field.Enum[i], val)
		}
	}
}

func TestParser_ConvertParsedField_Deprecated(t *testing.T) {
	annotation := &ParsedAnnotation{
		Children: map[string]*ParsedAnnotation{
			"@deprecated": {IsFlag: true},
		},
	}

	field, err := convertParsedField("OldField", annotation)
	if err != nil {
		t.Fatalf("convertParsedField() error = %v", err)
	}

	if !field.Deprecated {
		t.Error("Field should be marked as deprecated")
	}
}

func TestParser_ConvertParsedField_InvalidFloat(t *testing.T) {
	annotation := &ParsedAnnotation{
		Children: map[string]*ParsedAnnotation{
			"@minimum": {Value: "abc"},
		},
	}

	_, err := convertParsedField("BadField", annotation)
	if err == nil {
		t.Fatal("expected error for invalid @minimum value")
	}

	if !strings.Contains(err.Error(), "@minimum") || !strings.Contains(err.Error(), "abc") {
		t.Errorf("error message = %q, want it to mention @minimum and the invalid value", err.Error())
	}
}

func TestParser_ConvertParsedField_InvalidInt(t *testing.T) {
	annotation := &ParsedAnnotation{
		Children: map[string]*ParsedAnnotation{
			"@minLength": {Value: "1.5"},
		},
	}

	_, err := convertParsedField("BadField", annotation)
	if err == nil {
		t.Fatal("expected error for invalid @minLength value")
	}

	if !strings.Contains(err.Error(), "@minLength") || !strings.Contains(err.Error(), "1.5") {
		t.Errorf("error message = %q, want it to mention @minLength and the invalid value", err.Error())
	}
}

func TestParser_ConvertParsedField_ValidNumeric(t *testing.T) {
	annotation := &ParsedAnnotation{
		Children: map[string]*ParsedAnnotation{
			"@minimum":   {Value: "1.5"},
			"@maximum":   {Value: "99.9"},
			"@minLength": {Value: "3"},
			"@maxLength": {Value: "100"},
		},
	}

	field, err := convertParsedField("GoodField", annotation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if field.Minimum == nil || *field.Minimum != 1.5 {
		t.Error("Minimum not parsed correctly")
	}
	if field.Maximum == nil || *field.Maximum != 99.9 {
		t.Error("Maximum not parsed correctly")
	}
	if field.MinLength == nil || *field.MinLength != 3 {
		t.Error("MinLength not parsed correctly")
	}
	if field.MaxLength == nil || *field.MaxLength != 100 {
		t.Error("MaxLength not parsed correctly")
	}
}

func TestParser_ConvertParsedField_RequiredNullableOverrides(t *testing.T) {
	tests := []struct {
		name         string
		annotation   *ParsedAnnotation
		wantRequired *bool
		wantNullable *bool
	}{
		{
			name:         "no overrides",
			annotation:   &ParsedAnnotation{},
			wantRequired: nil,
			wantNullable: nil,
		},
		{
			name: "@required true",
			annotation: &ParsedAnnotation{
				Children: map[string]*ParsedAnnotation{
					"@required": {Value: "true"},
				},
			},
			wantRequired: boolPtr(true),
		},
		{
			name: "@required false",
			annotation: &ParsedAnnotation{
				Children: map[string]*ParsedAnnotation{
					"@required": {Value: "false"},
				},
			},
			wantRequired: boolPtr(false),
		},
		{
			name: "@nullable true",
			annotation: &ParsedAnnotation{
				Children: map[string]*ParsedAnnotation{
					"@nullable": {Value: "true"},
				},
			},
			wantNullable: boolPtr(true),
		},
		{
			name: "@nullable false",
			annotation: &ParsedAnnotation{
				Children: map[string]*ParsedAnnotation{
					"@nullable": {Value: "false"},
				},
			},
			wantNullable: boolPtr(false),
		},
		{
			name: "both set",
			annotation: &ParsedAnnotation{
				Children: map[string]*ParsedAnnotation{
					"@required": {Value: "false"},
					"@nullable": {Value: "true"},
				},
			},
			wantRequired: boolPtr(false),
			wantNullable: boolPtr(true),
		},
		{
			name: "bare @required means true",
			annotation: &ParsedAnnotation{
				Children: map[string]*ParsedAnnotation{
					"@required": {Value: ""},
				},
			},
			wantRequired: boolPtr(true),
		},
		{
			name: "bare @nullable means true",
			annotation: &ParsedAnnotation{
				Children: map[string]*ParsedAnnotation{
					"@nullable": {Value: ""},
				},
			},
			wantNullable: boolPtr(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, err := convertParsedField("F", tt.annotation)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !boolPtrEqual(field.Required, tt.wantRequired) {
				t.Errorf("Required = %v, want %v", boolPtrStr(field.Required), boolPtrStr(tt.wantRequired))
			}
			if !boolPtrEqual(field.Nullable, tt.wantNullable) {
				t.Errorf("Nullable = %v, want %v", boolPtrStr(field.Nullable), boolPtrStr(tt.wantNullable))
			}
		})
	}
}

func TestParser_ConvertParsedField_InvalidBool(t *testing.T) {
	annotation := &ParsedAnnotation{
		Children: map[string]*ParsedAnnotation{
			"@required": {Value: "maybe"},
		},
	}

	_, err := convertParsedField("BadField", annotation)
	if err == nil {
		t.Fatal("expected error for invalid @required value")
	}
	if !strings.Contains(err.Error(), "@required") || !strings.Contains(err.Error(), "maybe") {
		t.Errorf("error message = %q, want mention of @required and invalid value", err.Error())
	}
}

func boolPtr(b bool) *bool { return &b }

func boolPtrEqual(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func boolPtrStr(b *bool) string {
	if b == nil {
		return "nil"
	}
	if *b {
		return "true"
	}
	return "false"
}

func TestParser_ParseEndpoint_Metadata(t *testing.T) {
	// Test that endpoint metadata (method and path) is parsed correctly
	p := &parser{
		comments: &PackageComments{
			FunctionComments: map[string]*CommentBlock{
				"GetUser": {
					Lines: []string{
						"@endpoint GET /users/{id} {",
						"  @summary Get user by ID",
						"}",
					},
				},
			},
		},
	}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseEndpoints(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseEndpoints() error = %v", err)
	}

	if len(result.Endpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(result.Endpoints))
	}

	endpoint := result.Endpoints[0]

	if endpoint.Method != "GET" {
		t.Errorf("Method = %q, want %q", endpoint.Method, "GET")
	}

	if endpoint.Path != "/users/{id}" {
		t.Errorf("Path = %q, want %q", endpoint.Path, "/users/{id}")
	}

	if endpoint.Summary != "Get user by ID" {
		t.Errorf("Summary = %q, want %q", endpoint.Summary, "Get user by ID")
	}
}

func TestParser_ParseEndpoint_Tags(t *testing.T) {
	// Test that tags are parsed from multiple annotations
	p := &parser{
		comments: &PackageComments{
			FunctionComments: map[string]*CommentBlock{
				"GetUser": {
					Lines: []string{
						"@endpoint GET /users {",
						"  @tag users",
						"  @tag admin",
						"  @tag v1",
						"}",
					},
				},
			},
		},
	}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseEndpoints(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseEndpoints() error = %v", err)
	}

	endpoint := result.Endpoints[0]

	if len(endpoint.Tags) != 3 {
		t.Fatalf("Expected 3 tags, got %d", len(endpoint.Tags))
	}

	expectedTags := []string{"users", "admin", "v1"}
	for i, tag := range expectedTags {
		if endpoint.Tags[i] != tag {
			t.Errorf("Tags[%d] = %q, want %q", i, endpoint.Tags[i], tag)
		}
	}
}

func TestParser_ParseEndpoint_Request(t *testing.T) {
	// Test that request body is parsed
	p := &parser{
		comments: &PackageComments{
			FunctionComments: map[string]*CommentBlock{
				"CreateUser": {
					Lines: []string{
						"@endpoint POST /users {",
						"  @request {",
						"    @contentType application/json",
						"    @body CreateUserRequest",
						"  }",
						"}",
					},
				},
			},
		},
	}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseEndpoints(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseEndpoints() error = %v", err)
	}

	endpoint := result.Endpoints[0]

	if endpoint.Request == nil {
		t.Fatal("Request not parsed")
	}

	if endpoint.Request.ContentType != "application/json" {
		t.Errorf("Request.ContentType = %q, want %q", endpoint.Request.ContentType, "application/json")
	}

	if endpoint.Request.Body == nil || endpoint.Request.Body.Schema != "CreateUserRequest" {
		schema := ""
		if endpoint.Request.Body != nil {
			schema = endpoint.Request.Body.Schema
		}
		t.Errorf("Request.Body.Schema = %q, want %q", schema, "CreateUserRequest")
	}
}

func TestParser_ParseEndpoint_Responses(t *testing.T) {
	// Test that multiple responses are parsed
	p := &parser{
		comments: &PackageComments{
			FunctionComments: map[string]*CommentBlock{
				"CreateUser": {
					Lines: []string{
						"@endpoint POST /users {",
						"  @response 200 {",
						"    @contentType application/json",
						"    @body User",
						"    @description Success",
						"  }",
						"  @response 400 {",
						"    @description Bad request",
						"  }",
						"}",
					},
				},
			},
		},
	}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseEndpoints(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseEndpoints() error = %v", err)
	}

	endpoint := result.Endpoints[0]

	if len(endpoint.Responses) != 2 {
		t.Fatalf("Expected 2 responses, got %d", len(endpoint.Responses))
	}

	// Check 200 response
	resp200, ok := endpoint.Responses["200"]
	if !ok {
		t.Fatal("200 response not found")
	}

	if resp200.StatusCode != "200" {
		t.Errorf("Response.StatusCode = %q, want %q", resp200.StatusCode, "200")
	}

	if resp200.ContentType != "application/json" {
		t.Errorf("Response.ContentType = %q, want %q", resp200.ContentType, "application/json")
	}

	if resp200.Body == nil || resp200.Body.Schema != "User" {
		schema := ""
		if resp200.Body != nil {
			schema = resp200.Body.Schema
		}
		t.Errorf("Response.Body.Schema = %q, want %q", schema, "User")
	}

	// Check 400 response
	resp400, ok := endpoint.Responses["400"]
	if !ok {
		t.Fatal("400 response not found")
	}

	if resp400.Description != "Bad request" {
		t.Errorf("Response.Description = %q, want %q", resp400.Description, "Bad request")
	}
}

func TestParser_ParseAPI_Contact(t *testing.T) {
	p := &parser{
		comments: &PackageComments{
			PackageComments: &CommentBlock{
				Lines: []string{
					"@api {",
					"  @title Test API",
					"  @version 1.0.0",
					"  @contact {",
					"    @name API Team",
					"    @email api\\@example.com",
					"    @url https://example.com",
					"  }",
					"}",
				},
			},
		},
	}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseAPI(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseAPI() error = %v", err)
	}

	if result.API.Contact == nil {
		t.Fatal("Contact not parsed")
	}

	if result.API.Contact.Name != "API Team" {
		t.Errorf("Contact.Name = %q, want %q", result.API.Contact.Name, "API Team")
	}

	if result.API.Contact.Email != "api@example.com" {
		t.Errorf("Contact.Email = %q, want %q", result.API.Contact.Email, "api@example.com")
	}

	if result.API.Contact.URL != "https://example.com" {
		t.Errorf("Contact.URL = %q, want %q", result.API.Contact.URL, "https://example.com")
	}
}

func TestParser_ParseAPI_Servers(t *testing.T) {
	p := &parser{
		comments: &PackageComments{
			PackageComments: &CommentBlock{
				Lines: []string{
					"@api {",
					"  @title Test API",
					"  @version 1.0.0",
					"  @server https://api.example.com {",
					"    @description Production",
					"  }",
					"  @server https://staging.example.com {",
					"    @description Staging",
					"  }",
					"}",
				},
			},
		},
	}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseAPI(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseAPI() error = %v", err)
	}

	if len(result.API.Servers) != 2 {
		t.Fatalf("Expected 2 servers, got %d", len(result.API.Servers))
	}

	if result.API.Servers[0].URL != "https://api.example.com" {
		t.Errorf("Server[0].URL = %q, want %q", result.API.Servers[0].URL, "https://api.example.com")
	}

	if result.API.Servers[0].Description != "Production" {
		t.Errorf("Server[0].Description = %q, want %q", result.API.Servers[0].Description, "Production")
	}
}

func TestParser_ParseAPI_SecuritySchemes(t *testing.T) {
	p := &parser{
		comments: &PackageComments{
			PackageComments: &CommentBlock{
				Lines: []string{
					"@api {",
					"  @title Test API",
					"  @version 1.0.0",
					"  @securityScheme bearer {",
					"    @type http",
					"    @scheme bearer",
					"    @bearerFormat JWT",
					"    @description Bearer token",
					"  }",
					"}",
				},
			},
		},
	}

	result := &Package{
		Schemas:    make(map[string]*Schema),
		Parameters: make(map[string]*Parameter),
		Endpoints:  make([]*Endpoint, 0),
	}

	p.parseAPI(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseAPI() error = %v", err)
	}

	if len(result.API.SecuritySchemes) != 1 {
		t.Fatalf("Expected 1 security scheme, got %d", len(result.API.SecuritySchemes))
	}

	scheme, ok := result.API.SecuritySchemes["bearer"]
	if !ok {
		t.Fatal("bearer security scheme not found")
	}

	if scheme.Type != "http" {
		t.Errorf("SecurityScheme.Type = %q, want %q", scheme.Type, "http")
	}

	if scheme.Scheme != "bearer" {
		t.Errorf("SecurityScheme.Scheme = %q, want %q", scheme.Scheme, "bearer")
	}

	if scheme.BearerFormat != "JWT" {
		t.Errorf("SecurityScheme.BearerFormat = %q, want %q", scheme.BearerFormat, "JWT")
	}
}

func TestParser_ParseAPI_DefaultContentType(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected string
	}{
		{
			name: "short name json",
			lines: []string{
				"@api {",
				"  @title Test API",
				"  @version 1.0.0",
				"  @defaultContentType json",
				"}",
			},
			expected: "application/json",
		},
		{
			name: "full MIME type",
			lines: []string{
				"@api {",
				"  @title Test API",
				"  @version 1.0.0",
				"  @defaultContentType application/vnd.api+json",
				"}",
			},
			expected: "application/vnd.api+json",
		},
		{
			name: "no default content type",
			lines: []string{
				"@api {",
				"  @title Test API",
				"  @version 1.0.0",
				"}",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{
				comments: &PackageComments{
					PackageComments: &CommentBlock{
						Lines: tt.lines,
					},
				},
			}

			result := &Package{
				Schemas:    make(map[string]*Schema),
				Parameters: make(map[string]*Parameter),
				Endpoints:  make([]*Endpoint, 0),
			}

			p.parseAPI(result)
			err := p.errs.Err()
			if err != nil {
				t.Fatalf("parseAPI() error = %v", err)
			}

			if result.API.DefaultContentType != tt.expected {
				t.Errorf("DefaultContentType = %q, want %q", result.API.DefaultContentType, tt.expected)
			}
		})
	}
}

func TestParser_ParseBody_Simple(t *testing.T) {
	// Test @body without binding
	p := &parser{
		comments: &PackageComments{
			FunctionComments: map[string]*CommentBlock{
				"GetUser": {
					Lines: []string{
						"@endpoint GET /users/{id} {",
						"  @response 200 {",
						"    @contentType json",
						"    @body User",
						"  }",
						"}",
					},
				},
			},
		},
	}

	result := &Package{
		Endpoints: make([]*Endpoint, 0),
	}

	p.parseEndpoints(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseEndpoints failed: %v", err)
	}

	if len(result.Endpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(result.Endpoints))
	}

	endpoint := result.Endpoints[0]
	resp, ok := endpoint.Responses["200"]
	if !ok {
		t.Fatal("200 response not found")
	}

	if resp.Body == nil {
		t.Fatal("Response body is nil")
	}

	if resp.Body.Schema != "User" {
		t.Errorf("Body.Schema = %q, want %q", resp.Body.Schema, "User")
	}

	if resp.Body.Bind != nil {
		t.Error("Expected Bind to be nil for simple body")
	}
}

func TestParser_ParseBody_WithBind(t *testing.T) {
	// Test @body with @bind using new syntax: @body User @bind DataResponse.Data
	p := &parser{
		comments: &PackageComments{
			FunctionComments: map[string]*CommentBlock{
				"GetUser": {
					Lines: []string{
						"@endpoint GET /me {",
						"  @response 200 {",
						"    @contentType json",
						"    @body User",
						"    @bind DataResponse.Data",
						"  }",
						"}",
					},
				},
			},
		},
	}

	result := &Package{
		Endpoints: make([]*Endpoint, 0),
	}

	p.parseEndpoints(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseEndpoints failed: %v", err)
	}

	if len(result.Endpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(result.Endpoints))
	}

	endpoint := result.Endpoints[0]
	resp, ok := endpoint.Responses["200"]
	if !ok {
		t.Fatal("200 response not found")
	}

	if resp.Body == nil {
		t.Fatal("Response body is nil")
	}

	if resp.Body.Schema != "User" {
		t.Errorf("Body.Schema = %q, want %q", resp.Body.Schema, "User")
	}

	if resp.Body.Bind == nil {
		t.Fatal("Expected Bind to be set")
	}

	if resp.Body.Bind.Wrapper != "DataResponse" {
		t.Errorf("Bind.Wrapper = %q, want %q", resp.Body.Bind.Wrapper, "DataResponse")
	}
	if resp.Body.Bind.Field != "Data" {
		t.Errorf("Bind.Field = %q, want %q", resp.Body.Bind.Field, "Data")
	}
}

func TestParser_ParseBody_WithArrayBind(t *testing.T) {
	// Test @body with array schema: @body []User @bind DataResponse.Data
	p := &parser{
		comments: &PackageComments{
			FunctionComments: map[string]*CommentBlock{
				"ListUsers": {
					Lines: []string{
						"@endpoint GET /users {",
						"  @response 200 {",
						"    @contentType json",
						"    @body []User",
						"    @bind DataResponse.Data",
						"  }",
						"}",
					},
				},
			},
		},
	}

	result := &Package{
		Endpoints: make([]*Endpoint, 0),
	}

	p.parseEndpoints(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseEndpoints failed: %v", err)
	}

	endpoint := result.Endpoints[0]
	resp := endpoint.Responses["200"]

	if resp.Body.Schema != "[]User" {
		t.Errorf("Body.Schema = %q, want %q", resp.Body.Schema, "[]User")
	}

	if resp.Body.Bind == nil {
		t.Fatal("Expected Bind to be set")
	}

	if resp.Body.Bind.Wrapper != "DataResponse" {
		t.Errorf("Bind.Wrapper = %q, want %q", resp.Body.Bind.Wrapper, "DataResponse")
	}
}

func TestParser_ParseBody_Request(t *testing.T) {
	// Test @body in request with @bind
	p := &parser{
		comments: &PackageComments{
			FunctionComments: map[string]*CommentBlock{
				"CreateUser": {
					Lines: []string{
						"@endpoint POST /users {",
						"  @request {",
						"    @contentType json",
						"    @body CreateUserRequest",
						"    @bind RequestWrapper.Payload",
						"  }",
						"  @response 201 {",
						"    @description Created",
						"  }",
						"}",
					},
				},
			},
		},
	}

	result := &Package{
		Endpoints: make([]*Endpoint, 0),
	}

	p.parseEndpoints(result)
	err := p.errs.Err()
	if err != nil {
		t.Fatalf("parseEndpoints failed: %v", err)
	}

	endpoint := result.Endpoints[0]

	if endpoint.Request == nil {
		t.Fatal("Request is nil")
	}

	if endpoint.Request.Body == nil {
		t.Fatal("Request.Body is nil")
	}

	if endpoint.Request.Body.Schema != "CreateUserRequest" {
		t.Errorf("Request.Body.Schema = %q, want %q", endpoint.Request.Body.Schema, "CreateUserRequest")
	}

	if endpoint.Request.Body.Bind == nil {
		t.Fatal("Expected Bind to be set")
	}

	if endpoint.Request.Body.Bind.Wrapper != "RequestWrapper" {
		t.Errorf("Bind.Wrapper = %q, want %q", endpoint.Request.Body.Bind.Wrapper, "RequestWrapper")
	}
	if endpoint.Request.Body.Bind.Field != "Payload" {
		t.Errorf("Bind.Field = %q, want %q", endpoint.Request.Body.Bind.Field, "Payload")
	}
}

func TestParseBindTarget(t *testing.T) {
	tests := []struct {
		input       string
		wantWrapper string
		wantField   string
	}{
		{"DataResponse.Data", "DataResponse", "Data"},
		{"Wrapper.Items", "Wrapper", "Items"},
		{"Response.Payload", "Response", "Payload"},
		{"  Padded.Field  ", "Padded", "Field"},

		// Only the first dot separates, so a nested target keeps its dots.
		{"Wrapper.Outer.Inner", "Wrapper", "Outer.Inner"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bind, err := ParseBindTarget(tt.input)
			if err != nil {
				t.Fatalf("ParseBindTarget(%q) returned error: %v", tt.input, err)
			}

			if bind.Wrapper != tt.wantWrapper {
				t.Errorf("Wrapper = %q, want %q", bind.Wrapper, tt.wantWrapper)
			}
			if bind.Field != tt.wantField {
				t.Errorf("Field = %q, want %q", bind.Field, tt.wantField)
			}
		})
	}
}

// A @bind that names no target is an error, not a nil. nil is indistinguishable
// from "no @bind was written", so returning it dropped the annotation and let
// the run exit 0 with an unwrapped body.
func TestParseBindTarget_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"no separator", "NoSeparator"},
		{"empty", ""},
		{"blank", "   "},
		{"wrapper only", "OnlyWrapper."},
		{"field only", ".OnlyField"},
		{"separator only", "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bind, err := ParseBindTarget(tt.input)
			if err == nil {
				t.Fatalf("ParseBindTarget(%q) = %+v, want an error", tt.input, bind)
			}
			if bind != nil {
				t.Errorf("ParseBindTarget(%q) returned a target alongside its error: %+v", tt.input, bind)
			}
		})
	}
}

func TestSchemaHasBodyAnnotation(t *testing.T) {
	// Verify @body is in schema
	requestNode := annotation.Schema.GetChild("@endpoint").GetChild("@request")
	bodyNode := requestNode.GetChild("@body")

	if bodyNode == nil {
		t.Fatal("@body not found in @request schema")
	}

	if !bodyNode.HasMetadata {
		t.Error("@body should have HasMetadata=true")
	}

	// Verify @bind is a sibling of @body in request/response
	bindNode := requestNode.GetChild("@bind")
	if bindNode == nil {
		t.Fatal("@bind not found in @request schema")
	}
}

func TestExpandContentType(t *testing.T) {
	tests := []struct {
		name      string
		shortName string
		expected  string
	}{
		{
			name:      "json short name",
			shortName: "json",
			expected:  "application/json",
		},
		{
			name:      "form short name",
			shortName: "form",
			expected:  "application/x-www-form-urlencoded",
		},
		{
			name:      "multipart short name",
			shortName: "multipart",
			expected:  "multipart/form-data",
		},
		{
			name:      "text short name",
			shortName: "text",
			expected:  "text/plain",
		},
		{
			name:      "binary short name",
			shortName: "binary",
			expected:  "application/octet-stream",
		},
		{
			name:      "empty short name",
			shortName: "empty",
			expected:  "",
		},
		{
			name:      "xml short name",
			shortName: "xml",
			expected:  "application/xml",
		},
		{
			name:      "csv short name",
			shortName: "csv",
			expected:  "text/csv",
		},
		{
			name:      "html short name",
			shortName: "html",
			expected:  "text/html",
		},
		{
			name:      "uppercase short name",
			shortName: "JSON",
			expected:  "application/json",
		},
		{
			name:      "mixed case short name",
			shortName: "Json",
			expected:  "application/json",
		},
		{
			name:      "full MIME type passed through",
			shortName: "application/json",
			expected:  "application/json",
		},
		{
			name:      "custom MIME type passed through",
			shortName: "application/vnd.api+json",
			expected:  "application/vnd.api+json",
		},
		{
			name:      "unknown short name passed through",
			shortName: "unknown",
			expected:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandContentType(tt.shortName)
			if result != tt.expected {
				t.Errorf("ExpandContentType(%q) = %q, want %q", tt.shortName, result, tt.expected)
			}
		})
	}
}

// TestParse_LogsSkippedAnnotations covers the declarations specgen passes over
// because their doc comment opens with a name it does not recognize.
//
// These are reported rather than raised. A name the grammar has never heard of
// cannot be told apart from prose, so failing the run would reject a package
// that merely documents itself with a line starting @ — but saying nothing is
// how a misspelled @endpoint used to cost a whole operation in silence.
func TestParse_LogsSkippedAnnotations(t *testing.T) {
	var out bytes.Buffer

	flags, prefix := log.Flags(), log.Prefix()
	log.SetOutput(&out)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(flags)
		log.SetPrefix(prefix)
	})

	// The package lives under examples/ rather than in this package's testdata:
	// it is meant to generate, so the golden harness renders it and pins what
	// survives the skipping -- Widget and /widgets are absent from examples/
	// skipped/skipped.yaml precisely because @schemaa and @endpoin were passed
	// over. What a golden cannot see is the log, which is the whole point of the
	// rule, so that half is asserted here.
	if _, err := Parse("../../examples/skipped"); err != nil {
		t.Fatalf("Parse() error = %v; the package is meant to generate", err)
	}

	logged := out.String()

	// Order is not meaningful: these are written as each declaration is passed
	// over, across two stages of the parser.
	want := []string{
		"type Widget: @schemaa is not an annotation; it was skipped",
		"func ListWidgets: @endpoin is not an annotation; it was skipped",
		"field Widget.ID: @fild is not an annotation; it was skipped",
		"declaration ListGadgets.filters: @quer is not an annotation; it was skipped",
	}
	for _, line := range want {
		if !strings.Contains(logged, line) {
			t.Errorf("missing from the log:\n  %s\ngot:\n%s", line, logged)
		}
	}

	// Everything the package documents in prose must stay quiet: Gadget names
	// @schema only after discussing it, Helper carries no @ at all, and Escaped
	// opens with \@ — the escape the rest of the syntax uses for a literal @.
	for _, quiet := range []string{"type Gadget", "func Helper", "func Escaped", "@author"} {
		if strings.Contains(logged, quiet) {
			t.Errorf("%q should not be reported; got:\n%s", quiet, logged)
		}
	}
}
