package parser

import (
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/schema"
)

func TestIsInlineFormat(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected bool
	}{
		{
			name:     "inline format",
			lines:    []string{"@field { @description Test @format email }"},
			expected: true,
		},
		{
			name: "multi-line format",
			lines: []string{
				"@field {",
				"  @description Test",
				"}",
			},
			expected: false,
		},
		{
			name:     "no braces",
			lines:    []string{"@schema"},
			expected: false,
		},
		{
			name:     "only opening brace",
			lines:    []string{"@field {"},
			expected: false,
		},
		{
			name:     "empty",
			lines:    []string{},
			expected: false,
		},
		{
			name:     "braces in wrong order",
			lines:    []string{"@field } {"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInlineFormat(tt.lines)
			if got != tt.expected {
				t.Errorf("IsInlineFormat() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseInlineAnnotation(t *testing.T) {
	fieldNode := schema.AnnotationSchema.GetChild("@field")

	tests := []struct {
		name    string
		line    string
		wantErr bool
	}{
		{
			name:    "simple inline",
			line:    "@field { @description Test }",
			wantErr: false,
		},
		{
			name:    "multiple children (no @ in values)",
			line:    "@field { @description User email @format email @example test }",
			wantErr: false,
		},
		{
			name:    "empty inline",
			line:    "@field { }",
			wantErr: false,
		},
		{
			name:    "nested braces (should error)",
			line:    "@field { @with oauth { @scope read } }",
			wantErr: true,
		},
		{
			name:    "no braces",
			line:    "@field",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseInlineAnnotation(tt.line, "@field", fieldNode)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInlineAnnotation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && parsed == nil {
				t.Error("ParseInlineAnnotation() returned nil without error")
			}
		})
	}
}

func TestParseInlineAnnotation_Values(t *testing.T) {
	fieldNode := schema.AnnotationSchema.GetChild("@field")
	// NOTE: @ symbols in values not supported in current inline implementation
	line := "@field { @description User email @format email @example test }"

	parsed, err := ParseInlineAnnotation(line, "@field", fieldNode)
	if err != nil {
		t.Fatalf("ParseInlineAnnotation() error = %v", err)
	}

	tests := []struct {
		child    string
		expected string
	}{
		{"@description", "User email"},
		{"@format", "email"},
		{"@example", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.child, func(t *testing.T) {
			got := parsed.GetChildValue(tt.child)
			if got != tt.expected {
				t.Errorf("GetChildValue(%s) = %q, want %q", tt.child, got, tt.expected)
			}
		})
	}
}

func TestParseInlineAnnotation_EmailInValue(t *testing.T) {
	// NOTE: @ symbols in values are NOT supported in inline format
	// This is a known limitation - use multi-line format for values with @ symbols
	apiNode := schema.AnnotationSchema.GetChild("@api")
	contactNode := apiNode.GetChild("@contact")
	line := "@contact { @name API Team @url https://example.com }"

	parsed, err := ParseInlineAnnotation(line, "@contact", contactNode)
	if err != nil {
		t.Fatalf("ParseInlineAnnotation() error = %v", err)
	}

	if !parsed.HasChild("@name") {
		t.Error("Should have @name child")
	}
	if !parsed.HasChild("@url") {
		t.Error("Should have @url child")
	}

	// Limitation: @ in values not supported - use multi-line format instead
	t.Log("NOTE: Inline format doesn't support @ symbols in values. Use multi-line for email addresses.")
}

func TestParseInlineAnnotation_FlagAnnotation(t *testing.T) {
	fieldNode := schema.AnnotationSchema.GetChild("@field")
	line := "@field { @description Test @deprecated }"

	parsed, err := ParseInlineAnnotation(line, "@field", fieldNode)
	if err != nil {
		t.Fatalf("ParseInlineAnnotation() error = %v", err)
	}

	deprecated := parsed.Children["@deprecated"]
	if deprecated == nil {
		t.Fatal("@deprecated child not found")
	}

	if !deprecated.IsFlag {
		t.Error("@deprecated should be a flag annotation")
	}
}

func TestParseInlineAnnotation_NestedBlocksNotAllowed(t *testing.T) {
	// All block annotations support inline, but nested blocks within inline are not allowed
	endpointNode := schema.AnnotationSchema.GetChild("@endpoint")
	line := "@endpoint GET /users { @response 200 { @body User } }"

	_, err := ParseInlineAnnotation(line, "@endpoint", endpointNode)
	if err == nil {
		t.Error("ParseInlineAnnotation() should error for nested blocks in inline format")
	}
}

func TestParseInlineAnnotation_AllBlocksSupported(t *testing.T) {
	// All block annotations now support inline format
	apiNode := schema.AnnotationSchema.GetChild("@api")
	line := "@api { @title Test API @version 1.0.0 }"

	parsed, err := ParseInlineAnnotation(line, "@api", apiNode)
	if err != nil {
		t.Fatalf("ParseInlineAnnotation() error = %v, all blocks should support inline", err)
	}

	if parsed.GetChildValue("@title") != "Test API" {
		t.Errorf("@title = %q, want %q", parsed.GetChildValue("@title"), "Test API")
	}
	if parsed.GetChildValue("@version") != "1.0.0" {
		t.Errorf("@version = %q, want %q", parsed.GetChildValue("@version"), "1.0.0")
	}
}

func TestConvertToMultiLine(t *testing.T) {
	inlineLine := "@field { @description Test @format email }"
	annotationName := "@field"

	result := ConvertToMultiLine(inlineLine, annotationName)

	if len(result) < 3 {
		t.Errorf("ConvertToMultiLine() returned %d lines, expected at least 3", len(result))
	}

	// Should start with opening brace
	if result[0] != "@field {" {
		t.Errorf("First line = %q, want %q", result[0], "@field {")
	}

	// Should end with closing brace
	if result[len(result)-1] != "}" {
		t.Errorf("Last line = %q, want %q", result[len(result)-1], "}")
	}

	// Middle lines should be indented annotations
	for i := 1; i < len(result)-1; i++ {
		if !strings.HasPrefix(result[i], "  @") {
			t.Errorf("Line %d = %q, should start with '  @'", i, result[i])
		}
	}
}

func TestParseInlineChildren_Repeatable(t *testing.T) {
	// Test repeatable sub-commands in inline format
	// @with supports inline since it's a SubCommand
	apiNode := schema.AnnotationSchema.GetChild("@api")
	securityNode := apiNode.GetChild("@security")
	withNode := securityNode.GetChild("@with")

	// Verify @with is a SubCommand (all SubCommands support inline)
	if withNode.Type != schema.SubCommand {
		t.Fatalf("@with should be SubCommand, got %v", withNode.Type)
	}

	// @security is a BlockAnnotation and supports inline format
	// But we can't have nested blocks in inline, so @with with children would fail
	if securityNode.Type != schema.BlockAnnotation {
		t.Fatalf("@security should be BlockAnnotation, got %v", securityNode.Type)
	}

	// Test flat inline security (no nested blocks)
	line := "@security { @with apiKey }"
	parsed, err := ParseInlineAnnotation(line, "@security", securityNode)
	if err != nil {
		t.Fatalf("ParseInlineAnnotation() error = %v", err)
	}

	// @with is repeatable, so it's stored in RepeatedChildren
	withChildren := parsed.RepeatedChildren["@with"]
	if len(withChildren) != 1 {
		t.Errorf("expected 1 @with child, got %d", len(withChildren))
	}
	if withChildren[0].Value != "apiKey" {
		t.Errorf("@with value = %q, want %q", withChildren[0].Value, "apiKey")
	}
}

// Tests for inline struct extraction from function bodies

func TestDetectInlineAnnotation(t *testing.T) {
	tests := []struct {
		name           string
		lines          []string
		wantAnnotation string
		wantStatusCode string
	}{
		{
			name:           "query annotation",
			lines:          []string{"@query"},
			wantAnnotation: "query",
			wantStatusCode: "",
		},
		{
			name:           "path annotation",
			lines:          []string{"@path"},
			wantAnnotation: "path",
			wantStatusCode: "",
		},
		{
			name:           "header annotation",
			lines:          []string{"@header"},
			wantAnnotation: "header",
			wantStatusCode: "",
		},
		{
			name:           "cookie annotation",
			lines:          []string{"@cookie"},
			wantAnnotation: "cookie",
			wantStatusCode: "",
		},
		{
			name:           "request annotation",
			lines:          []string{"@request"},
			wantAnnotation: "request",
			wantStatusCode: "",
		},
		{
			name:           "response without status code",
			lines:          []string{"@response"},
			wantAnnotation: "response",
			wantStatusCode: "200",
		},
		{
			name:           "response with status code 200",
			lines:          []string{"@response 200"},
			wantAnnotation: "response",
			wantStatusCode: "200",
		},
		{
			name:           "response with status code 201",
			lines:          []string{"@response 201"},
			wantAnnotation: "response",
			wantStatusCode: "201",
		},
		{
			name:           "response with status code 404",
			lines:          []string{"@response 404"},
			wantAnnotation: "response",
			wantStatusCode: "404",
		},
		{
			name:           "response with status code and block",
			lines:          []string{"@response 201 {"},
			wantAnnotation: "response",
			wantStatusCode: "201",
		},
		{
			name:           "no annotation",
			lines:          []string{"some comment", "more text"},
			wantAnnotation: "",
			wantStatusCode: "",
		},
		{
			name:           "annotation with preceding text",
			lines:          []string{"some description", "@query"},
			wantAnnotation: "query",
			wantStatusCode: "",
		},
		{
			name:           "whitespace before annotation",
			lines:          []string{"  @path  "},
			wantAnnotation: "path",
			wantStatusCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAnnotation, gotStatusCode := detectInlineAnnotation(tt.lines)
			if gotAnnotation != tt.wantAnnotation {
				t.Errorf("detectInlineAnnotation() annotation = %q, want %q", gotAnnotation, tt.wantAnnotation)
			}
			if gotStatusCode != tt.wantStatusCode {
				t.Errorf("detectInlineAnnotation() statusCode = %q, want %q", gotStatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestParseInlineAnnotation_EscapedCharacters(t *testing.T) {
	fieldNode := schema.AnnotationSchema.GetChild("@field")

	tests := []struct {
		name     string
		line     string
		child    string
		expected string
	}{
		{
			name:     "escaped braces in pattern",
			line:     `@field { @pattern ^[A-Z]\{2\}$ }`,
			child:    "@pattern",
			expected: "^[A-Z]{2}$",
		},
		{
			name:     "escaped @ in description",
			line:     `@field { @description Email uses \@ symbol }`,
			child:    "@description",
			expected: "Email uses @ symbol",
		},
		{
			name:     "escaped backslash",
			line:     `@field { @example path\\to\\file }`,
			child:    "@example",
			expected: `path\to\file`,
		},
		{
			name:     "JSON example with escapes",
			line:     `@field { @example \{"key": "value"\} }`,
			child:    "@example",
			expected: `{"key": "value"}`,
		},
		{
			name:     "regex quantifier range",
			line:     `@field { @pattern ^[a-z]\{3,5\}$ }`,
			child:    "@pattern",
			expected: "^[a-z]{3,5}$",
		},
		{
			name:     "email in example",
			line:     `@field { @example admin\@example.com }`,
			child:    "@example",
			expected: "admin@example.com",
		},
		{
			name:     "mixed escapes",
			line:     `@field { @description Use \@ and \{ \} for escaping }`,
			child:    "@description",
			expected: "Use @ and { } for escaping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseInlineAnnotation(tt.line, "@field", fieldNode)
			if err != nil {
				t.Fatalf("ParseInlineAnnotation() error = %v", err)
			}

			got := parsed.GetChildValue(tt.child)
			if got != tt.expected {
				t.Errorf("GetChildValue(%s) = %q, want %q", tt.child, got, tt.expected)
			}
		})
	}
}

func TestParseInlineAnnotation_EscapedBracesAllowed(t *testing.T) {
	// Test that escaped braces don't trigger "nested blocks" error
	fieldNode := schema.AnnotationSchema.GetChild("@field")

	// This previously failed with "nested blocks cannot be inlined"
	line := `@field { @pattern ^[A-Z]\{2\}$ @description Country code }`

	parsed, err := ParseInlineAnnotation(line, "@field", fieldNode)
	if err != nil {
		t.Fatalf("ParseInlineAnnotation() should not error for escaped braces, got: %v", err)
	}

	pattern := parsed.GetChildValue("@pattern")
	if pattern != "^[A-Z]{2}$" {
		t.Errorf("pattern = %q, want %q", pattern, "^[A-Z]{2}$")
	}

	desc := parsed.GetChildValue("@description")
	if desc != "Country code" {
		t.Errorf("description = %q, want %q", desc, "Country code")
	}
}

func TestFuncInlineInfo_Fields(t *testing.T) {
	// Test that the FuncInlineInfo struct is correctly initialized
	info := &FuncInlineInfo{
		Query:     &InlineStructInfo{VarName: "query"},
		Path:      &InlineStructInfo{VarName: "path"},
		Header:    &InlineStructInfo{VarName: "header"},
		Cookie:    &InlineStructInfo{VarName: "cookie"},
		Request:   &InlineStructInfo{VarName: "request"},
		Responses: make(map[string]*InlineStructInfo),
	}
	info.Responses["200"] = &InlineStructInfo{VarName: "resp200", StatusCode: "200"}
	info.Responses["404"] = &InlineStructInfo{VarName: "resp404", StatusCode: "404"}

	if info.Query.VarName != "query" {
		t.Errorf("Query.VarName = %q, want %q", info.Query.VarName, "query")
	}
	if info.Path.VarName != "path" {
		t.Errorf("Path.VarName = %q, want %q", info.Path.VarName, "path")
	}
	if info.Header.VarName != "header" {
		t.Errorf("Header.VarName = %q, want %q", info.Header.VarName, "header")
	}
	if info.Cookie.VarName != "cookie" {
		t.Errorf("Cookie.VarName = %q, want %q", info.Cookie.VarName, "cookie")
	}
	if info.Request.VarName != "request" {
		t.Errorf("Request.VarName = %q, want %q", info.Request.VarName, "request")
	}
	if len(info.Responses) != 2 {
		t.Errorf("len(Responses) = %d, want %d", len(info.Responses), 2)
	}
	if info.Responses["200"].StatusCode != "200" {
		t.Errorf("Responses[200].StatusCode = %q, want %q", info.Responses["200"].StatusCode, "200")
	}
}

func TestInlineStructInfo_Fields(t *testing.T) {
	// Test that the InlineStructInfo struct is correctly initialized
	info := &InlineStructInfo{
		VarName:       "testVar",
		Annotation:    "query",
		StatusCode:    "",
		FieldComments: make(map[string]*CommentBlock),
	}
	info.FieldComments["ID"] = &CommentBlock{Lines: []string{"@field { @description User ID }"}}

	if info.VarName != "testVar" {
		t.Errorf("VarName = %q, want %q", info.VarName, "testVar")
	}
	if info.Annotation != "query" {
		t.Errorf("Annotation = %q, want %q", info.Annotation, "query")
	}
	if len(info.FieldComments) != 1 {
		t.Errorf("len(FieldComments) = %d, want %d", len(info.FieldComments), 1)
	}
}
