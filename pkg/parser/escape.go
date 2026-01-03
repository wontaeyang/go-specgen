package parser

import "strings"

// Escape sequence support for annotation values.
// Supported escapes:
//   \{ → {
//   \} → }
//   \@ → @
//   \\ → \

// atPlaceholder is used to protect escaped @ during splitting
const atPlaceholder = "\x00AT\x00"

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

// CountUnescapedBraces counts brace depth ignoring escaped braces.
// Returns the final depth and whether braces are balanced.
func CountUnescapedBraces(content string) (depth int, balanced bool) {
	i := 0
	for i < len(content) {
		if content[i] == '\\' && i+1 < len(content) {
			next := content[i+1]
			if next == '{' || next == '}' || next == '@' || next == '\\' {
				i += 2 // skip escape sequence
				continue
			}
		}
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
		}
		i++
	}
	return depth, depth == 0
}

// ProtectEscapedAt replaces \@ with placeholder before splitting by @.
func ProtectEscapedAt(content string) string {
	return strings.ReplaceAll(content, "\\@", atPlaceholder)
}

// RestoreEscapedAt restores placeholder back to \@.
func RestoreEscapedAt(content string) string {
	return strings.ReplaceAll(content, atPlaceholder, "\\@")
}

// ContainsUnescapedBrace checks if string has unescaped { or }.
func ContainsUnescapedBrace(content string) bool {
	i := 0
	for i < len(content) {
		if content[i] == '\\' && i+1 < len(content) {
			next := content[i+1]
			if next == '{' || next == '}' || next == '@' || next == '\\' {
				i += 2
				continue
			}
		}
		if content[i] == '{' || content[i] == '}' {
			return true
		}
		i++
	}
	return false
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
