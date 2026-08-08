package parser

import (
	"iter"
	"strings"
)

// This file is the escape engine. Annotation text supports four escapes:
//
//	\{ → {   \} → }   \@ → @   \\ → \
//
// Every operation that has to tell structure from text — brace counting, @
// splitting, unescaping, special-character validation, block-opener detection —
// goes through scan, so all of them agree on what is escaped and what is not.
// A second opinion here is how the same annotation ends up parsed two ways.

// char is one character of annotation text as the engine sees it.
type char struct {
	// Pos is the byte index in the scanned string. For an escape sequence it
	// is the index of the backslash.
	Pos int

	// Val is the character value: for an escape sequence, the character the
	// sequence stands for.
	Val byte

	// Escaped reports that Val arrived as a two-byte escape sequence, and so is
	// literal text rather than structure.
	Escaped bool
}

// isEscapeAt reports whether content[i:i+2] is a recognized escape sequence
// (\{, \}, \@, \\).
func isEscapeAt(content string, i int) bool {
	if i+1 >= len(content) || content[i] != '\\' {
		return false
	}
	next := content[i+1]
	return next == '{' || next == '}' || next == '@' || next == '\\'
}

// scan walks content left to right, yielding one char per source character with
// escape sequences collapsed into the single character they stand for.
//
// A backslash that does not begin a recognized sequence is yielded as itself,
// so a Windows path or a regex escape such as \d survives untouched.
func scan(content string) iter.Seq[char] {
	return func(yield func(char) bool) {
		for i := 0; i < len(content); {
			if isEscapeAt(content, i) {
				if !yield(char{Pos: i, Val: content[i+1], Escaped: true}) {
					return
				}
				i += 2
				continue
			}
			if !yield(char{Pos: i, Val: content[i], Escaped: false}) {
				return
			}
			i++
		}
	}
}

// UnescapeValue converts escape sequences to the characters they stand for.
//
// It reads the same left-to-right scan as everything else rather than
// substituting one sequence at a time, which needed a placeholder to keep \\
// from being unescaped twice and could not agree with the scanners by
// construction.
func UnescapeValue(value string) string {
	if !strings.ContainsRune(value, '\\') {
		return value
	}

	var b strings.Builder
	b.Grow(len(value))
	for c := range scan(value) {
		b.WriteByte(c.Val)
	}
	return b.String()
}

// CountUnescapedBraces returns the brace depth of content, ignoring escaped braces.
// Zero means balanced; positive means more openers than closers.
func CountUnescapedBraces(content string) int {
	depth := 0
	for c := range scan(content) {
		if c.Escaped {
			continue
		}
		switch c.Val {
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
	for c := range scan(content) {
		if c.Escaped {
			continue
		}
		if strings.IndexByte(targets, c.Val) >= 0 {
			return c.Val, c.Pos
		}
	}
	return 0, -1
}

// LastUnescaped returns the index of the last unescaped occurrence of target in
// content, or -1 when there is none.
//
// It is how a block written on one line finds its closing brace. Counting
// cannot: a RawValue child such as @pattern may carry braces of its own, and
// only the final brace on the line can be the one that closes the block.
func LastUnescaped(content string, target byte) int {
	idx := -1
	for c := range scan(content) {
		if !c.Escaped && c.Val == target {
			idx = c.Pos
		}
	}
	return idx
}

// SplitOnUnescapedAt splits content at every unescaped @. This is how the
// children of a block written on one line are separated:
// "@description Email @format email".
func SplitOnUnescapedAt(content string) []string {
	var parts []string
	start := 0
	for c := range scan(content) {
		if c.Escaped || c.Val != '@' {
			continue
		}
		parts = append(parts, content[start:c.Pos])
		start = c.Pos + 1
	}
	return append(parts, content[start:])
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
