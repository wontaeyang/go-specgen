package resolver

import (
	"fmt"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/annotation"
	"github.com/wontaeyang/go-specgen/pkg/parser"
)

// ParseInlineDeclaration parses an inline @request/@response comment against
// the in-function grammar.
// Returns parsed annotation or error if invalid annotations are used.
func ParseInlineDeclaration(comment *parser.CommentBlock, annotationType string) (*parser.ParsedAnnotation, error) {
	if comment == nil {
		return nil, nil
	}

	def := annotation.Declaration.GetChild("@" + annotationType)
	if def == nil {
		return nil, fmt.Errorf("unknown inline annotation type: %s", annotationType)
	}

	// Extract the block content from comment lines
	lines := extractInlineBlockContent(comment.Lines, annotationType)
	if len(lines) == 0 {
		return nil, nil
	}

	return parser.ParseAnnotationBlock(lines, "@"+annotationType, def)
}

// extractInlineBlockContent extracts the annotation block content from comment lines.
// It finds the line starting with the annotation and extracts the block content.
func extractInlineBlockContent(lines []string, annotationType string) []string {
	if len(lines) == 0 {
		return nil
	}

	prefix := "@" + annotationType

	// Find the line with the annotation
	startIndex := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			startIndex = i
			break
		}
	}

	if startIndex == -1 {
		return nil
	}

	// Collect all lines from the annotation start
	// The parser.ParseAnnotationBlock will handle brace matching
	return lines[startIndex:]
}
