// Package annotation is the grammar: what every @annotation is called, what
// shape it takes in a comment, and what may appear inside it.
//
// It is data plus query methods and nothing else. Reading comments is
// pkg/parser's job, resolving Go types is pkg/resolver's, and emitting OpenAPI
// is pkg/generator's — none of that belongs here. Adding an annotation means
// editing grammar.go and nothing else.
package annotation

// Kind is the shape an annotation takes in a comment.
type Kind int

const (
	// Block wraps nested annotations in braces: @api { @title ... }
	Block Kind = iota

	// Value carries a single value: @title My API
	Value

	// Flag is present or absent, carrying nothing: @deprecated
	Flag

	// Marker tags the declaration below it: @schema
	Marker

	// Reference names another declaration: @query Filters
	Reference

	// SubCommand is a metadata-carrying entry inside a block: @with basicAuth
	SubCommand
)

// String returns the string representation of Kind
func (k Kind) String() string {
	switch k {
	case Block:
		return "Block"
	case Value:
		return "Value"
	case Flag:
		return "Flag"
	case Marker:
		return "Marker"
	case Reference:
		return "Reference"
	case SubCommand:
		return "SubCommand"
	default:
		return "Unknown"
	}
}

// Def is one annotation's entry in the grammar.
type Def struct {
	// Name is the annotation name (e.g., "@api", "@field")
	Name string

	// Kind is the shape this annotation takes
	Kind Kind

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

	// RawValue indicates the annotation's value is passed through verbatim:
	// no escape validation, no UnescapeValue. Used when the value is itself
	// a DSL with its own escape grammar (e.g. @pattern regex).
	RawValue bool

	// Children are nested annotations within this annotation
	Children map[string]*Def

	// Parent is a reference to the parent node (set by InitializeParents)
	Parent *Def
}

// GetChild returns a child node by name, or nil if not found
func (n *Def) GetChild(name string) *Def {
	if n.Children == nil {
		return nil
	}
	return n.Children[name]
}

// HasChild returns true if a child with the given name exists
func (n *Def) HasChild(name string) bool {
	return n.GetChild(name) != nil
}

// CanBeEmpty returns true if the block can be empty.
// A block can be empty if it has no required children.
func (n *Def) CanBeEmpty() bool {
	for _, child := range n.Children {
		if child.Required {
			return false
		}
	}
	return true
}

// InitializeParents recursively sets parent references in the grammar tree
func (n *Def) InitializeParents() {
	if n.Children == nil {
		return
	}

	for _, child := range n.Children {
		child.Parent = n
		child.InitializeParents()
	}
}

// Validate checks that the grammar tree is internally consistent. It says
// nothing about any particular comment — it is a self-check on the definitions
// in grammar.go, run by the package's own tests.
func (n *Def) Validate() error {
	// Required nodes cannot be repeatable (doesn't make sense)
	if n.Required && n.Repeatable {
		return &GrammarError{
			Node:    n.Name,
			Message: "annotation cannot be both required and repeatable",
		}
	}

	// Marker annotations should not have children
	if n.Kind == Marker && len(n.Children) > 0 {
		return &GrammarError{
			Node:    n.Name,
			Message: "marker annotations cannot have children",
		}
	}

	// Block annotations should have children (unless they can be empty)
	if n.Kind == Block && len(n.Children) == 0 && !n.CanBeEmpty() {
		return &GrammarError{
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

// GrammarError reports an inconsistency in the grammar definitions themselves.
type GrammarError struct {
	Node    string
	Message string
}

func (e *GrammarError) Error() string {
	return "grammar error for " + e.Node + ": " + e.Message
}
