package parser

import (
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/annotation"
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
			content, err := ParseBracedBlock(tt.lines, nil)
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
			content, err := ParseBracedBlock(tt.lines, nil)
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

// TestParseBracedBlock_RawValueBraces pins the rule that a raw value's braces
// are text. @pattern holds a regex, where {2} is a quantifier and } may appear
// unpaired; counting those as structure closed the block early and dropped
// everything after it, silently.
//
// The two forms are checked against each other on purpose. They used to be
// parsed by different code and gave different answers to the same input.
func TestParseBracedBlock_RawValueBraces(t *testing.T) {
	field := annotation.Schema.GetChild("@field")

	tests := []struct {
		name    string
		lines   []string
		want    []string
		wantErr bool
	}{
		{
			name:  "unpaired brace in regex, one line",
			lines: []string{"@field { @pattern ^a}b$ }"},
			want:  []string{"@pattern ^a}b$"},
		},
		{
			name:  "unpaired brace in regex, block",
			lines: []string{"@field {", "@pattern ^a}b$", "}"},
			want:  []string{"@pattern ^a}b$"},
		},
		{
			name:  "quantifier, one line",
			lines: []string{"@field { @description Country code @pattern ^[A-Z]{2}$ }"},
			want:  []string{"@description Country code @pattern ^[A-Z]{2}$"},
		},
		{
			name:  "quantifier, block",
			lines: []string{"@field {", "@description Locale", "@pattern ^[a-z]{2}(-[A-Z]{2})?$", "}"},
			want:  []string{"@description Locale", "@pattern ^[a-z]{2}(-[A-Z]{2})?$"},
		},
		{
			name:    "a block that really is unbalanced still reports",
			lines:   []string{"@field {", "@description Test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := ParseBracedBlock(tt.lines, field)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseBracedBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(content) != len(tt.want) {
				t.Fatalf("ParseBracedBlock() = %q, want %q", content, tt.want)
			}
			for i := range tt.want {
				if content[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, content[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseAnnotationBlock_Marker(t *testing.T) {
	// Use @path which is still a MarkerAnnotation
	node := annotation.Schema.GetChild("@path")
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
	apiNode := annotation.Schema.GetChild("@api")
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
	fieldNode := annotation.Schema.GetChild("@field")
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
	apiNode := annotation.Schema.GetChild("@api")
	lines := []string{
		"@api {",
		"  @title Test API",
		"  @version 1.0.0",
		"  @contact {",
		"    @name API Team",
		"    @email api\\@example.com",
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
	apiNode := annotation.Schema.GetChild("@api")
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
	fieldNode := annotation.Schema.GetChild("@field")
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
	endpointNode := annotation.Schema.GetChild("@endpoint")
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
			got := ExtractAnnotationName(tt.line)
			if got != tt.expected {
				t.Errorf("ExtractAnnotationName(%q) = %q, want %q", tt.line, got, tt.expected)
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
	endpointNode := annotation.Schema.GetChild("@endpoint")
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
	endpointNode := annotation.Schema.GetChild("@endpoint")

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

// TestParseAnnotationBlock_MultilineTermination covers what ends a multiline
// value. The rule is positional, not name-based: a line starting with an
// unescaped @ ends the value whatever the name is, and \@ is how a description
// writes a literal @ at the start of a line.
//
// The unknown-name case used to be read as prose, so a misspelled annotation was
// reported as a bare @ in the description. Escaping it, as that message advised,
// put the typo in the generated spec.
func TestParseAnnotationBlock_MultilineTermination(t *testing.T) {
	schemaNode := annotation.Schema.GetChild("@schema")

	t.Run("known annotation ends the value", func(t *testing.T) {
		lines := []string{
			"@schema {",
			"  @description A widget",
			"  @deprecated",
			"}",
		}

		parsed, err := ParseAnnotationBlock(lines, "@schema", schemaNode)
		if err != nil {
			t.Fatalf("ParseAnnotationBlock() error = %v", err)
		}

		if got := parsed.GetChildValue("@description"); got != "A widget" {
			t.Errorf("@description = %q, want %q", got, "A widget")
		}
		if !parsed.HasChild("@deprecated") {
			t.Error("@deprecated was swallowed into the description")
		}
	})

	t.Run("unknown annotation ends the value and is reported", func(t *testing.T) {
		lines := []string{
			"@schema {",
			"  @description A widget",
			"  @bogusStructRule yes",
			"}",
		}

		_, err := ParseAnnotationBlock(lines, "@schema", schemaNode)
		if err == nil {
			t.Fatal("expected an error for the unknown annotation")
		}
		if !strings.Contains(err.Error(), "unknown annotation @bogusStructRule") {
			t.Errorf("error = %q, want it to name the unknown annotation", err)
		}
	})

	t.Run("escaped @ continues the value", func(t *testing.T) {
		lines := []string{
			"@schema {",
			"  @description Contact us at",
			"  \\@support for help",
			"  @deprecated",
			"}",
		}

		parsed, err := ParseAnnotationBlock(lines, "@schema", schemaNode)
		if err != nil {
			t.Fatalf("ParseAnnotationBlock() error = %v", err)
		}

		want := "Contact us at\n@support for help"
		if got := parsed.GetChildValue("@description"); got != want {
			t.Errorf("@description = %q, want %q", got, want)
		}
		if !parsed.HasChild("@deprecated") {
			t.Error("@deprecated after an escaped line was swallowed")
		}
	})
}

// TestParseAnnotationBlock_StrayLine covers text inside a block that is not an
// annotation. It used to be skipped, so prose written inside the braces — or a
// single-line value wrapped onto a second line — disappeared without a word.
func TestParseAnnotationBlock_StrayLine(t *testing.T) {
	schemaNode := annotation.Schema.GetChild("@schema")
	fieldNode := annotation.Schema.GetChild("@field")

	tests := []struct {
		name  string
		node  *annotation.Def
		lines []string
		want  string
	}{
		{
			name: "prose inside a block",
			node: schemaNode,
			lines: []string{
				"@schema {",
				"  @deprecated",
				"  this line is prose",
				"}",
			},
			want: `"this line is prose" is not an annotation`,
		},
		{
			name: "single-line value wrapped onto the next line",
			node: fieldNode,
			lines: []string{
				"@field {",
				"  @format A Very Long",
				"    Wrapped Value",
				"}",
			},
			want: `"Wrapped Value" is not an annotation`,
		},
		{
			name: "blank lines are still skipped",
			node: schemaNode,
			lines: []string{
				"@schema {",
				"",
				"  @description A widget",
				"",
				"}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAnnotationBlock(tt.lines, tt.node.Name, tt.node)

			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error for the stray line")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

// The tests below cover a block written on one line. They used to call a second
// entry point of their own, which extracted the block by a different rule than
// the multi-line path and disagreed with it. Both forms now go through
// ParseAnnotationBlock.

// parseOneLine parses a block written on a single line, failing the test if it
// does not parse.
func parseOneLine(t *testing.T, line, name string, node *annotation.Def) *ParsedAnnotation {
	t.Helper()
	parsed, err := ParseAnnotationBlock([]string{line}, name, node)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock(%q) error = %v", line, err)
	}
	if parsed == nil {
		t.Fatalf("ParseAnnotationBlock(%q) returned nil without an error", line)
	}
	return parsed
}

func TestParseAnnotationBlock_OneLine(t *testing.T) {
	fieldNode := annotation.Schema.GetChild("@field")

	tests := []struct {
		name    string
		line    string
		wantErr bool
	}{
		{name: "simple", line: "@field { @description Test }"},
		{name: "several children", line: "@field { @description User email @format email @example test }"},
		{name: "empty block", line: "@field { }"},
		{name: "child that does not belong here", line: "@field { @with oauth { @scope read } }", wantErr: true},
		// A bare @field is not an error. The block is optional, and a field
		// that opens none simply carries no annotation.
		{name: "no block at all", line: "@field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseAnnotationBlock([]string{tt.line}, "@field", fieldNode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseAnnotationBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && parsed == nil {
				t.Error("ParseAnnotationBlock() returned nil without an error")
			}
		})
	}
}

func TestParseAnnotationBlock_OneLineValues(t *testing.T) {
	fieldNode := annotation.Schema.GetChild("@field")
	parsed := parseOneLine(t, "@field { @description User email @format email @example test }", "@field", fieldNode)

	for _, tt := range []struct{ child, want string }{
		{"@description", "User email"},
		{"@format", "email"},
		{"@example", "test"},
	} {
		if got := parsed.GetChildValue(tt.child); got != tt.want {
			t.Errorf("GetChildValue(%s) = %q, want %q", tt.child, got, tt.want)
		}
	}
}

func TestParseAnnotationBlock_OneLineFlag(t *testing.T) {
	fieldNode := annotation.Schema.GetChild("@field")
	parsed := parseOneLine(t, "@field { @description Test @deprecated }", "@field", fieldNode)

	deprecated := parsed.Children["@deprecated"]
	if deprecated == nil {
		t.Fatal("@deprecated child not found")
	}
	if !deprecated.IsFlag {
		t.Error("@deprecated should be a flag annotation")
	}
}

func TestParseAnnotationBlock_OneLineNestedBlockRejected(t *testing.T) {
	// A child that opens its own block cannot be written on one line: its
	// braces would be indistinguishable from the parent's.
	endpointNode := annotation.Schema.GetChild("@endpoint")
	line := "@endpoint GET /users { @response 200 { @body User } }"

	if _, err := ParseAnnotationBlock([]string{line}, "@endpoint", endpointNode); err == nil {
		t.Error("a nested block written on one line should be rejected")
	}
}

func TestParseAnnotationBlock_OneLineBlockChildren(t *testing.T) {
	apiNode := annotation.Schema.GetChild("@api")
	parsed := parseOneLine(t, "@api { @title Test API @version 1.0.0 }", "@api", apiNode)

	if got := parsed.GetChildValue("@title"); got != "Test API" {
		t.Errorf("@title = %q, want %q", got, "Test API")
	}
	if got := parsed.GetChildValue("@version"); got != "1.0.0" {
		t.Errorf("@version = %q, want %q", got, "1.0.0")
	}

	contactNode := apiNode.GetChild("@contact")
	contact := parseOneLine(t, "@contact { @name API Team @url https://example.com }", "@contact", contactNode)
	if !contact.HasChild("@name") || !contact.HasChild("@url") {
		t.Error("@contact should carry both @name and @url")
	}
}

func TestParseAnnotationBlock_OneLineRepeatable(t *testing.T) {
	securityNode := annotation.Schema.GetChild("@api").GetChild("@security")
	parsed := parseOneLine(t, "@security { @with apiKey }", "@security", securityNode)

	withChildren := parsed.RepeatedChildren["@with"]
	if len(withChildren) != 1 {
		t.Fatalf("expected 1 @with child, got %d", len(withChildren))
	}
	if withChildren[0].Value != "apiKey" {
		t.Errorf("@with value = %q, want %q", withChildren[0].Value, "apiKey")
	}
}

func TestParseAnnotationBlock_OneLineEscapes(t *testing.T) {
	fieldNode := annotation.Schema.GetChild("@field")

	tests := []struct {
		name  string
		line  string
		child string
		want  string
	}{
		{
			name:  "raw braces in pattern pass through",
			line:  `@field { @pattern ^[A-Z]{2}$ }`,
			child: "@pattern",
			want:  "^[A-Z]{2}$",
		},
		{
			name:  "quantifier range",
			line:  `@field { @pattern ^[a-z]{3,5}$ }`,
			child: "@pattern",
			want:  "^[a-z]{3,5}$",
		},
		{
			name:  "escaped @ in description",
			line:  `@field { @description Email uses \@ symbol }`,
			child: "@description",
			want:  "Email uses @ symbol",
		},
		{
			name:  "escaped backslash",
			line:  `@field { @example path\\to\\file }`,
			child: "@example",
			want:  `path\to\file`,
		},
		{
			name:  "JSON example with escaped braces",
			line:  `@field { @example \{"key": "value"\} }`,
			child: "@example",
			want:  `{"key": "value"}`,
		},
		{
			name:  "email in example",
			line:  `@field { @example admin\@example.com }`,
			child: "@example",
			want:  "admin@example.com",
		},
		{
			name:  "mixed escapes",
			line:  `@field { @description Use \@ and \{ \} for escaping }`,
			child: "@description",
			want:  "Use @ and { } for escaping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := parseOneLine(t, tt.line, "@field", fieldNode)
			if got := parsed.GetChildValue(tt.child); got != tt.want {
				t.Errorf("GetChildValue(%s) = %q, want %q", tt.child, got, tt.want)
			}
		})
	}
}

func TestParseAnnotationBlock_OneLineRawPatternWithSibling(t *testing.T) {
	// A raw value's braces must not end the block early, leaving the sibling
	// after it unparsed.
	fieldNode := annotation.Schema.GetChild("@field")
	parsed := parseOneLine(t, `@field { @pattern ^[a-fA-F0-9]{64}$ @description SHA-256 digest }`, "@field", fieldNode)

	if got := parsed.GetChildValue("@pattern"); got != "^[a-fA-F0-9]{64}$" {
		t.Errorf("@pattern = %q, want %q", got, "^[a-fA-F0-9]{64}$")
	}
	if got := parsed.GetChildValue("@description"); got != "SHA-256 digest" {
		t.Errorf("@description = %q, want %q", got, "SHA-256 digest")
	}
}

func TestParseAnnotation_RejectsUnescapedSpecials(t *testing.T) {
	fieldNode := annotation.Schema.GetChild("@field")
	apiNode := annotation.Schema.GetChild("@api")

	tests := []struct {
		name  string
		lines []string
		node  *annotation.Def
		root  string
		want  string // substring expected in the error
	}{
		{
			name: "unescaped @ in block @email value",
			lines: []string{
				"@api {",
				"  @title T",
				"  @version 1",
				"  @contact {",
				"    @email user@example.com",
				"  }",
				"}",
			},
			node: apiNode,
			root: "@api",
			want: "unescaped '@' in @email",
		},
		{
			name:  "unescaped { in one-line @description",
			lines: []string{`@field { @description Use {placeholder} here }`},
			node:  fieldNode,
			root:  "@field",
			want:  "unescaped '{' in @description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAnnotationBlock(tt.lines, tt.root, tt.node)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}
