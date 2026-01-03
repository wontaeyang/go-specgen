package schema

import (
	"fmt"
	"strings"
)

// Validator validates parsed data against the annotation schema
type Validator struct {
	schema *SchemaNode
}

// NewValidator creates a new validator with the given schema
func NewValidator(schema *SchemaNode) *Validator {
	return &Validator{schema: schema}
}

// ValidateAnnotation validates a parsed annotation against its schema
func (v *Validator) ValidateAnnotation(annotationName string, data map[string]interface{}) error {
	node := v.schema.GetChild(annotationName)
	if node == nil {
		return fmt.Errorf("unknown annotation: %s", annotationName)
	}

	return v.validateNode(node, data)
}

// validateNode recursively validates data against a schema node
func (v *Validator) validateNode(node *SchemaNode, data map[string]interface{}) error {
	if data == nil && node.Required {
		return fmt.Errorf("%s is required", node.Name)
	}

	if data == nil {
		return nil
	}

	// Validate required children
	for childName, childNode := range node.Children {
		if childNode.Required {
			if _, exists := data[childName]; !exists {
				return fmt.Errorf("%s requires %s", node.Name, childName)
			}
		}
	}

	// Validate each field in data has a corresponding schema node
	for key, value := range data {
		childNode := node.GetChild(key)
		if childNode == nil {
			return fmt.Errorf("unknown field %s in %s", key, node.Name)
		}

		// Recursively validate children
		if childMap, ok := value.(map[string]interface{}); ok {
			if err := v.validateNode(childNode, childMap); err != nil {
				return err
			}
		}

		// Validate using custom validator if present
		if childNode.Validator != nil {
			if strValue, ok := value.(string); ok {
				if err := childNode.Validator(strValue); err != nil {
					return fmt.Errorf("%s validation failed: %w", key, err)
				}
			}
		}
	}

	return nil
}

// ValidateInlineFormat checks if an annotation can use inline format.
// All block annotations support inline format, but nested blocks are not allowed.
func (v *Validator) ValidateInlineFormat(annotationName string) error {
	node := v.schema.GetChild(annotationName)
	if node == nil {
		return fmt.Errorf("unknown annotation: %s", annotationName)
	}

	// All block annotations support inline format
	// The only restriction is that nested blocks cannot be inlined,
	// which is validated by ValidateNoNestedBraces
	return nil
}

// ValidateNoNestedBraces ensures inline content doesn't have nested braces.
// Escaped braces (\{ and \}) are ignored and don't count as nesting.
func ValidateNoNestedBraces(content string) error {
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

// GetTopLevelAnnotations returns all top-level annotation names
func GetTopLevelAnnotations() []string {
	annotations := make([]string, 0)
	for name, node := range AnnotationSchema.Children {
		if node.IsTopLevel() {
			annotations = append(annotations, name)
		}
	}
	return annotations
}

// IsTopLevelAnnotation checks if an annotation is top-level
func IsTopLevelAnnotation(name string) bool {
	node := AnnotationSchema.GetChild(name)
	return node != nil && node.IsTopLevel()
}

// GetAnnotationNode returns the schema node for an annotation
func GetAnnotationNode(name string) *SchemaNode {
	return AnnotationSchema.GetChild(name)
}

// IsMarkerAnnotation checks if an annotation is a marker type
func IsMarkerAnnotation(name string) bool {
	node := GetAnnotationNode(name)
	return node != nil && node.Type == MarkerAnnotation
}

// IsBlockAnnotation checks if an annotation is a block type
func IsBlockAnnotation(name string) bool {
	node := GetAnnotationNode(name)
	return node != nil && node.Type == BlockAnnotation
}

// IsValueAnnotation checks if an annotation is a value type
func IsValueAnnotation(name string) bool {
	node := GetAnnotationNode(name)
	return node != nil && node.Type == ValueAnnotation
}

// IsFlagAnnotation checks if an annotation is a flag type
func IsFlagAnnotation(name string) bool {
	node := GetAnnotationNode(name)
	return node != nil && node.Type == FlagAnnotation
}

// IsReferenceAnnotation checks if an annotation is a reference type
func IsReferenceAnnotation(name string) bool {
	node := GetAnnotationNode(name)
	return node != nil && node.Type == ReferenceAnnotation
}

// HasMetadata checks if an annotation has metadata on its opening line
func HasMetadata(name string) bool {
	node := GetAnnotationNode(name)
	return node != nil && node.HasMetadata
}

// IsRepeatable checks if an annotation can appear multiple times
func IsRepeatable(name string) bool {
	node := GetAnnotationNode(name)
	return node != nil && node.Repeatable
}

// AllowsEmpty checks if an annotation can exist without content.
// An annotation can be empty if it has no required children.
func AllowsEmpty(name string) bool {
	node := GetAnnotationNode(name)
	return node != nil && node.CanBeEmpty()
}

// SupportsInline checks if an annotation supports inline format.
// All block annotations support inline format (nested blocks cannot be inlined).
func SupportsInline(name string) bool {
	node := GetAnnotationNode(name)
	return node != nil && (node.Type == BlockAnnotation || node.Type == SubCommand)
}

// ValidateSchemaIntegrity validates the entire annotation schema
func ValidateSchemaIntegrity() error {
	return AnnotationSchema.Validate()
}

// GetChildrenNames returns the names of all children of an annotation
func GetChildrenNames(annotationName string) []string {
	node := GetAnnotationNode(annotationName)
	if node == nil || node.Children == nil {
		return nil
	}

	names := make([]string, 0, len(node.Children))
	for name := range node.Children {
		names = append(names, name)
	}
	return names
}

// IsSiblingAnnotation checks if two annotations are siblings
func IsSiblingAnnotation(name1, name2 string) bool {
	node1 := GetAnnotationNode(name1)
	if node1 == nil {
		return false
	}
	return node1.IsSibling(name2)
}

// FormatAnnotationPath returns a formatted path for error messages
func FormatAnnotationPath(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return strings.Join(path, " > ")
}
