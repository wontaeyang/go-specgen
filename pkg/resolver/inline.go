package resolver

import (
	"fmt"

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

	name := "@" + annotationType

	// Matched whole, not by prefix. The name is what decides which block this
	// is, so a prefix match let a longer name claim a shorter one's place --
	// @requestor opening a comment would have been read as the @request block.
	// ExtractAnnotationName draws the boundary the parser draws everywhere else.
	startIndex := -1
	for i, line := range lines {
		if parser.ExtractAnnotationName(line) == name {
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
