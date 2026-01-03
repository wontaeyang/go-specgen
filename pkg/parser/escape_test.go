package parser

import "testing"

func TestUnescapeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no escapes",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "escaped opening brace",
			input:    `^[A-Z]\{2\}$`,
			expected: "^[A-Z]{2}$",
		},
		{
			name:     "escaped closing brace",
			input:    `\}`,
			expected: "}",
		},
		{
			name:     "escaped at symbol",
			input:    `user\@example.com`,
			expected: "user@example.com",
		},
		{
			name:     "escaped backslash",
			input:    `path\\to\\file`,
			expected: `path\to\file`,
		},
		{
			name:     "mixed escapes",
			input:    `\{a\@b\\c\}`,
			expected: `{a@b\c}`,
		},
		{
			name:     "JSON example",
			input:    `\{"key": "value"\}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "regex quantifier",
			input:    `^[a-z]\{3,5\}$`,
			expected: "^[a-z]{3,5}$",
		},
		{
			name:     "multiple emails",
			input:    `admin\@example.com, support\@example.com`,
			expected: "admin@example.com, support@example.com",
		},
		{
			name:     "backslash before non-escape char",
			input:    `\n\t`,
			expected: `\n\t`, // not escape sequences, left as-is
		},
		{
			name:     "double backslash followed by brace",
			input:    `\\\{`,
			expected: `\{`, // \\ becomes \, then \{ becomes {? No - \\ becomes \, { stays
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnescapeValue(tt.input)
			if got != tt.expected {
				t.Errorf("UnescapeValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCountUnescapedBraces(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantDepth    int
		wantBalanced bool
	}{
		{
			name:         "no braces",
			content:      "hello world",
			wantDepth:    0,
			wantBalanced: true,
		},
		{
			name:         "balanced braces",
			content:      "{ hello }",
			wantDepth:    0,
			wantBalanced: true,
		},
		{
			name:         "unbalanced open",
			content:      "{ hello",
			wantDepth:    1,
			wantBalanced: false,
		},
		{
			name:         "unbalanced close",
			content:      "hello }",
			wantDepth:    -1,
			wantBalanced: false,
		},
		{
			name:         "nested braces",
			content:      "{ { } }",
			wantDepth:    0,
			wantBalanced: true,
		},
		{
			name:         "escaped braces ignored",
			content:      `\{ hello \}`,
			wantDepth:    0,
			wantBalanced: true,
		},
		{
			name:         "mixed real and escaped",
			content:      `{ \{2\} }`,
			wantDepth:    0,
			wantBalanced: true,
		},
		{
			name:         "regex pattern in inline",
			content:      `@field { @pattern ^[A-Z]\{2\}$ }`,
			wantDepth:    0,
			wantBalanced: true,
		},
		{
			name:         "escaped backslash then brace",
			content:      `\\{`,
			wantDepth:    1,
			wantBalanced: false, // \\ is escape, { is real brace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDepth, gotBalanced := CountUnescapedBraces(tt.content)
			if gotDepth != tt.wantDepth {
				t.Errorf("CountUnescapedBraces(%q) depth = %d, want %d", tt.content, gotDepth, tt.wantDepth)
			}
			if gotBalanced != tt.wantBalanced {
				t.Errorf("CountUnescapedBraces(%q) balanced = %v, want %v", tt.content, gotBalanced, tt.wantBalanced)
			}
		})
	}
}

func TestProtectAndRestoreEscapedAt(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "single escaped at",
			content: `user\@example.com`,
		},
		{
			name:    "multiple escaped at",
			content: `admin\@example.com and support\@test.org`,
		},
		{
			name:    "no escaped at",
			content: "hello world",
		},
		{
			name:    "mixed",
			content: `@description Contact \@admin for help`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protected := ProtectEscapedAt(tt.content)
			restored := RestoreEscapedAt(protected)
			if restored != tt.content {
				t.Errorf("Round-trip failed: got %q, want %q", restored, tt.content)
			}
		})
	}
}

func TestContainsUnescapedBrace(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "no braces",
			content:  "hello world",
			expected: false,
		},
		{
			name:     "unescaped open brace",
			content:  "hello { world",
			expected: true,
		},
		{
			name:     "unescaped close brace",
			content:  "hello } world",
			expected: true,
		},
		{
			name:     "escaped braces only",
			content:  `\{2\}`,
			expected: false,
		},
		{
			name:     "mixed",
			content:  `{ \{2\} }`,
			expected: true,
		},
		{
			name:     "regex pattern",
			content:  `^[A-Z]\{2\}$`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsUnescapedBrace(tt.content)
			if got != tt.expected {
				t.Errorf("ContainsUnescapedBrace(%q) = %v, want %v", tt.content, got, tt.expected)
			}
		})
	}
}

func TestStartsWithUnescapedAt(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "starts with @",
			line:     "@description hello",
			expected: true,
		},
		{
			name:     "starts with escaped @",
			line:     `\@example.com`,
			expected: false,
		},
		{
			name:     "starts with space then @",
			line:     "  @field",
			expected: true,
		},
		{
			name:     "starts with space then escaped @",
			line:     `  \@admin`,
			expected: false,
		},
		{
			name:     "no @",
			line:     "hello world",
			expected: false,
		},
		{
			name:     "empty",
			line:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StartsWithUnescapedAt(tt.line)
			if got != tt.expected {
				t.Errorf("StartsWithUnescapedAt(%q) = %v, want %v", tt.line, got, tt.expected)
			}
		})
	}
}
