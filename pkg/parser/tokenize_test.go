package parser

import (
	"errors"
	"go/token"
	"strings"
	"testing"
)

// at builds comment lines with fake but valid positions, so tests can assert
// that the reported position points at the right line.
func at(texts ...string) []Line {
	lines := make([]Line, len(texts))
	for i, text := range texts {
		lines[i] = Line{
			Text: text,
			Pos:  token.Position{Filename: "fixture.go", Line: i + 1, Column: 1},
		}
	}
	return lines
}

func TestUnescapeValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no escapes", "hello world", "hello world"},
		{"escaped opening brace", `^[A-Z]\{2\}$`, "^[A-Z]{2}$"},
		{"escaped closing brace", `\}`, "}"},
		{"escaped at", `user\@example.com`, "user@example.com"},
		{"escaped backslash", `path\\to\\file`, `path\to\file`},
		{"mixed escapes", `\{a\@b\\c\}`, `{a@b\c}`},
		{"json example", `\{"key": "value"\}`, `{"key": "value"}`},
		{"regex quantifier", `^[a-z]\{3,5\}$`, "^[a-z]{3,5}$"},
		{"multiple emails", `admin\@example.com, support\@example.com`, "admin@example.com, support@example.com"},
		{"backslash before non-escape char", `\n\t`, `\n\t`},
		{"escaped backslash then escaped brace", `\\\{`, `\{`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnescapeValue(tt.input); got != tt.want {
				t.Errorf("UnescapeValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCountUnescapedBraces(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{"no braces", "hello world", 0},
		{"balanced", "{ hello }", 0},
		{"unbalanced open", "{ hello", 1},
		{"unbalanced close", "hello }", -1},
		{"nested", "{ { } }", 0},
		{"escaped braces ignored", `\{ hello \}`, 0},
		{"mixed real and escaped", `{ \{2\} }`, 0},
		{"regex pattern inline", `@field { @pattern ^[A-Z]\{2\}$ }`, 0},
		{"escaped backslash then real brace", `\\{`, 1},
		{"unescaped regex quantifier is balanced", `@field { @pattern ^\d{3}-\d{4}$ }`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CountUnescapedBraces(tt.content); got != tt.want {
				t.Errorf("CountUnescapedBraces(%q) = %d, want %d", tt.content, got, tt.want)
			}
		})
	}
}

