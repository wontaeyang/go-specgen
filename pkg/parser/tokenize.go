package parser

import (
	"go/token"
	"iter"
	"strings"
)

// This file is the tokenizer: it turns comment lines into an Annotation tree,
// guided by the grammar. It owns the one escape engine everything else builds
// on.
//
// Escape sequences inside annotation text:
//
//	\{ → {   \} → }   \@ → @   \\ → \
//
// Every escape-aware operation (brace counting, @ splitting, unescaping,
// special-character validation) goes through scan, so they all agree on what
// is escaped and what is structure.

// Line is one comment line with the source position it came from. Lines are
// threaded through the tokenizer so every error can name its own line.
type Line struct {
	Text string
	Pos  token.Position
}

// Annotation is a parsed annotation, with its children.
type Annotation struct {
	// Name is the annotation name, including the @.
	Name string

	// Pos is the position of the annotation's opening line.
	Pos token.Position

	// Metadata is the opening-line value of an annotation declared with
	// HasMetadata: the "200" of "@response 200 {", the scheme name of
	// "@securityScheme bearerAuth {".
	Metadata string

	// Value is the value of a value or reference annotation.
	Value string

	// Children holds non-repeatable children by name.
	Children map[string]*Annotation

	// Repeated holds repeatable children by name, in source order.
	Repeated map[string][]*Annotation
}

func newAnnotation(name string, pos token.Position) *Annotation {
	return &Annotation{
		Name:     name,
		Pos:      pos,
		Children: make(map[string]*Annotation),
		Repeated: make(map[string][]*Annotation),
	}
}

// Child returns a child annotation by name, or nil if absent.
func (a *Annotation) Child(name string) *Annotation {
	if a == nil {
		return nil
	}
	return a.Children[name]
}

// HasChild reports whether a child annotation is present. This is how flag
// annotations are read: presence is the value.
func (a *Annotation) HasChild(name string) bool {
	return a.Child(name) != nil
}

// ChildValue returns the value of a child annotation, or "" if absent.
func (a *Annotation) ChildValue(name string) string {
	if child := a.Child(name); child != nil {
		return child.Value
	}
	return ""
}

// ChildPos returns the position of a child annotation, falling back to the
// parent's position when the child is absent.
func (a *Annotation) ChildPos(name string) token.Position {
	if child := a.Child(name); child != nil {
		return child.Pos
	}
	return a.Pos
}

// RepeatedChildren returns every instance of a repeatable child, in source
// order.
func (a *Annotation) RepeatedChildren(name string) []*Annotation {
	if a == nil {
		return nil
	}
	return a.Repeated[name]
}

// Char is one character of annotation text as the escape engine sees it.
type Char struct {
	// Pos is the byte index in the scanned string. For an escape sequence it
	// is the index of the backslash.
	Pos int

	// Val is the character value: for an escape sequence, the character the
	// sequence stands for.
	Val byte

	// Escaped reports that Val arrived as a two-byte escape sequence, and so
	// is literal text rather than structure.
	Escaped bool
}

// isEscapeAt reports whether content[i:i+2] is a recognized escape sequence.
func isEscapeAt(content string, i int) bool {
	if i+1 >= len(content) || content[i] != '\\' {
		return false
	}
	switch content[i+1] {
	case '{', '}', '@', '\\':
		return true
	}
	return false
}

// scan walks content left to right, yielding one Char per source character —
// with escape sequences collapsed into the single character they stand for.
// A backslash that does not start a recognized sequence is yielded as itself.
func scan(content string) iter.Seq[Char] {
	return func(yield func(Char) bool) {
		for i := 0; i < len(content); {
			if isEscapeAt(content, i) {
				if !yield(Char{Pos: i, Val: content[i+1], Escaped: true}) {
					return
				}
				i += 2
				continue
			}
			if !yield(Char{Pos: i, Val: content[i], Escaped: false}) {
				return
			}
			i++
		}
	}
}

// UnescapeValue converts escape sequences to the characters they stand for.
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

// CountUnescapedBraces returns the brace depth of content, ignoring escaped
// braces. Zero means balanced; positive means more openers than closers.
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

// FindUnescaped returns the first unescaped character of content that appears
// in targets, and its index. Returns (0, -1) when there is none.
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

// FindUnescapedSpecial scans content for an unescaped special character
// ({, } or @). Annotation values must escape these if they are meant
// literally, so finding one is an error everywhere except raw values.
func FindUnescapedSpecial(content string) (byte, bool) {
	b, idx := FindUnescaped(content, "{}@")
	return b, idx >= 0
}

// ContainsUnescapedBrace reports whether content has an unescaped { or }.
func ContainsUnescapedBrace(content string) bool {
	_, idx := FindUnescaped(content, "{}")
	return idx >= 0
}

// SplitOnUnescapedAt splits content at every unescaped @. This is how inline
// children are separated: "@description Email @format email".
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

