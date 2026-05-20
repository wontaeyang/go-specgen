package parser

import (
	"fmt"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/schema"
)

// IsInlineFormat checks if an annotation uses inline format
// Inline format: @field { @description Test @format email }
// Multi-line format: @field {\n  @description Test\n  @format email\n}
func IsInlineFormat(lines []string) bool {
	if len(lines) == 0 {
		return false
	}

	// Check if all content is on a single line or first line
	firstLine := lines[0]

	// Must have opening brace on first line
	if !strings.Contains(firstLine, "{") {
		return false
	}

	// Must have closing brace on first line for inline
	if !strings.Contains(firstLine, "}") {
		return false
	}

	// Check that opening brace comes before closing brace
	openIdx := strings.Index(firstLine, "{")
	closeIdx := strings.Index(firstLine, "}")
	return openIdx < closeIdx
}

// ParseInlineAnnotation parses an inline annotation
// Example: @field { @description User email @format email }
// Note: All block annotations support inline format, but nested blocks are not allowed.
// The nested-brace check is skipped when the annotation has no block-producing children
// (e.g. @field's children are all value/flag annotations), so values like a regex
// quantifier `{64}` are not misread as a nested block.
func ParseInlineAnnotation(line, annotationName string, node *schema.SchemaNode) (*ParsedAnnotation, error) {
	if node.HasBlockChildren() {
		if err := validateNoNestedBraces(line); err != nil {
			return nil, err
		}
	}

	result := &ParsedAnnotation{
		Name:             annotationName,
		Children:         make(map[string]*ParsedAnnotation),
		RepeatedChildren: make(map[string][]*ParsedAnnotation),
		Lines:            []string{line},
	}

	// Extract content between braces
	openIdx := strings.Index(line, "{")
	closeIdx := strings.LastIndex(line, "}")

	if openIdx == -1 || closeIdx == -1 || openIdx >= closeIdx {
		return nil, fmt.Errorf("invalid inline format for %s", annotationName)
	}

	content := line[openIdx+1 : closeIdx]
	content = strings.TrimSpace(content)

	// Empty content is allowed if schema has no required children
	if content == "" {
		if !node.CanBeEmpty() {
			return nil, fmt.Errorf("%s cannot be empty (has required children)", annotationName)
		}
		return result, nil
	}

	// Parse inline children
	if err := parseInlineChildren(content, node, result); err != nil {
		return nil, err
	}

	return result, nil
}

// parseInlineChildren parses children in inline format
// Example: "@description User email @format email @example test@example.com"
// Supports escape sequences: \{, \}, \@, \\
func parseInlineChildren(content string, parentNode *schema.SchemaNode, result *ParsedAnnotation) error {
	if content == "" {
		return nil
	}

	// Protect escaped @ before splitting by @
	protected := ProtectEscapedAt(content)
	parts := strings.Split(protected, "@")

	for i, part := range parts {
		// Skip empty parts (first part before first @)
		if i == 0 && strings.TrimSpace(part) == "" {
			continue
		}

		// Restore escaped @ in this part
		part = RestoreEscapedAt(part)
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Extract annotation name and value
		spaceIdx := strings.IndexAny(part, " \t")
		var annotationName, value string

		if spaceIdx == -1 {
			// No space, just annotation name (flag annotation)
			annotationName = "@" + part
			value = ""
		} else {
			annotationName = "@" + part[:spaceIdx]
			value = strings.TrimSpace(part[spaceIdx+1:])
		}

		// Get schema node
		childNode := parentNode.GetChild(annotationName)
		if childNode == nil {
			return fmt.Errorf("unknown annotation %s in %s", annotationName, parentNode.Name)
		}

		// Check if sub-command has sub-blocks (not allowed in inline)
		if childNode.Type == schema.SubCommand && len(childNode.Children) > 0 {
			// Check if value contains unescaped braces (sub-block)
			if ContainsUnescapedBrace(value) {
				return fmt.Errorf("sub-commands with sub-blocks cannot be inlined: %s", annotationName)
			}
		}

		// Create parsed annotation with unescaped value
		unescapedValue := UnescapeValue(value)
		parsed := &ParsedAnnotation{
			Name:             annotationName,
			Value:            unescapedValue,
			IsFlag:           childNode.Type == schema.FlagAnnotation,
			Children:         make(map[string]*ParsedAnnotation),
			RepeatedChildren: make(map[string][]*ParsedAnnotation),
		}

		// For annotations with metadata (like @body), also set Metadata
		if childNode.HasMetadata {
			parsed.Metadata = unescapedValue
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

// validateNoNestedBraces ensures inline content doesn't have nested braces.
// Escaped braces (\{ and \}) are ignored and don't count as nesting.
func validateNoNestedBraces(content string) error {
	depth := 0
	i := 0
	for i < len(content) {
		// Skip escape sequences: \{, \}, \@, \\
		if i+1 < len(content) && content[i] == '\\' {
			next := content[i+1]
			if next == '{' || next == '}' || next == '@' || next == '\\' {
				i += 2
				continue
			}
		}
		if content[i] == '{' {
			depth++
			if depth > 1 {
				return fmt.Errorf("nested blocks cannot be inlined, use multi-line format")
			}
		} else if content[i] == '}' {
			depth--
		}
		i++
	}

	if depth != 0 {
		return fmt.Errorf("unbalanced braces in inline content")
	}

	return nil
}
