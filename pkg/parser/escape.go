package parser

import (
	"iter"
	"strings"
)

// Escape sequence support for annotation values.
// Supported escapes:
//   \{ → {
//   \} → }
//   \@ → @
//   \\ → \

// isEscapeAt reports whether content[i:i+2] is a recognized escape sequence
// (\{, \}, \@, \\). Callers that respect escapes should advance by 2 when
// this returns true.
func isEscapeAt(content string, i int) bool {
	if i+1 >= len(content) || content[i] != '\\' {
		return false
	}
	next := content[i+1]
	return next == '{' || next == '}' || next == '@' || next == '\\'
}

// unescapedBytes iterates content yielding only the bytes that are NOT part
// of an escape sequence. Escape pairs (\{, \}, \@, \\) are consumed atomically
// and skipped — the iterator never yields the backslash or its successor.
// Yielded positions are absolute indices in content.
func unescapedBytes(content string) iter.Seq2[int, byte] {
	return func(yield func(int, byte) bool) {
		i := 0
		for i < len(content) {
			if isEscapeAt(content, i) {
				i += 2
				continue
			}
			if !yield(i, content[i]) {
				return
			}
			i++
		}
	}
}

// UnescapeValue converts escape sequences to literal characters.
// Order matters: handle \\ first to avoid double-unescaping.
func UnescapeValue(value string) string {
	// First replace \\ with a placeholder to avoid issues with other escapes
	const backslashPlaceholder = "\x00BS\x00"
	result := strings.ReplaceAll(value, "\\\\", backslashPlaceholder)
	result = strings.ReplaceAll(result, "\\{", "{")
	result = strings.ReplaceAll(result, "\\}", "}")
	result = strings.ReplaceAll(result, "\\@", "@")
	result = strings.ReplaceAll(result, backslashPlaceholder, "\\")
	return result
}

// CountUnescapedBraces returns the brace depth of content, ignoring escaped braces.
// Zero means balanced; positive means more openers than closers.
func CountUnescapedBraces(content string) int {
	depth := 0
	for _, b := range unescapedBytes(content) {
		switch b {
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	return depth
}

// FindUnescaped returns the first unescaped byte in content that appears in
// targets, and its index. Returns (0, -1) if none found.
func FindUnescaped(content, targets string) (byte, int) {
	for i, b := range unescapedBytes(content) {
		if strings.IndexByte(targets, b) >= 0 {
			return b, i
		}
	}
	return 0, -1
}

// SplitOnUnescapedAt splits content at every @ that isn't part of an escape
// sequence. Escapes (\@, \\, \{, \}) are consumed atomically left-to-right,
// matching the grammar used by UnescapeValue and FindUnescapedSpecial.
func SplitOnUnescapedAt(content string) []string {
	var parts []string
	start := 0
	for i, b := range unescapedBytes(content) {
		if b == '@' {
			parts = append(parts, content[start:i])
			start = i + 1
		}
	}
	parts = append(parts, content[start:])
	return parts
}

// ContainsUnescapedBrace checks if string has unescaped { or }.
func ContainsUnescapedBrace(content string) bool {
	_, idx := FindUnescaped(content, "{}")
	return idx >= 0
}

// FindUnescapedSpecial scans content for an unescaped special character ({, }, @).
// Returns the offending character and true if found. Used to validate annotation
// values where these chars must be escaped (\{, \}, \@) if intended as literals.
func FindUnescapedSpecial(content string) (byte, bool) {
	b, idx := FindUnescaped(content, "{}@")
	return b, idx >= 0
}

// StartsWithUnescapedAt checks if a line starts with an unescaped @ symbol.
// Used for multi-line parsing to detect annotation boundaries.
func StartsWithUnescapedAt(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return false
	}
	// Check if starts with @ (not escaped)
	if line[0] == '@' {
		return true
	}
	// Check if starts with \@ (escaped - not a real annotation)
	if len(line) >= 2 && line[0] == '\\' && line[1] == '@' {
		return false
	}
	return false
}
