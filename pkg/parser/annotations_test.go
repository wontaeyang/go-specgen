package parser

import (
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/schema"
)

func TestParseBracedBlock(t *testing.T) {
	tests := []struct {
		name          string
		lines         []string
		expectedLines int
		wantErr       bool
	}{
		{
			name: "simple block",
			lines: []string{
				"@field {",
				"  @description Test",
				"}",
			},
			expectedLines: 1,
			wantErr:       false,
		},
		{
			name: "nested blocks",
			lines: []string{
				"@api {",
				"  @contact {",
				"    @name Test",
				"  }",
				"}",
			},
			expectedLines: 3,
			wantErr:       false,
		},
		{
			name: "no braces (marker)",
			lines: []string{
				"@schema",
			},
			expectedLines: 0,
			wantErr:       false,
		},
		{
			name: "unbalanced braces",
			lines: []string{
				"@field {",
				"  @description Test",
			},
			expectedLines: 0,
			wantErr:       true,
		},
		{
			name:          "empty lines",
			lines:         []string{},
			expectedLines: 0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := ParseBracedBlock(tt.lines)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBracedBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(content) != tt.expectedLines {
				t.Errorf("ParseBracedBlock() returned %d lines, want %d", len(content), tt.expectedLines)
			}
		})
	}
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		annotationName string
		expected       string
	}{
		{
			name:           "endpoint with method and path",
			line:           "@endpoint GET /users/{id} {",
			annotationName: "@endpoint",
			expected:       "GET /users/{id}",
		},
		{
			name:           "server with URL",
			line:           "@server https://api.example.com {",
			annotationName: "@server",
			expected:       "https://api.example.com",
		},
		{
			name:           "response with status code",
			line:           "@response 200 {",
			annotationName: "@response",
			expected:       "200",
		},
		{
			name:           "no metadata",
			line:           "@field {",
			annotationName: "@field",
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractMetadata(tt.line, tt.annotationName)
			if got != tt.expected {
				t.Errorf("ExtractMetadata() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFindBlockOpener(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected int
	}{
		{
			name:     "simple block opener",
			line:     "@field {",
			expected: 7,
		},
		{
			name:     "path param with block",
			line:     "@endpoint GET /users/{id} {",
			expected: 26,
		},
		{
			name:     "multiple path params with block",
			line:     "@endpoint GET /orgs/{orgId}/projects/{projectId} {",
			expected: 49,
		},
		{
			name:     "no block - just path param",
			line:     "@endpoint GET /users/{id}",
			expected: -1,
		},
		{
			name:     "empty block",
			line:     "@endpoint GET /users/{id} {}",
			expected: 26,
		},
		{
			name:     "inline block with content",
			line:     "@field { @description test }",
			expected: 7,
		},
		{
			name:     "no block at all",
			line:     "@schema",
			expected: -1,
		},
		{
			name:     "tab before brace",
			line:     "@field\t{",
			expected: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findBlockOpener(tt.line)
			if got != tt.expected {
				t.Errorf("findBlockOpener(%q) = %d, want %d", tt.line, got, tt.expected)
			}
		})
	}
}

func TestExtractMetadata_PathParams(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		annotationName string
		expected       string
	}{
		{
			name:           "single path param",
			line:           "@endpoint GET /users/{id} {",
			annotationName: "@endpoint",
			expected:       "GET /users/{id}",
		},
		{
			name:           "multiple path params",
			line:           "@endpoint GET /orgs/{orgId}/projects/{projectId} {",
			annotationName: "@endpoint",
			expected:       "GET /orgs/{orgId}/projects/{projectId}",
		},
		{
			name:           "path param with empty block",
			line:           "@endpoint GET /users/{id} {}",
			annotationName: "@endpoint",
			expected:       "GET /users/{id}",
		},
		{
			name:           "path param inline block",
			line:           "@endpoint GET /users/{id} { @operationID getUser }",
			annotationName: "@endpoint",
			expected:       "GET /users/{id}",
		},
		{
			name:           "no block - path param only",
			line:           "@endpoint GET /users/{id}",
			annotationName: "@endpoint",
			expected:       "GET /users/{id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractMetadata(tt.line, tt.annotationName)
			if got != tt.expected {
				t.Errorf("ExtractMetadata() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseBracedBlock_PathParams(t *testing.T) {
	tests := []struct {
		name          string
		lines         []string
		expectedLines int
		wantErr       bool
	}{
		{
			name: "path param with multiline block",
			lines: []string{
				"@endpoint GET /users/{id} {",
				"  @operationID getUser",
				"  @response 200 { @body User }",
				"}",
			},
			expectedLines: 2,
			wantErr:       false,
		},
		{
			name: "multiple path params with block",
			lines: []string{
				"@endpoint GET /orgs/{orgId}/projects/{projectId} {",
				"  @operationID getProject",
				"}",
			},
			expectedLines: 1,
			wantErr:       false,
		},
		{
			name: "path param with empty block",
			lines: []string{
				"@endpoint GET /users/{id} {}",
			},
			expectedLines: 0,
			wantErr:       false,
		},
		{
			name: "path param with inline block content",
			lines: []string{
				"@endpoint GET /users/{id} { @operationID getUser }",
			},
			expectedLines: 1,
			wantErr:       false,
		},
		{
			name: "path param no block",
			lines: []string{
				"@endpoint GET /users/{id}",
			},
			expectedLines: 0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := ParseBracedBlock(tt.lines)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBracedBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(content) != tt.expectedLines {
				t.Errorf("ParseBracedBlock() returned %d lines, want %d. Content: %v", len(content), tt.expectedLines, content)
			}
		})
	}
}

func TestParseAnnotationBlock_Marker(t *testing.T) {
	// Use @path which is still a MarkerAnnotation
	node := schema.AnnotationSchema.GetChild("@path")
	lines := []string{"@path"}

	parsed, err := ParseAnnotationBlock(lines, "@path", node)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}

	if !parsed.IsFlag {
		t.Error("Marker annotation should have IsFlag = true")
	}

	if parsed.Name != "@path" {
		t.Errorf("Name = %s, want @path", parsed.Name)
	}
}

func TestParseAnnotationBlock_Value(t *testing.T) {
	apiNode := schema.AnnotationSchema.GetChild("@api")
	titleNode := apiNode.GetChild("@title")

	lines := []string{"@title My API Title"}

	parsed, err := ParseAnnotationBlock(lines, "@title", titleNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}

	if parsed.Value != "My API Title" {
		t.Errorf("Value = %q, want %q", parsed.Value, "My API Title")
	}
}

func TestParseAnnotationBlock_SimpleBlock(t *testing.T) {
	fieldNode := schema.AnnotationSchema.GetChild("@field")
	lines := []string{
		"@field {",
		"  @description User email address",
		"  @format email",
		"}",
	}

	parsed, err := ParseAnnotationBlock(lines, "@field", fieldNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}

	if parsed.Name != "@field" {
		t.Errorf("Name = %s, want @field", parsed.Name)
	}

	// Check children
	if !parsed.HasChild("@description") {
		t.Error("Should have @description child")
	}

	if !parsed.HasChild("@format") {
		t.Error("Should have @format child")
	}

	desc := parsed.GetChildValue("@description")
	if desc != "User email address" {
		t.Errorf("@description value = %q, want %q", desc, "User email address")
	}

	format := parsed.GetChildValue("@format")
	if format != "email" {
		t.Errorf("@format value = %q, want %q", format, "email")
	}
}

func TestParseAnnotationBlock_NestedBlocks(t *testing.T) {
	apiNode := schema.AnnotationSchema.GetChild("@api")
	lines := []string{
		"@api {",
		"  @title Test API",
		"  @version 1.0.0",
		"  @contact {",
		"    @name API Team",
		"    @email api@example.com",
		"  }",
		"}",
	}

	parsed, err := ParseAnnotationBlock(lines, "@api", apiNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}

	// Check top-level children
	if parsed.GetChildValue("@title") != "Test API" {
		t.Error("@title not parsed correctly")
	}

	if parsed.GetChildValue("@version") != "1.0.0" {
		t.Error("@version not parsed correctly")
	}

	// Check nested @contact
	contact := parsed.Children["@contact"]
	if contact == nil {
		t.Fatal("@contact child not found")
	}

	if contact.GetChildValue("@name") != "API Team" {
		t.Error("@contact.@name not parsed correctly")
	}

	if contact.GetChildValue("@email") != "api@example.com" {
		t.Error("@contact.@email not parsed correctly")
	}
}

func TestParseAnnotationBlock_Repeatable(t *testing.T) {
	apiNode := schema.AnnotationSchema.GetChild("@api")
	lines := []string{
		"@api {",
		"  @server https://api.example.com {",
		"    @description Production",
		"  }",
		"  @server https://staging.example.com {",
		"    @description Staging",
		"  }",
		"}",
	}

	parsed, err := ParseAnnotationBlock(lines, "@api", apiNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}

	servers := parsed.GetRepeatedChildren("@server")
	if len(servers) != 2 {
		t.Fatalf("Expected 2 servers, got %d", len(servers))
	}

	// Check first server
	if servers[0].Metadata != "https://api.example.com" {
		t.Errorf("First server metadata = %q, want %q", servers[0].Metadata, "https://api.example.com")
	}

	if servers[0].GetChildValue("@description") != "Production" {
		t.Error("First server description not parsed correctly")
	}

	// Check second server
	if servers[1].Metadata != "https://staging.example.com" {
		t.Errorf("Second server metadata = %q, want %q", servers[1].Metadata, "https://staging.example.com")
	}
}

func TestParseAnnotationBlock_EmptyBlock(t *testing.T) {
	// @field allows empty because it's marker pattern
	fieldNode := schema.AnnotationSchema.GetChild("@field")
	lines := []string{"@field { }"}

	parsed, err := ParseAnnotationBlock(lines, "@field", fieldNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v, want nil (AllowEmpty=true)", err)
	}

	if len(parsed.Children) != 0 {
		t.Error("Empty block should have no children")
	}
}

func TestParseAnnotationBlock_WithMetadata(t *testing.T) {
	endpointNode := schema.AnnotationSchema.GetChild("@endpoint")
	lines := []string{
		"@endpoint GET /users/{id} {",
		"@summary Get user by ID",
		"}",
	}

	parsed, err := ParseAnnotationBlock(lines, "@endpoint", endpointNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}

	if parsed.Metadata != "GET /users/{id}" {
		t.Errorf("Metadata = %q, want %q", parsed.Metadata, "GET /users/{id}")
	}

	if parsed.GetChildValue("@summary") != "Get user by ID" {
		t.Error("@summary not parsed correctly")
	}
}

func TestExtractAnnotationName(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"simple", "@field", "@field"},
		{"with value", "@title My API", "@title"},
		{"with brace", "@field {", "@field"},
		{"with spaces", "  @description  ", "@description"},
		{"not annotation", "regular text", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAnnotationName(tt.line)
			if got != tt.expected {
				t.Errorf("extractAnnotationName(%q) = %q, want %q", tt.line, got, tt.expected)
			}
		})
	}
}

