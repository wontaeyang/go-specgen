package schema

// AnnotationType defines the type of annotation
type AnnotationType int

const (
	// BlockAnnotation has a { } block with nested content
	BlockAnnotation AnnotationType = iota

	// ValueAnnotation has a simple value (e.g., @title My API)
	ValueAnnotation

	// FlagAnnotation is a marker with no value (e.g., @deprecated)
	FlagAnnotation

	// MarkerAnnotation marks a struct for inclusion (e.g., @schema)
	MarkerAnnotation

	// ReferenceAnnotation references another struct (e.g., @schema User)
	ReferenceAnnotation

	// SubCommand is a sub-command within a block (e.g., @with)
	SubCommand
)

// String returns the string representation of AnnotationType
func (a AnnotationType) String() string {
	switch a {
	case BlockAnnotation:
		return "BlockAnnotation"
	case ValueAnnotation:
		return "ValueAnnotation"
	case FlagAnnotation:
		return "FlagAnnotation"
	case MarkerAnnotation:
		return "MarkerAnnotation"
	case ReferenceAnnotation:
		return "ReferenceAnnotation"
	case SubCommand:
		return "SubCommand"
	default:
		return "Unknown"
	}
}

// SchemaNode defines a node in the annotation schema tree
type SchemaNode struct {
	// Name is the annotation name (e.g., "@api", "@field")
	Name string

	// Type is the annotation type
	Type AnnotationType

	// Required indicates if this annotation must be present
	Required bool

	// HasMetadata indicates if the opening line contains metadata
	// Example: @endpoint GET /users - "GET /users" is metadata
	// Example: @server https://api.com - "https://api.com" is metadata
	HasMetadata bool

	// Repeatable indicates if this annotation can appear multiple times
	// Example: @response can repeat with different status codes
	Repeatable bool

	// SupportsMultiline indicates if this annotation supports multi-line values
	// Only @description annotations should have this set to true
	SupportsMultiline bool

	// Children are nested annotations within this annotation
	Children map[string]*SchemaNode

	// Validator is an optional custom validation function
	Validator func(value string) error

	// Parent is a reference to the parent node (set during schema initialization)
	Parent *SchemaNode
}

// IsTopLevel returns true if this is a top-level annotation
func (n *SchemaNode) IsTopLevel() bool {
	return n.Parent == nil || n.Parent.Name == "root"
}

// GetChild returns a child node by name, or nil if not found
func (n *SchemaNode) GetChild(name string) *SchemaNode {
	if n.Children == nil {
		return nil
	}
	return n.Children[name]
}

// HasChild returns true if a child with the given name exists
func (n *SchemaNode) HasChild(name string) bool {
	return n.GetChild(name) != nil
}

// IsSibling checks if a given annotation name is a sibling of this node
func (n *SchemaNode) IsSibling(name string) bool {
	if n.Parent == nil {
		return false
	}
	return n.Parent.HasChild(name)
}

// GetSiblings returns all sibling annotation names
func (n *SchemaNode) GetSiblings() []string {
	if n.Parent == nil || n.Parent.Children == nil {
		return nil
	}

	siblings := make([]string, 0, len(n.Parent.Children))
	for name := range n.Parent.Children {
		if name != n.Name {
			siblings = append(siblings, name)
		}
	}
	return siblings
}

// CanBeEmpty returns true if the block can be empty.
// A block can be empty if it has no required children.
func (n *SchemaNode) CanBeEmpty() bool {
	for _, child := range n.Children {
		if child.Required {
			return false
		}
	}
	return true
}

// InitializeParents recursively sets parent references in the schema tree
func (n *SchemaNode) InitializeParents() {
	if n.Children == nil {
		return
	}

	for _, child := range n.Children {
		child.Parent = n
		child.InitializeParents()
	}
}

// Validate validates the schema node structure
func (n *SchemaNode) Validate() error {
	// Required nodes cannot be repeatable (doesn't make sense)
	if n.Required && n.Repeatable {
		return &ValidationError{
			Node:    n.Name,
			Message: "annotation cannot be both required and repeatable",
		}
	}

	// Marker annotations should not have children
	if n.Type == MarkerAnnotation && len(n.Children) > 0 {
		return &ValidationError{
			Node:    n.Name,
			Message: "marker annotations cannot have children",
		}
	}

	// Block annotations should have children (unless they can be empty)
	if n.Type == BlockAnnotation && len(n.Children) == 0 && !n.CanBeEmpty() {
		return &ValidationError{
			Node:    n.Name,
			Message: "block annotations must have children",
		}
	}

	// Validate children recursively
	for _, child := range n.Children {
		if err := child.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ValidationError represents a schema validation error
type ValidationError struct {
	Node    string
	Message string
}

func (e *ValidationError) Error() string {
	return "schema validation error for " + e.Node + ": " + e.Message
}
