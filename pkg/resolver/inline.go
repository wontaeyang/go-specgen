package resolver

import (
	"fmt"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/parser"
	"github.com/wontaeyang/go-specgen/pkg/schema"
)

// InlineAnnotationSchema defines valid annotations for inline declarations.
// These are temporary and only exist during @endpoint resolution.
// Note: @body is intentionally excluded - the struct IS the body in inline context.
var InlineAnnotationSchema = map[string]*schema.SchemaNode{
	"@response": {
		Name:        "@response",
		Type:        schema.BlockAnnotation,
		HasMetadata: true, // status code
		Children: map[string]*schema.SchemaNode{
			"@contentType": {
				Name: "@contentType",
				Type: schema.ValueAnnotation,
			},
			"@description": {
				Name:              "@description",
				Type:              schema.ValueAnnotation,
				SupportsMultiline: true,
			},
			"@header": {
				Name:       "@header",
				Type:       schema.ValueAnnotation,
				Repeatable: true,
			},
			"@bind": {
				Name: "@bind",
				Type: schema.ValueAnnotation,
			},
		},
	},
	"@request": {
		Name: "@request",
		Type: schema.BlockAnnotation,
		Children: map[string]*schema.SchemaNode{
			"@contentType": {
				Name: "@contentType",
				Type: schema.ValueAnnotation,
			},
			"@description": {
				Name:              "@description",
				Type:              schema.ValueAnnotation,
				SupportsMultiline: true,
			},
			"@bind": {
				Name: "@bind",
				Type: schema.ValueAnnotation,
			},
		},
	},
}

func init() {
	// Initialize parent references for inline schema nodes
	for _, node := range InlineAnnotationSchema {
		node.InitializeParents()
	}
}

// ParseInlineAnnotation parses an inline @request/@response comment using InlineAnnotationSchema.
// Returns parsed annotation or error if invalid annotations are used.
func ParseInlineAnnotation(comment *parser.CommentBlock, annotationType string) (*parser.ParsedAnnotation, error) {
	if comment == nil {
		return nil, nil
	}

	schemaNode, ok := InlineAnnotationSchema["@"+annotationType]
	if !ok {
		return nil, fmt.Errorf("unknown inline annotation type: %s", annotationType)
	}

	// Extract the block content from comment lines
	lines := extractInlineBlockContent(comment.Lines, annotationType)
	if len(lines) == 0 {
		return nil, nil
	}

	return parser.ParseAnnotationBlock(lines, "@"+annotationType, schemaNode)
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