func TestParsedAnnotation_Helpers(t *testing.T) {
	parsed := &ParsedAnnotation{
		Name: "@field",
		Children: map[string]*ParsedAnnotation{
			"@description": {Name: "@description", Value: "test"},
			"@format":      {Name: "@format", Value: "email"},
		},
		RepeatedChildren: map[string][]*ParsedAnnotation{
			"@enum": {
				{Name: "@enum", Value: "value1"},
				{Name: "@enum", Value: "value2"},
			},
		},
	}

	// Test GetChildValue
	if got := parsed.GetChildValue("@description"); got != "test" {
		t.Errorf("GetChildValue(@description) = %q, want %q", got, "test")
	}

	if got := parsed.GetChildValue("@nonexistent"); got != "" {
		t.Errorf("GetChildValue(@nonexistent) = %q, want empty", got)
	}

	// Test HasChild
	if !parsed.HasChild("@description") {
		t.Error("HasChild(@description) = false, want true")
	}

	if parsed.HasChild("@nonexistent") {
		t.Error("HasChild(@nonexistent) = true, want false")
	}

	// Test GetRepeatedChildren
	enums := parsed.GetRepeatedChildren("@enum")
	if len(enums) != 2 {
		t.Errorf("GetRepeatedChildren(@enum) returned %d items, want 2", len(enums))
	}
}

