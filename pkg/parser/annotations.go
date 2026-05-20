package parser

import (
	"fmt"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/schema"
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
	for i := 0; i < len(line)-1; i++ {
		// Skip escape sequences
		if line[i] == '\\' && i+1 < len(line) {
			next := line[i+1]
			if next == '{' || next == '}' || next == '@' || next == '\\' {
				i++ // Skip the escaped character
				continue
			}
		}
		// Look for space/tab followed by {
		if (line[i] == ' ' || line[i] == '\t') && line[i+1] == '{' {
			return i + 1
		}
	}
	return -1
}

// countBracesFromPosition counts unescaped braces starting from a given position.
// Returns the final depth.
func countBracesFromPosition(line string, startPos int) int {
	depth := 0
	i := startPos
	for i < len(line) {
		if line[i] == '\\' && i+1 < len(line) {
			next := line[i+1]
			if next == '{' || next == '}' || next == '@' || next == '\\' {
				i += 2
				continue
			}
		}
		if line[i] == '{' {
			depth++
		} else if line[i] == '}' {
			depth--
		}
		i++
	}
	return depth
}

// ParseBracedBlock extracts content within braces { }
// Returns the content lines and any error.
// Uses position-based detection: block delimiter is " {" (space + brace).
// This allows path parameters like {id} to coexist with block delimiters.
func ParseBracedBlock(lines []string) ([]string, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	// Find the line with block opener (" {" pattern)
	openLineIndex := -1
	openBracePos := -1
	for i, line := range lines {
		pos := findBlockOpener(line)
		if pos >= 0 {
			openLineIndex = i
			openBracePos = pos
			break
		}
	}

	// No block delimiter found - might be a marker annotation or flag
	if openLineIndex == -1 {
		return nil, nil
	}

	// Extract content between braces
	var content []string
	braceDepth := 0

	for i := openLineIndex; i < len(lines); i++ {
		line := lines[i]
		originalLine := line

		// For the first line, only count braces from the block opener position
		if i == openLineIndex {
			braceDepth += countBracesFromPosition(line, openBracePos)
			// Take everything after the opening brace
			line = line[openBracePos+1:]
		} else {
			// For subsequent lines, count all braces
			lineDepth, _ := CountUnescapedBraces(line)
			braceDepth += lineDepth
		}

		line = strings.TrimSpace(line)

		// Check if we've closed all braces
		if braceDepth == 0 {
			// For inline blocks like "{ @desc foo }", extract content before closing brace
			if i == openLineIndex && line != "" {
				// Remove the trailing } if present
				line = strings.TrimSuffix(line, "}")
				line = strings.TrimSpace(line)
				if line != "" {
					content = append(content, line)
				}
			}
			return content, nil
		}

		// Add non-empty lines to content
		if line != "" {
			content = append(content, line)
		}

		// Safety check
		if braceDepth < 0 {
			return nil, fmt.Errorf("unbalanced braces at line: %s", originalLine)
		}
	}

	// If we get here, braces are unbalanced
	if braceDepth != 0 {
		return nil, fmt.Errorf("unbalanced braces: depth=%d", braceDepth)
	}

	return content, nil
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
func resolveValue(value string, node *schema.SchemaNode, annotationName string) (string, error) {
	if node.RawValue {
		return value, nil
	}
	if ch, found := FindUnescapedSpecial(value); found {
		return "", fmt.Errorf("unescaped %q in %s value: use \\%c for literals", ch, annotationName, ch)
	}
	return UnescapeValue(value), nil
}

// ParseAnnotationBlock parses an annotation block using the schema
func ParseAnnotationBlock(lines []string, annotationName string, node *schema.SchemaNode) (*ParsedAnnotation, error) {
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
	if node.Type == schema.MarkerAnnotation {
		result.IsFlag = true
		return result, nil
	}

	// Handle flag annotations
	if node.Type == schema.FlagAnnotation {
		result.IsFlag = true
		return result, nil
	}

	// Extract metadata if present
	if node.HasMetadata && len(lines) > 0 {
		result.Metadata = ExtractMetadata(lines[0], annotationName)
	}

	// Handle value annotations
	if node.Type == schema.ValueAnnotation {
		if len(lines) > 0 {
			// Extract value (everything after annotation name)
			firstLine := lines[0]
			value := strings.TrimPrefix(firstLine, annotationName)
			value = strings.TrimSpace(value)

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
	if node.Type == schema.ReferenceAnnotation {
		if len(lines) > 0 {
			// Extract reference name(s) - can be comma-separated
			firstLine := lines[0]
			value := strings.TrimPrefix(firstLine, annotationName)
			value = strings.TrimSpace(value)
			resolved, err := resolveValue(value, node, annotationName)
			if err != nil {
				return nil, err
			}
			result.Value = resolved
		}
		return result, nil
	}

	// Handle block annotations and sub-commands
	if node.Type == schema.BlockAnnotation || node.Type == schema.SubCommand {
		// Extract content within braces
		content, err := ParseBracedBlock(lines)
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
		if len(lines) == 1 {
			if err := parseInlineChildren(content[0], node, result); err != nil {
				return nil, fmt.Errorf("failed to parse %s children: %w", annotationName, err)
			}
		} else {
			if err := parseChildren(content, node, result); err != nil {
				return nil, fmt.Errorf("failed to parse %s children: %w", annotationName, err)
			}
		}
	}

	return result, nil
}

// parseChildren parses child annotations within a block
func parseChildren(lines []string, parentNode *schema.SchemaNode, result *ParsedAnnotation) error {
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

		// Check if this annotation has a block delimiter (" {" pattern)
		blockPos := findBlockOpener(line)
		if blockPos >= 0 {
			// Count braces starting from block position
			braceDepth := countBracesFromPosition(line, blockPos)

			for i < len(lines) && braceDepth > 0 {
				nextLine := lines[i]
				annotationLines = append(annotationLines, nextLine)

				lineDepth, _ := CountUnescapedBraces(nextLine)
				braceDepth += lineDepth
				i++
			}
		} else if childNode.SupportsMultiline {
			// No braces, collect continuation lines only for multiline annotations
			for i < len(lines) {
				nextLine := lines[i]
				if strings.TrimSpace(nextLine) == "" {
					i++
					continue
				}

				// Check if this is a sibling annotation (unescaped @ at start)
				if StartsWithUnescapedAt(nextLine) {
					nextAnnotationName := extractAnnotationName(nextLine)
					if parentNode.HasChild(nextAnnotationName) {
						// This is a sibling, stop collecting
						break
					}
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