// StartsWithUnescapedAt reports whether a line begins an annotation, i.e.
// starts with an @ that is not escaped. Used to find where a multi-line value
// ends and the next sibling begins.
func StartsWithUnescapedAt(line string) bool {
	line = strings.TrimSpace(line)
	return len(line) > 0 && line[0] == '@'
}

// findBlockOpener returns the index of a block-opening brace in line, or -1.
//
// A block opener is a brace preceded by a space or tab. That is what keeps a
// path parameter apart from a block: "/users/{id}" has no space before its
// brace, "@endpoint GET /users/{id} {" does.
func findBlockOpener(line string) int {
	for c := range scan(line) {
		if c.Escaped {
			continue
		}
		if c.Val != ' ' && c.Val != '\t' {
			continue
		}
		if c.Pos+1 < len(line) && line[c.Pos+1] == '{' {
			return c.Pos + 1
		}
	}
	return -1
}

// countBracesFrom returns the brace depth of line starting at startPos.
func countBracesFrom(line string, startPos int) int {
	return CountUnescapedBraces(line[startPos:])
}

// extractAnnotationName returns the annotation name a line starts with:
// "@field {" → "@field", "@title My API" → "@title". Returns "" when the line
// is not an annotation.
func extractAnnotationName(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@") {
		return ""
	}
	if i := strings.IndexAny(line, " \t{"); i > 0 {
		return line[:i]
	}
	return line
}

// ExtractMetadata returns the opening-line value of an annotation: everything
// between the annotation name and the block opener.
//
//	"@endpoint GET /users/{id} {" → "GET /users/{id}"
//	"@server https://api.com {"   → "https://api.com"
func ExtractMetadata(line, annotationName string) string {
	line = strings.TrimPrefix(line, annotationName)

	// Cut at the block opener, if any, keeping path params like {id} intact.
	if pos := findBlockOpener(line); pos >= 0 {
		line = line[:pos-1]
	}
	return UnescapeValue(strings.TrimSpace(line))
}

// ParseBracedBlock returns the content of the { } block that opens somewhere in
// lines, with each content line trimmed and carrying its own position.
//
// Returns nil content when the lines contain no block opener at all — a flag
// or a block written without braces.
func ParseBracedBlock(lines []Line) ([]Line, error) {
	openLine, openPos := -1, -1
	for i, line := range lines {
		if pos := findBlockOpener(line.Text); pos >= 0 {
			openLine, openPos = i, pos
			break
		}
	}
	if openLine == -1 {
		return nil, nil
	}

	var content []Line
	depth := 0

	for i := openLine; i < len(lines); i++ {
		line := lines[i]
		text := line.Text

		if i == openLine {
			// Only the part from the opening brace onward is block content;
			// anything before it is the annotation name and its metadata.
			depth += countBracesFrom(text, openPos)
			text = text[openPos+1:]
		} else {
			depth += CountUnescapedBraces(text)
		}

		text = strings.TrimSpace(text)

		if depth == 0 {
			// The block closed on this line. For a single-line block
			// ("{ @description foo }") the content sits before the closer.
			if i == openLine && text != "" {
				text = strings.TrimSpace(strings.TrimSuffix(text, "}"))
				if text != "" {
					content = append(content, Line{Text: text, Pos: line.Pos})
				}
			}
			return content, nil
		}

		if text != "" {
			content = append(content, Line{Text: text, Pos: line.Pos})
		}

		if depth < 0 {
			return nil, errorf(line.Pos, "unbalanced braces at line: %s", line.Text)
		}
	}

	return nil, errorf(startPos(lines), "unbalanced braces: depth=%d", depth)
}

// ParseAnnotationBlock parses one annotation — its metadata, its value, and
// its children — from the comment lines it spans.
//
// Single-line ("@field { @format email }") and multi-line blocks go through
// the same path: the difference is only how the children are separated once
// the block content has been extracted.
func ParseAnnotationBlock(lines []Line, annotationName string, node *GrammarNode) (*Annotation, error) {
	if node == nil {
		return nil, errorf(startPos(lines), "unknown annotation: %s", annotationName)
	}

	result := newAnnotation(annotationName, startPos(lines))

	if node.Type == FlagAnnotation {
		return result, nil
	}

	if node.HasMetadata && len(lines) > 0 {
		result.Metadata = ExtractMetadata(lines[0].Text, annotationName)
	}

	if node.Type == ValueAnnotation || node.Type == ReferenceAnnotation {
		if len(lines) == 0 {
			return result, nil
		}

		value := strings.TrimSpace(strings.TrimPrefix(lines[0].Text, annotationName))

		// Continuation lines belong to the value only for annotations that
		// declare multi-line support.
		if node.SupportsMultiline {
			for _, line := range lines[1:] {
				if continuation := strings.TrimSpace(line.Text); continuation != "" {
					value += "\n" + continuation
				}
			}
		}

		resolved, err := resolveValue(value, node, result.Pos)
		if err != nil {
			return nil, err
		}
		result.Value = resolved
		return result, nil
	}

	content, err := ParseBracedBlock(lines)
	if err != nil {
		return nil, wrapf(result.Pos, err, "failed to parse %s", annotationName)
	}

	if len(content) == 0 {
		if !node.CanBeEmpty() {
			return nil, errorf(result.Pos, "%s cannot be empty (has required children)", annotationName)
		}
		return result, nil
	}

	// Route by source syntax: one input line means the user wrote the whole
	// block inline, so children are separated by @ rather than by newline.
	if len(lines) == 1 {
		err = parseInlineChildren(content[0], node, result)
	} else {
		err = parseChildren(content, node, result)
	}
	if err != nil {
		return nil, wrapf(result.Pos, err, "failed to parse %s children", annotationName)
	}

	return result, nil
}