func TestParseAnnotationBlock_InlineResponse(t *testing.T) {
	endpointNode := schema.AnnotationSchema.GetChild("@endpoint")
	responseNode := endpointNode.GetChild("@response")

	lines := []string{"@response 200 { @body User @description User found }"}

	parsed, err := ParseAnnotationBlock(lines, "@response", responseNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}

	// Check metadata
	if parsed.Metadata != "200" {
		t.Errorf("Metadata = %q, want %q", parsed.Metadata, "200")
	}

	// Check @body child
	body, ok := parsed.Children["@body"]
	if !ok {
		t.Fatal("@body child not found")
	}
	if body.Value != "User" {
		t.Errorf("@body value = %q, want %q", body.Value, "User")
	}

	// Check @description child
	desc, ok := parsed.Children["@description"]
	if !ok {
		t.Fatal("@description child not found")
	}
	if desc.Value != "User found" {
		t.Errorf("@description value = %q, want %q", desc.Value, "User found")
	}
}

func TestParseAnnotationBlock_NestedInlineResponse(t *testing.T) {
	endpointNode := schema.AnnotationSchema.GetChild("@endpoint")

	// Simulating the actual lines from a comment block
	lines := []string{
		"@endpoint GET /users/{id} {",
		"@summary Get user by ID",
		"@description Retrieves a single user by their unique identifier.",
		"@path UserPath",
		"@response 200 { @body User @description User found successfully }",
		"@response 404 { @body Error @description User not found }",
		"}",
	}

	parsed, err := ParseAnnotationBlock(lines, "@endpoint", endpointNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}

	// Check metadata
	if parsed.Metadata != "GET /users/{id}" {
		t.Errorf("Metadata = %q, want %q", parsed.Metadata, "GET /users/{id}")
	}

	// Check @response children (repeatable)
	responses := parsed.GetRepeatedChildren("@response")
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	// Check first response
	resp200 := responses[0]
	if resp200.Metadata != "200" {
		t.Errorf("resp200.Metadata = %q, want %q", resp200.Metadata, "200")
	}

	body200, ok := resp200.Children["@body"]
	if !ok {
		t.Fatal("resp200 @body not found")
	}
	if body200.Value != "User" {
		t.Errorf("resp200 @body.Value = %q, want %q", body200.Value, "User")
	}
	if body200.Metadata != "User" {
		t.Errorf("resp200 @body.Metadata = %q, want %q", body200.Metadata, "User")
	}

	desc200, ok := resp200.Children["@description"]
	if !ok {
		t.Fatal("resp200 @description not found")
	}
	if desc200.Value != "User found successfully" {
		t.Errorf("resp200 @description.Value = %q, want %q", desc200.Value, "User found successfully")
	}
}
