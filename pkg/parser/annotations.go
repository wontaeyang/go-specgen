package parser

import (
	"fmt"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/annotation"
)

// ParsedAnnotation represents a parsed annotation with its content
type ParsedAnnotation struct {
	// Name is the annotation name (e.g., "@api", "@field")
	Name string

	// Metadata is extracted from the opening line for annotations with HasMetadata
	// Example: "@endpoint GET /users" -> "GET /users"
	// Example: "@server https://api.com" -> "https://api.com"
	Metadata string

	// Children are nested annotations
	Children map[string]*ParsedAnnotation

	// RepeatedChildren are children that can appear multiple times
	// Key is annotation name, value is slice of parsed annotations
	RepeatedChildren map[string][]*ParsedAnnotation

	// Value is the annotation value for ValueAnnotation types
	Value string

	// IsFlag indicates if this is a FlagAnnotation (e.g., @deprecated)
	IsFlag bool

	// Lines are the original comment lines for debugging
	Lines []string
}

// findBlockOpener finds the position of block delimiter " {" or "\t{" in a line.
// Block delimiters are distinguished from path parameters by the preceding space/tab.
// Path params like {id} have no space before the brace.
// Returns the position of '{' or -1 if no block delimiter found.
func findBlockOpener(line string) int {
	for c := range scan(line) {
		if c.Escaped {
			continue
		}
		if c.Val != ' ' && c.Val != '\t' {
			continue
		}
		// The brace is read from the raw line on purpose: an escaped \{ leaves
		// a backslash here, so it is not a block opener.
		if c.Pos+1 < len(line) && line[c.Pos+1] == '{' {
			return c.Pos + 1
		}
	}
	return -1
}

// openerDepth returns the brace depth contributed by a block's opening line,
// counting from the opener at startPos.
//
// Only the braces before the first child annotation are structure. Everything
// from that annotation on is its text, and a value is free to contain braces
// that close nothing — which is why counting the whole line is wrong.
func openerDepth(line string, startPos int) int {
	head := line[startPos:]
	if _, at := FindUnescaped(head, "@"); at >= 0 {
		head = head[:at]
	}
	return CountUnescapedBraces(head)
}

// startsWithRawValue reports whether line begins with a child of node whose
// value is raw.
//
// A raw value is a grammar of its own — @pattern holds a regex, where {2} is a
// quantifier and } may appear unpaired. Those braces are text, so a line
// carrying one must not move the block's depth.
func startsWithRawValue(line string, node *annotation.Def) bool {
	if node == nil {
		return false
	}
	child := node.GetChild(extractAnnotationName(line))
	return child != nil && child.RawValue
}

// ParseBracedBlock extracts the content of the { } block that opens somewhere in
// lines. Returns nil content when there is no block at all — a marker or a flag.
//
// The opener is found by position: a brace preceded by a space or tab. That is
// what keeps a path parameter apart from a block, since "/users/{id}" has no
// space before its brace and "@endpoint GET /users/{id} {" does.
//
// node is the annotation whose block this is. It is needed because finding the
// closing brace is not a counting problem: a child may hold a raw value with
// braces of its own. Both rules below exist for that reason.
func ParseBracedBlock(lines []string, node *annotation.Def) ([]string, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	openLineIndex, openBracePos := -1, -1
	for i, line := range lines {
		if pos := findBlockOpener(line); pos >= 0 {
			openLineIndex, openBracePos = i, pos
			break
		}
	}

	// No block delimiter found - might be a marker annotation or flag
	if openLineIndex == -1 {
		return nil, nil
	}

	first := lines[openLineIndex]

	// A block that closes on its opening line ends at the last unescaped brace
	// of that line, whatever the depth in between says. "@field { @pattern
	// ^a}b$ }" is one block holding one regex, not a block that closed early.
	if closePos := LastUnescaped(first, '}'); closePos > openBracePos {
		content := strings.TrimSpace(first[openBracePos+1 : closePos])
		if content == "" {
			return nil, nil
		}
		return []string{content}, nil
	}

	// A multi-line block. Depth is counted from here on, skipping the lines
	// whose value is raw, so a regex on its own line closes nothing.
	var content []string
	braceDepth := openerDepth(first, openBracePos)

	if rest := strings.TrimSpace(first[openBracePos+1:]); rest != "" {
		content = append(content, rest)
	}

	for i := openLineIndex + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if !startsWithRawValue(line, node) {
			braceDepth += CountUnescapedBraces(line)
		}

		// The closing line is the block's, not its content's.
		if braceDepth == 0 {
			// A line that closes the block while carrying other text is almost
			// always a value with an unescaped brace in it. Returning here
			// would drop that line and every line under it, and say nothing.
			if line != "}" {
				return nil, fmt.Errorf("%q closes the block but carries other text; write \\} for a literal brace in a value", line)
			}
			return content, nil
		}
		if braceDepth < 0 {
			return nil, fmt.Errorf("unbalanced braces at line: %s", lines[i])
		}

		if line != "" {
			content = append(content, line)
		}
	}

	return nil, fmt.Errorf("unbalanced braces: depth=%d", braceDepth)
}