func TestSplitOnUnescapedAt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"no at", "hello world", []string{"hello world"}},
		{"single at", "@foo bar", []string{"", "foo bar"}},
		{"two ats", "@foo bar @baz qux", []string{"", "foo bar ", "baz qux"}},
		{"escaped at does not split", `@example user\@example.com`, []string{"", `example user\@example.com`}},
		{"escaped at then real one", `@example user\@example.com @description x`, []string{"", `example user\@example.com `, "description x"}},
		{"escaped backslash then real at", `@x \\@y`, []string{"", `x \\`, "y"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitOnUnescapedAt(tt.content)
			if len(got) != len(tt.want) {
				t.Fatalf("SplitOnUnescapedAt(%q) = %q, want %q", tt.content, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("part %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFindUnescapedSpecial(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"plain value", false},
		{"has { brace", true},
		{"has } brace", true},
		{"has @ at", true},
		{`escaped \{ \} \@`, false},
		{`^[A-Z]\{2\}$`, false},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			if _, got := FindUnescapedSpecial(tt.content); got != tt.want {
				t.Errorf("FindUnescapedSpecial(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestContainsUnescapedBrace(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"hello world", false},
		{"hello { world", true},
		{"hello } world", true},
		{`\{2\}`, false},
		{`{ \{2\} }`, true},
		{`^[A-Z]\{2\}$`, false},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			if got := ContainsUnescapedBrace(tt.content); got != tt.want {
				t.Errorf("ContainsUnescapedBrace(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestStartsWithUnescapedAt(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"@description hello", true},
		{`\@example.com`, false},
		{"  @field", true},
		{`  \@admin`, false},
		{"hello world", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := StartsWithUnescapedAt(tt.line); got != tt.want {
				t.Errorf("StartsWithUnescapedAt(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestFindBlockOpener(t *testing.T) {
	tests := []struct {
		name string
		line string
		want int
	}{
		{"no brace", "@summary Get user", -1},
		{"space before brace", "@field {", 7},
		{"tab before brace", "@field\t{", 7},
		{"path param is not an opener", "@endpoint GET /users/{id}", -1},
		{"path param then opener", "@endpoint GET /users/{id} {", 26},
		{"escaped brace is not an opener", `@field \{`, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findBlockOpener(tt.line); got != tt.want {
				t.Errorf("findBlockOpener(%q) = %d, want %d", tt.line, got, tt.want)
			}
		})
	}
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		annotation string
		want       string
	}{
		{"method and path", "@endpoint GET /users {", "@endpoint", "GET /users"},
		{"single path param", "@endpoint GET /users/{id} {", "@endpoint", "GET /users/{id}"},
		{"two path params", "@endpoint GET /orgs/{orgId}/projects/{projectId} {", "@endpoint", "GET /orgs/{orgId}/projects/{projectId}"},
		{"empty block", "@endpoint GET /users/{id} {}", "@endpoint", "GET /users/{id}"},
		{"inline block", "@endpoint GET /users/{id} { @operationID getUser }", "@endpoint", "GET /users/{id}"},
		{"no block", "@endpoint GET /users/{id}", "@endpoint", "GET /users/{id}"},
		{"status code", "@response 404 { @body Error }", "@response", "404"},
		{"literal default status", "@response default {", "@response", "default"},
		{"no metadata", "@response {", "@response", ""},
		{"server url", "@server https://api.example.com {", "@server", "https://api.example.com"},
		{"escaped at", `@tag support\@example.com`, "@tag", "support@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractMetadata(tt.line, tt.annotation); got != tt.want {
				t.Errorf("ExtractMetadata(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestParseBracedBlock(t *testing.T) {
	tests := []struct {
		name    string
		lines   []Line
		want    []string
		wantErr string
	}{
		{
			name:  "multi line",
			lines: at("@field {", "@description Test", "@format email", "}"),
			want:  []string{"@description Test", "@format email"},
		},
		{
			name:  "single line",
			lines: at("@field { @description Test @format email }"),
			want:  []string{"@description Test @format email"},
		},
		{
			name:  "empty block",
			lines: at("@field {}"),
			want:  nil,
		},
		{
			name:  "no block at all",
			lines: at("@field"),
			want:  nil,
		},
		{
			name:  "path params do not open a block",
			lines: at("@endpoint GET /users/{id}"),
			want:  nil,
		},
		{
			name:  "nested block is kept whole",
			lines: at("@endpoint GET /users/{id} {", "@response 200 {", "@body User", "}", "}"),
			want:  []string{"@response 200 {", "@body User", "}"},
		},
		{
			name:    "unbalanced open",
			lines:   at("@field {", "@description Test"),
			wantErr: "unbalanced braces",
		},
		{
			name:    "closes more than it opens",
			lines:   at("@field {", "} }"),
			wantErr: "unbalanced braces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBracedBlock(tt.lines)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseBracedBlock() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseBracedBlock() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseBracedBlock() = %v, want %v", texts(got), tt.want)
			}
			for i := range got {
				if got[i].Text != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i].Text, tt.want[i])
				}
			}
		})
	}
}

func texts(lines []Line) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = line.Text
	}
	return out
}

func TestParseBracedBlock_KeepsLinePositions(t *testing.T) {
	content, err := ParseBracedBlock(at("@field {", "@description Test", "}"))
	if err != nil {
		t.Fatalf("ParseBracedBlock() error = %v", err)
	}
	if len(content) != 1 {
		t.Fatalf("got %d content lines, want 1", len(content))
	}
	if content[0].Pos.Line != 2 {
		t.Errorf("content line position = %d, want 2", content[0].Pos.Line)
	}
}

// Single-line and multi-line blocks are two spellings of the same thing, and
// must parse to the same tree.
func TestParseAnnotationBlock_SpellingsAgree(t *testing.T) {
	single, err := ParseAnnotationBlock(at("@field { @description User email @format email @readOnly }"), "@field", fieldNode)
	if err != nil {
		t.Fatalf("inline spelling: %v", err)
	}

	multi, err := ParseAnnotationBlock(at("@field {", "@description User email", "@format email", "@readOnly", "}"), "@field", fieldNode)
	if err != nil {
		t.Fatalf("multi-line spelling: %v", err)
	}

	for _, a := range []*Annotation{single, multi} {
		if got := a.ChildValue("@description"); got != "User email" {
			t.Errorf("@description = %q, want %q", got, "User email")
		}
		if got := a.ChildValue("@format"); got != "email" {
			t.Errorf("@format = %q, want %q", got, "email")
		}
		if !a.HasChild("@readOnly") {
			t.Error("@readOnly missing")
		}
	}
}

func TestParseAnnotationBlock_Flag(t *testing.T) {
	parsed, err := ParseAnnotationBlock(at("@path"), "@path", Grammar["@path"])
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}
	if len(parsed.Children) != 0 {
		t.Errorf("flag annotation has %d children, want 0", len(parsed.Children))
	}
}

func TestParseAnnotationBlock_NestedBlocks(t *testing.T) {
	lines := at(
		"@endpoint GET /users/{id} {",
		"@summary Get user by ID",
		"@path UserPath",
		"@response 200 { @body User @description User found }",
		"@response 404 { @body Error @description User missing }",
		"}",
	)

	parsed, err := ParseAnnotationBlock(lines, "@endpoint", endpointNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}

	if parsed.Metadata != "GET /users/{id}" {
		t.Errorf("Metadata = %q, want %q", parsed.Metadata, "GET /users/{id}")
	}
	if got := parsed.ChildValue("@summary"); got != "Get user by ID" {
		t.Errorf("@summary = %q", got)
	}

	responses := parsed.RepeatedChildren("@response")
	if len(responses) != 2 {
		t.Fatalf("got %d responses, want 2", len(responses))
	}
	if responses[0].Metadata != "200" || responses[1].Metadata != "404" {
		t.Errorf("statuses = %q, %q; want 200, 404", responses[0].Metadata, responses[1].Metadata)
	}

	// A body value lands in exactly one slot.
	body := responses[0].Child("@body")
	if body.Value != "User" {
		t.Errorf("@body value = %q, want %q", body.Value, "User")
	}
	if body.Metadata != "" {
		t.Errorf("@body metadata = %q, want it empty (the value belongs in Value)", body.Metadata)
	}
}

func TestParseAnnotationBlock_MultilineDescription(t *testing.T) {
	lines := at(
		"@endpoint GET /users {",
		"@description First line.",
		"Second line.",
		"@summary List users",
		"}",
	)

	parsed, err := ParseAnnotationBlock(lines, "@endpoint", endpointNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}

	want := "First line.\nSecond line."
	if got := parsed.ChildValue("@description"); got != want {
		t.Errorf("@description = %q, want %q", got, want)
	}
	if got := parsed.ChildValue("@summary"); got != "List users" {
		t.Errorf("@summary = %q, want the sibling to end the description", got)
	}
}

func TestParseAnnotationBlock_SingleLineValueDoesNotAbsorbNextLine(t *testing.T) {
	lines := at("@securityScheme bearerAuth {", "@type http", "@scheme bearer", "}")

	parsed, err := ParseAnnotationBlock(lines, "@securityScheme", apiNode.GetChild("@securityScheme"))
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}
	if got := parsed.ChildValue("@type"); got != "http" {
		t.Errorf("@type = %q, want %q", got, "http")
	}
	if got := parsed.ChildValue("@scheme"); got != "bearer" {
		t.Errorf("@scheme = %q, want %q", got, "bearer")
	}
}

func TestParseAnnotationBlock_Errors(t *testing.T) {
	tests := []struct {
		name     string
		lines    []Line
		node     *GrammarNode
		annotate string
		wantErr  string
		wantLine int
	}{
		{
			name:     "unknown child",
			lines:    at("@endpoint GET /users {", "@summry Typo", "}"),
			node:     endpointNode,
			annotate: "@endpoint",
			wantErr:  "unknown annotation @summry in @endpoint",
			wantLine: 2,
		},
		{
			name:     "unknown child deep in the tree",
			lines:    at("@endpoint GET /users {", "@response 200 {", "@bogus x", "}", "}"),
			node:     endpointNode,
			annotate: "@endpoint",
			wantErr:  "unknown annotation @bogus in @response",
			wantLine: 3,
		},
		{
			name:     "unknown child inline",
			lines:    at("@field { @bogus x }"),
			node:     fieldNode,
			annotate: "@field",
			wantErr:  "unknown annotation @bogus in @field",
			wantLine: 1,
		},
		{
			name:     "repeat of a non-repeatable child",
			lines:    at("@field {", "@format email", "@format uuid", "}"),
			node:     fieldNode,
			annotate: "@field",
			wantErr:  "@format appears multiple times but is not repeatable",
			wantLine: 3,
		},
		{
			name:     "required child missing",
			lines:    at("@api {}"),
			node:     apiNode,
			annotate: "@api",
			wantErr:  "@api cannot be empty (has required children)",
			wantLine: 1,
		},
		{
			name:     "unescaped at in a value",
			lines:    at("@field { @description mail me at foo@example.com }"),
			node:     fieldNode,
			annotate: "@field",
			wantErr:  "unknown annotation @example.com in @field",
			wantLine: 1,
		},
		{
			name:     "unescaped brace in a value",
			lines:    at("@endpoint GET /users {", "@summary a{b}c", "}"),
			node:     endpointNode,
			annotate: "@endpoint",
			wantErr:  `unescaped '{' in @summary value`,
			wantLine: 2,
		},
		{
			name:     "nested block written inline",
			lines:    at("@api { @contact { @name Team } }"),
			node:     apiNode,
			annotate: "@api",
			wantErr:  "nested blocks cannot be inlined",
			wantLine: 1,
		},
		{
			name:     "unbalanced braces",
			lines:    at("@field {", "@format email"),
			node:     fieldNode,
			annotate: "@field",
			wantErr:  "unbalanced braces",
			wantLine: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAnnotationBlock(tt.lines, tt.annotate, tt.node)
			if err == nil {
				t.Fatalf("ParseAnnotationBlock() succeeded, want error %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}

			var perr *Error
			if !errors.As(err, &perr) {
				t.Fatalf("error %v is not a *parser.Error", err)
			}
			if perr.Pos.Line != tt.wantLine {
				t.Errorf("error line = %d, want %d (error: %v)", perr.Pos.Line, tt.wantLine, err)
			}
			if !strings.HasPrefix(err.Error(), "fixture.go:") {
				t.Errorf("error %q does not start with the position", err)
			}
		})
	}
}

func TestParseAnnotationBlock_RawPatternKeepsBraces(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"unescaped quantifier", "@field { @pattern ^[A-Z]{2}$ }", "^[A-Z]{2}$"},
		{"escaped quantifier", `@field { @pattern ^[A-Z]\{2\}$ }`, `^[A-Z]\{2\}$`},
		{"digit classes", `@field { @pattern ^\d{3}-\d{4}$ }`, `^\d{3}-\d{4}$`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseAnnotationBlock(at(tt.line), "@field", fieldNode)
			if err != nil {
				t.Fatalf("ParseAnnotationBlock() error = %v", err)
			}
			if got := parsed.ChildValue("@pattern"); got != tt.want {
				t.Errorf("@pattern = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAnnotationBlock_EscapedValues(t *testing.T) {
	parsed, err := ParseAnnotationBlock(at(`@field { @example admin\@example.com @description braces \{here\} }`), "@field", fieldNode)
	if err != nil {
		t.Fatalf("ParseAnnotationBlock() error = %v", err)
	}
	if got := parsed.ChildValue("@example"); got != "admin@example.com" {
		t.Errorf("@example = %q", got)
	}
	if got := parsed.ChildValue("@description"); got != "braces {here}" {
		t.Errorf("@description = %q", got)
	}
}

func TestParseAnnotationBlock_UnknownRootAnnotation(t *testing.T) {
	_, err := ParseAnnotationBlock(at("@nope"), "@nope", Grammar["@nope"])
	if err == nil || !strings.Contains(err.Error(), "unknown annotation: @nope") {
		t.Fatalf("error = %v, want an unknown-annotation error", err)
	}
}

func TestError_Rendering(t *testing.T) {
	positioned := &Error{Pos: token.Position{Filename: "x.go", Line: 12, Column: 3}, Msg: "boom"}
	if got := positioned.Error(); got != "x.go:12:3: boom" {
		t.Errorf("Error() = %q, want %q", got, "x.go:12:3: boom")
	}

	bare := &Error{Msg: "boom"}
	if got := bare.Error(); got != "boom" {
		t.Errorf("Error() = %q, want %q", got, "boom")
	}

	// Wrapping keeps the innermost position, so context reads before the
	// detail rather than after the position.
	wrapped := wrapf(token.Position{Filename: "y.go", Line: 1, Column: 1}, positioned, "failed to parse @field for User.ID")
	want := "x.go:12:3: failed to parse @field for User.ID: boom"
	if got := wrapped.Error(); got != want {
		t.Errorf("wrapf() = %q, want %q", got, want)
	}
}