// parseChildren parses the children of a multi-line block, one annotation per
// line (plus the continuation lines and nested blocks they own).
func parseChildren(lines []Line, parent *GrammarNode, result *Annotation) error {
	for i := 0; i < len(lines); {
		line := lines[i]

		// Lines that do not start an annotation are leftovers from a value we
		// have already consumed.
		if !strings.HasPrefix(line.Text, "@") {
			i++
			continue
		}

		annotationName := extractAnnotationName(line.Text)
		if annotationName == "" {
			i++
			continue
		}

		childNode := parent.GetChild(annotationName)
		if childNode == nil {
			return errorf(line.Pos, "unknown annotation %s in %s", annotationName, parent.Name)
		}

		childLines := []Line{line}
		i++

		if blockPos := findBlockOpener(line.Text); blockPos >= 0 {
			// A nested block: keep taking lines until its braces balance.
			depth := countBracesFrom(line.Text, blockPos)
			for i < len(lines) && depth > 0 {
				childLines = append(childLines, lines[i])
				depth += CountUnescapedBraces(lines[i].Text)
				i++
			}
		} else if childNode.SupportsMultiline {
			// A multi-line value: keep taking lines until the next sibling.
			for i < len(lines) {
				next := lines[i]
				if StartsWithUnescapedAt(next.Text) && parent.HasChild(extractAnnotationName(next.Text)) {
					break
				}
				childLines = append(childLines, next)
				i++
			}
		}

		parsed, err := ParseAnnotationBlock(childLines, annotationName, childNode)
		if err != nil {
			return err
		}
		if err := addChild(result, childNode, parsed); err != nil {
			return err
		}
	}

	return nil
}

// parseInlineChildren parses the children of a block written on one line:
// "@description User email @format email @example test\@example.com".
func parseInlineChildren(content Line, parent *GrammarNode, result *Annotation) error {
	for _, part := range SplitOnUnescapedAt(content.Text) {
		part = strings.TrimSpace(part)

		// The text before the first @ is empty for well-formed content. When
		// it is not, it is stray text, and reading it as an annotation name is
		// what surfaces it as an error.
		if part == "" {
			continue
		}

		annotationName, value := "@"+part, ""
		if spaceIdx := strings.IndexAny(part, " \t"); spaceIdx >= 0 {
			annotationName = "@" + part[:spaceIdx]
			value = strings.TrimSpace(part[spaceIdx+1:])
		}

		childNode := parent.GetChild(annotationName)
		if childNode == nil {
			return errorf(content.Pos, "unknown annotation %s in %s", annotationName, parent.Name)
		}

		// A child that opens its own block cannot be written inline: its
		// braces would be indistinguishable from the parent's.
		if len(childNode.Children) > 0 && ContainsUnescapedBrace(value) {
			return errorf(content.Pos, "nested blocks cannot be inlined, use multi-line format: %s", annotationName)
		}

		resolved, err := resolveValue(value, childNode, content.Pos)
		if err != nil {
			return err
		}

		parsed := newAnnotation(annotationName, content.Pos)
		parsed.Value = resolved
		if childNode.HasMetadata {
			parsed.Metadata = resolved
		}

		if err := addChild(result, childNode, parsed); err != nil {
			return err
		}
	}

	return nil
}

// addChild files a parsed child under its parent, keeping repeatable children
// in source order and rejecting repeats of the others.
func addChild(result *Annotation, node *GrammarNode, parsed *Annotation) error {
	if node.Repeatable {
		result.Repeated[node.Name] = append(result.Repeated[node.Name], parsed)
		return nil
	}
	if _, exists := result.Children[node.Name]; exists {
		return errorf(parsed.Pos, "%s appears multiple times but is not repeatable", node.Name)
	}
	result.Children[node.Name] = parsed
	return nil
}

// resolveValue applies the leaf-value rules: raw values pass through verbatim,
// everything else must escape the special characters and is then unescaped.
func resolveValue(value string, node *GrammarNode, pos token.Position) (string, error) {
	if node.RawValue {
		return value, nil
	}
	if ch, found := FindUnescapedSpecial(value); found {
		return "", errorf(pos, "unescaped %q in %s value: use \\%c for literals", ch, node.Name, ch)
	}
	return UnescapeValue(value), nil
}

// startPos returns the position of the first line, or the zero position.
func startPos(lines []Line) token.Position {
	if len(lines) == 0 {
		return token.Position{}
	}
	return lines[0].Pos
}