// ExtractMetadata extracts metadata from the opening line of an annotation
// Example: "@endpoint GET /users/{id} {" -> "GET /users/{id}"
// Example: "@server https://api.com {" -> "https://api.com"
// Uses position-based detection to preserve path parameters like {id}.
func ExtractMetadata(line, annotationName string) string {
	// Remove annotation name
	line = strings.TrimPrefix(line, annotationName)

	// Find block opener position (space + brace) before trimming
	blockPos := findBlockOpener(line)
	if blockPos >= 0 {
		// Take everything before the block opener (excluding the space before it)
		line = line[:blockPos-1]
	}

	return UnescapeValue(strings.TrimSpace(line))
}

// resolveValue applies leaf-value rules for an annotation value.
// RawValue nodes pass through verbatim; all others are validated for
// unescaped specials ({, }, @) and then unescaped.
func resolveValue(value string, node *annotation.Def, annotationName string) (string, error) {
	if node.RawValue {
		return value, nil
	}
	if ch, found := FindUnescapedSpecial(value); found {
		return "", fmt.Errorf("unescaped %q in %s value: use \\%c for literals", ch, annotationName, ch)
	}
	return UnescapeValue(value), nil
}

// ParseAnnotationBlock parses an annotation block using the schema
func ParseAnnotationBlock(lines []string, annotationName string, node *annotation.Def) (*ParsedAnnotation, error) {
	if node == nil {
		return nil, fmt.Errorf("unknown annotation: %s", annotationName)
	}

	result := &ParsedAnnotation{
		Name:             annotationName,
		Children:         make(map[string]*ParsedAnnotation),
		RepeatedChildren: make(map[string][]*ParsedAnnotation),
		Lines:            lines,
	}

	// Handle marker annotations
	if node.Kind == annotation.Marker {
		result.IsFlag = true
		return result, nil
	}

	// Handle flag annotations
	if node.Kind == annotation.Flag {
		result.IsFlag = true
		return result, nil
	}

	// Extract metadata if present
	if node.HasMetadata && len(lines) > 0 {
		result.Metadata = ExtractMetadata(lines[0], annotationName)
	}

	// Handle value annotations
	if node.Kind == annotation.Value {
		if len(lines) > 0 {
			// Extract value (everything after annotation name)
			value := strings.TrimSpace(strings.TrimPrefix(lines[0], annotationName))

			// Append continuation lines only for annotations that support multiline
			if node.SupportsMultiline {
				for i := 1; i < len(lines); i++ {
					continuation := strings.TrimSpace(lines[i])
					if continuation != "" {
						value += "\n" + continuation
					}
				}
			}

			resolved, err := resolveValue(value, node, annotationName)
			if err != nil {
				return nil, err
			}
			result.Value = resolved
		}
		return result, nil
	}

	// Handle reference annotations
	if node.Kind == annotation.Reference {
		if len(lines) > 0 {
			// Extract reference name(s) - can be comma-separated
			value := strings.TrimSpace(strings.TrimPrefix(lines[0], annotationName))
			resolved, err := resolveValue(value, node, annotationName)
			if err != nil {
				return nil, err
			}
			result.Value = resolved
		}
		return result, nil
	}

	// Handle block annotations and sub-commands
	if node.Kind == annotation.Block || node.Kind == annotation.SubCommand {
		// Extract content within braces
		content, err := ParseBracedBlock(lines, node)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", annotationName, err)
		}

		// Empty block is allowed if schema has no required children
		if len(content) == 0 {
			if !node.CanBeEmpty() {
				return nil, fmt.Errorf("%s cannot be empty (has required children)", annotationName)
			}
			return result, nil
		}

		// Route by source syntax: a single input line means the user wrote inline
		// format (opener and closer on the same line). Multiple input lines is a
		// multi-line block, regardless of whether the inner content collapses to
		// a single line.
		// A child's error is returned as it stands. It already names the
		// annotation it is about, and the caller prefixes the declaration, so
		// wrapping it here only repeats the annotation a third time.
		if len(lines) == 1 {
			if err := parseInlineChildren(content[0], node, result); err != nil {
				return nil, err
			}
		} else {
			if err := parseChildren(content, node, result); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// parseChildren parses child annotations within a block
func parseChildren(lines []string, parentNode *annotation.Def, result *ParsedAnnotation) error {
	if len(lines) == 0 {
		return nil
	}

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// Find annotation
		if !strings.HasPrefix(line, "@") {
			// Not an annotation line, might be continuation
			i++
			continue
		}

		// Extract annotation name
		annotationName := extractAnnotationName(line)
		if annotationName == "" {
			i++
			continue
		}

		// Get schema node for this annotation
		childNode := parentNode.GetChild(annotationName)
		if childNode == nil {
			return fmt.Errorf("unknown annotation %s in %s", annotationName, parentNode.Name)
		}

		// Collect all lines for this annotation
		annotationLines := []string{line}
		i++

		// How far this child reaches. The three cases are a block that spans
		// lines, a block that opened and closed on this one, and a value that
		// continues onto the next lines.
		blockPos := findBlockOpener(line)
		switch {
		case blockPos >= 0 && LastUnescaped(line, '}') <= blockPos:
			braceDepth := openerDepth(line, blockPos)

			for i < len(lines) && braceDepth > 0 {
				nextLine := lines[i]
				annotationLines = append(annotationLines, nextLine)

				// Same rule as ParseBracedBlock: a raw value's braces are text.
				if !startsWithRawValue(nextLine, childNode) {
					braceDepth += CountUnescapedBraces(nextLine)
				}
				i++
			}

		case blockPos >= 0:
			// Opened and closed on this line; there is nothing to collect.

		case childNode.SupportsMultiline:
			// No braces, collect continuation lines only for multiline annotations
			for i < len(lines) {
				nextLine := lines[i]
				if strings.TrimSpace(nextLine) == "" {
					i++
					continue
				}

				// A line starting with an unescaped @ ends the value, whether or
				// not this grammar knows the name. Continuing on an unknown one
				// would read a misspelled annotation as prose, and the only
				// complaint would be about the bare @ — advice whose fix, \@,
				// publishes the typo in the description. Ending here instead
				// sends the name to the unknown-annotation check below, which
				// says what is actually wrong. A description that really does
				// begin a line with @ writes it as \@.
				if StartsWithUnescapedAt(nextLine) {
					break
				}

				// This is a continuation line
				annotationLines = append(annotationLines, nextLine)
				i++
			}
		}

		// Parse this annotation
		parsed, err := ParseAnnotationBlock(annotationLines, annotationName, childNode)
		if err != nil {
			return err
		}

		// Store in result
		if childNode.Repeatable {
			result.RepeatedChildren[annotationName] = append(
				result.RepeatedChildren[annotationName],
				parsed,
			)
		} else {
			if _, exists := result.Children[annotationName]; exists {
				return fmt.Errorf("%s appears multiple times but is not repeatable", annotationName)
			}
			result.Children[annotationName] = parsed
		}
	}

	return nil
}

// parseInlineChildren parses the children of a block written on one line:
// "@description User email @format email @example test\@example.com".
//
// It is parseChildren's sibling: same job, different separator. A block written
// on one line separates its children by @ because it cannot separate them by
// newline. Which of the two runs is decided in ParseAnnotationBlock and nowhere
// else — there used to be a second entry point that decided it again, by a
// different rule, and the two disagreed.
func parseInlineChildren(content string, parentNode *annotation.Def, result *ParsedAnnotation) error {
	if content == "" {
		return nil
	}

	for i, part := range SplitOnUnescapedAt(content) {
		// The text before the first @ is empty for well-formed content.
		if i == 0 && strings.TrimSpace(part) == "" {
			continue
		}

		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Extract annotation name and value. No space means a flag.
		annotationName, value := "@"+part, ""
		if spaceIdx := strings.IndexAny(part, " \t"); spaceIdx >= 0 {
			annotationName = "@" + part[:spaceIdx]
			value = strings.TrimSpace(part[spaceIdx+1:])
		}

		childNode := parentNode.GetChild(annotationName)
		if childNode == nil {
			return fmt.Errorf("unknown annotation %s in %s", annotationName, parentNode.Name)
		}

		// A child that opens its own block cannot be written inline: its braces
		// would be indistinguishable from the parent's.
		if childNode.Kind == annotation.SubCommand && len(childNode.Children) > 0 {
			if ContainsUnescapedBrace(value) {
				return fmt.Errorf("sub-commands with sub-blocks cannot be inlined: %s", annotationName)
			}
		}

		resolvedValue, err := resolveValue(value, childNode, annotationName)
		if err != nil {
			return err
		}

		parsed := &ParsedAnnotation{
			Name:             annotationName,
			Value:            resolvedValue,
			IsFlag:           childNode.Kind == annotation.Flag,
			Children:         make(map[string]*ParsedAnnotation),
			RepeatedChildren: make(map[string][]*ParsedAnnotation),
		}
		if childNode.HasMetadata {
			parsed.Metadata = resolvedValue
		}

		if childNode.Repeatable {
			result.RepeatedChildren[annotationName] = append(
				result.RepeatedChildren[annotationName],
				parsed,
			)
			continue
		}
		if _, exists := result.Children[annotationName]; exists {
			return fmt.Errorf("%s appears multiple times but is not repeatable", annotationName)
		}
		result.Children[annotationName] = parsed
	}

	return nil
}

// extractAnnotationName extracts the annotation name from a line
// Example: "@field {" -> "@field"
// Example: "@title My API" -> "@title"
func extractAnnotationName(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@") {
		return ""
	}

	// Find the end of the annotation name (space, {, or end of line)
	for i, ch := range line {
		if ch == ' ' || ch == '{' || ch == '\t' {
			return line[:i]
		}
	}

	return line
}

// GetChildValue returns the value of a child annotation
func (pa *ParsedAnnotation) GetChildValue(name string) string {
	if child, ok := pa.Children[name]; ok {
		return child.Value
	}
	return ""
}

// GetRepeatedChildren returns all instances of a repeatable child
func (pa *ParsedAnnotation) GetRepeatedChildren(name string) []*ParsedAnnotation {
	return pa.RepeatedChildren[name]
}

// HasChild checks if a child annotation exists
func (pa *ParsedAnnotation) HasChild(name string) bool {
	_, ok := pa.Children[name]
	return ok
}
