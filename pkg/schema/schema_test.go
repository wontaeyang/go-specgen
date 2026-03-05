package schema

import (
	"testing"
)

func TestAnnotationType_String(t *testing.T) {
	tests := []struct {
		name     string
		annType  AnnotationType
		expected string
	}{
		{"BlockAnnotation", BlockAnnotation, "BlockAnnotation"},
		{"ValueAnnotation", ValueAnnotation, "ValueAnnotation"},
		{"FlagAnnotation", FlagAnnotation, "FlagAnnotation"},
		{"MarkerAnnotation", MarkerAnnotation, "MarkerAnnotation"},
		{"ReferenceAnnotation", ReferenceAnnotation, "ReferenceAnnotation"},
		{"SubCommand", SubCommand, "SubCommand"},
		{"Unknown", AnnotationType(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.annType.String(); got != tt.expected {
				t.Errorf("AnnotationType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSchemaNode_IsTopLevel(t *testing.T) {
	root := &SchemaNode{Name: "root"}
	child := &SchemaNode{Name: "child", Parent: root}
	grandchild := &SchemaNode{Name: "grandchild", Parent: child}

	tests := []struct {
		name     string
		node     *SchemaNode
		expected bool
	}{
		{"root is top level", root, true},
		{"child of root is top level", child, true},
		{"grandchild is not top level", grandchild, false},
		{"nil parent is top level", &SchemaNode{Name: "orphan"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.IsTopLevel(); got != tt.expected {
				t.Errorf("SchemaNode.IsTopLevel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSchemaNode_GetChild(t *testing.T) {
	parent := &SchemaNode{
		Name: "parent",
		Children: map[string]*SchemaNode{
			"@child1": {Name: "@child1"},
			"@child2": {Name: "@child2"},
		},
	}

	tests := []struct {
		name      string
		node      *SchemaNode
		childName string
		wantNil   bool
	}{
		{"existing child", parent, "@child1", false},
		{"another existing child", parent, "@child2", false},
		{"non-existing child", parent, "@child3", true},
		{"nil children map", &SchemaNode{Name: "leaf"}, "@any", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.node.GetChild(tt.childName)
			if tt.wantNil && got != nil {
				t.Errorf("GetChild(%s) = %v, want nil", tt.childName, got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("GetChild(%s) = nil, want non-nil", tt.childName)
			}
		})
	}
}

func TestSchemaNode_HasChild(t *testing.T) {
	parent := &SchemaNode{
		Name: "parent",
		Children: map[string]*SchemaNode{
			"@child": {Name: "@child"},
		},
	}

	if !parent.HasChild("@child") {
		t.Error("HasChild(@child) = false, want true")
	}

	if parent.HasChild("@nonexistent") {
		t.Error("HasChild(@nonexistent) = true, want false")
	}
}

func TestSchemaNode_IsSibling(t *testing.T) {
	parent := &SchemaNode{
		Name: "parent",
		Children: map[string]*SchemaNode{
			"@sibling1": {Name: "@sibling1"},
			"@sibling2": {Name: "@sibling2"},
		},
	}

	// Set up parent references
	parent.Children["@sibling1"].Parent = parent
	parent.Children["@sibling2"].Parent = parent

	sibling1 := parent.Children["@sibling1"]

	if !sibling1.IsSibling("@sibling2") {
		t.Error("IsSibling(@sibling2) = false, want true")
	}

	if sibling1.IsSibling("@nonexistent") {
		t.Error("IsSibling(@nonexistent) = true, want false")
	}

	orphan := &SchemaNode{Name: "orphan"}
	if orphan.IsSibling("@anything") {
		t.Error("orphan.IsSibling(@anything) = true, want false")
	}
}

func TestSchemaNode_GetSiblings(t *testing.T) {
	parent := &SchemaNode{
		Name: "parent",
		Children: map[string]*SchemaNode{
			"@child1": {Name: "@child1"},
			"@child2": {Name: "@child2"},
			"@child3": {Name: "@child3"},
		},
	}

	child1 := parent.Children["@child1"]
	child1.Parent = parent

	siblings := child1.GetSiblings()

	// Should return 2 siblings (child2 and child3), not including child1 itself
	if len(siblings) != 2 {
		t.Errorf("GetSiblings() returned %d siblings, want 2", len(siblings))
	}

	// Check that child1 is not in its own siblings
	for _, sib := range siblings {
		if sib == "@child1" {
			t.Error("GetSiblings() includes self, should only return other siblings")
		}
	}

	// Orphan node should return nil
	orphan := &SchemaNode{Name: "orphan"}
	if siblings := orphan.GetSiblings(); siblings != nil {
		t.Error("orphan.GetSiblings() != nil, want nil")
	}
}

func TestSchemaNode_InitializeParents(t *testing.T) {
	root := &SchemaNode{
		Name: "root",
		Children: map[string]*SchemaNode{
			"@child": {
				Name: "@child",
				Children: map[string]*SchemaNode{
					"@grandchild": {Name: "@grandchild"},
				},
			},
		},
	}

	root.InitializeParents()

	child := root.Children["@child"]
	if child.Parent != root {
		t.Error("child.Parent != root after InitializeParents")
	}

	grandchild := child.Children["@grandchild"]
	if grandchild.Parent != child {
		t.Error("grandchild.Parent != child after InitializeParents")
	}
}

func TestSchemaNode_CanBeEmpty(t *testing.T) {
	tests := []struct {
		name     string
		node     *SchemaNode
		expected bool
	}{
		{
			name:     "no children",
			node:     &SchemaNode{Name: "@test"},
			expected: true,
		},
		{
			name: "all optional children",
			node: &SchemaNode{
				Name: "@test",
				Children: map[string]*SchemaNode{
					"@a": {Name: "@a"},
					"@b": {Name: "@b"},
				},
			},
			expected: true,
		},
		{
			name: "one required child",
			node: &SchemaNode{
				Name: "@test",
				Children: map[string]*SchemaNode{
					"@a": {Name: "@a", Required: true},
					"@b": {Name: "@b"},
				},
			},
			expected: false,
		},
		{
			name: "all required children",
			node: &SchemaNode{
				Name: "@test",
				Children: map[string]*SchemaNode{
					"@a": {Name: "@a", Required: true},
					"@b": {Name: "@b", Required: true},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.CanBeEmpty(); got != tt.expected {
				t.Errorf("CanBeEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSchemaNode_Validate(t *testing.T) {
	tests := []struct {
		name    string
		node    *SchemaNode
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid block annotation with children",
			node: &SchemaNode{
				Name: "@test",
				Type: BlockAnnotation,
				Children: map[string]*SchemaNode{
					"@child": {Name: "@child", Type: ValueAnnotation},
				},
			},
			wantErr: false,
		},
		{
			name: "valid marker annotation",
			node: &SchemaNode{
				Name: "@test",
				Type: MarkerAnnotation,
			},
			wantErr: false,
		},
		{
			name: "valid block annotation without children (can be empty)",
			node: &SchemaNode{
				Name: "@test",
				Type: BlockAnnotation,
				// No children, so CanBeEmpty() returns true
			},
			wantErr: false,
		},
		{
			name: "valid block annotation with optional children",
			node: &SchemaNode{
				Name: "@test",
				Type: BlockAnnotation,
				Children: map[string]*SchemaNode{
					"@child": {Name: "@child", Type: ValueAnnotation},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid: marker with children",
			node: &SchemaNode{
				Name: "@test",
				Type: MarkerAnnotation,
				Children: map[string]*SchemaNode{
					"@child": {Name: "@child"},
				},
			},
			wantErr: true,
			errMsg:  "marker annotations cannot have children",
		},
		{
			name: "invalid: required and repeatable",
			node: &SchemaNode{
				Name:       "@test",
				Type:       ValueAnnotation,
				Required:   true,
				Repeatable: true,
			},
			wantErr: true,
			errMsg:  "annotation cannot be both required and repeatable",
		},
		{
			name: "invalid: recursive child validation failure",
			node: &SchemaNode{
				Name: "@parent",
				Type: BlockAnnotation,
				Children: map[string]*SchemaNode{
					"@child": {
						Name: "@child",
						Type: MarkerAnnotation,
						Children: map[string]*SchemaNode{
							"@invalid": {Name: "@invalid"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "marker annotations cannot have children",
		},
		{
			name: "valid value annotation",
			node: &SchemaNode{
				Name: "@test",
				Type: ValueAnnotation,
			},
			wantErr: false,
		},
		{
			name: "valid flag annotation",
			node: &SchemaNode{
				Name: "@test",
				Type: FlagAnnotation,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SchemaNode.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if ve, ok := err.(*ValidationError); ok {
					if ve.Message != tt.errMsg {
						t.Errorf("ValidationError.Message = %q, want %q", ve.Message, tt.errMsg)
					}
				}
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Node:    "@test",
		Message: "test error",
	}

	expected := "schema validation error for @test: test error"
	if got := err.Error(); got != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", got, expected)
	}
}
