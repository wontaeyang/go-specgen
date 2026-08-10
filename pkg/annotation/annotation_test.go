package annotation

import (
	"testing"
)

func TestAnnotationType_String(t *testing.T) {
	tests := []struct {
		name     string
		annType  Kind
		expected string
	}{
		{"Block", Block, "Block"},
		{"Value", Value, "Value"},
		{"Flag", Flag, "Flag"},
		{"Marker", Marker, "Marker"},
		{"Reference", Reference, "Reference"},
		{"SubCommand", SubCommand, "SubCommand"},
		{"Unknown", Kind(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.annType.String(); got != tt.expected {
				t.Errorf("Kind.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDef_GetChild(t *testing.T) {
	parent := &Def{
		Name: "parent",
		Children: map[string]*Def{
			"@child1": {Name: "@child1"},
			"@child2": {Name: "@child2"},
		},
	}

	tests := []struct {
		name      string
		node      *Def
		childName string
		wantNil   bool
	}{
		{"existing child", parent, "@child1", false},
		{"another existing child", parent, "@child2", false},
		{"non-existing child", parent, "@child3", true},
		{"nil children map", &Def{Name: "leaf"}, "@any", true},
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

func TestDef_HasChild(t *testing.T) {
	parent := &Def{
		Name: "parent",
		Children: map[string]*Def{
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

func TestDef_InitializeParents(t *testing.T) {
	root := &Def{
		Name: "root",
		Children: map[string]*Def{
			"@child": {
				Name: "@child",
				Children: map[string]*Def{
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

func TestDef_CanBeEmpty(t *testing.T) {
	tests := []struct {
		name     string
		node     *Def
		expected bool
	}{
		{
			name:     "no children",
			node:     &Def{Name: "@test"},
			expected: true,
		},
		{
			name: "all optional children",
			node: &Def{
				Name: "@test",
				Children: map[string]*Def{
					"@a": {Name: "@a"},
					"@b": {Name: "@b"},
				},
			},
			expected: true,
		},
		{
			name: "one required child",
			node: &Def{
				Name: "@test",
				Children: map[string]*Def{
					"@a": {Name: "@a", Required: true},
					"@b": {Name: "@b"},
				},
			},
			expected: false,
		},
		{
			name: "all required children",
			node: &Def{
				Name: "@test",
				Children: map[string]*Def{
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

func TestDef_Validate(t *testing.T) {
	tests := []struct {
		name    string
		node    *Def
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid block annotation with children",
			node: &Def{
				Name: "@test",
				Kind: Block,
				Children: map[string]*Def{
					"@child": {Name: "@child", Kind: Value},
				},
			},
			wantErr: false,
		},
		{
			name: "valid marker annotation",
			node: &Def{
				Name: "@test",
				Kind: Marker,
			},
			wantErr: false,
		},
		{
			// A block is the shape that nests. One declaring nothing to nest
			// should be a Marker, a Flag or a Value instead. This case used to
			// assert the opposite, because the check it exercises carried an
			// "&& !CanBeEmpty()" that made it unreachable -- a node with no
			// children has no required children either, so it could never fire.
			name: "invalid: block annotation without children",
			node: &Def{
				Name: "@test",
				Kind: Block,
			},
			wantErr: true,
			errMsg:  "block annotations must have children",
		},
		{
			name: "valid block annotation with optional children",
			node: &Def{
				Name: "@test",
				Kind: Block,
				Children: map[string]*Def{
					"@child": {Name: "@child", Kind: Value},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid: marker with children",
			node: &Def{
				Name: "@test",
				Kind: Marker,
				Children: map[string]*Def{
					"@child": {Name: "@child"},
				},
			},
			wantErr: true,
			errMsg:  "marker annotations cannot have children",
		},
		{
			name: "invalid: required and repeatable",
			node: &Def{
				Name:       "@test",
				Kind:       Value,
				Required:   true,
				Repeatable: true,
			},
			wantErr: true,
			errMsg:  "annotation cannot be both required and repeatable",
		},
		{
			name: "invalid: recursive child validation failure",
			node: &Def{
				Name: "@parent",
				Kind: Block,
				Children: map[string]*Def{
					"@child": {
						Name: "@child",
						Kind: Marker,
						Children: map[string]*Def{
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
			node: &Def{
				Name: "@test",
				Kind: Value,
			},
			wantErr: false,
		},
		{
			name: "valid flag annotation",
			node: &Def{
				Name: "@test",
				Kind: Flag,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Def.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if ve, ok := err.(*GrammarError); ok {
					if ve.Message != tt.errMsg {
						t.Errorf("GrammarError.Message = %q, want %q", ve.Message, tt.errMsg)
					}
				}
			}
		})
	}
}

// TestGrammarsAreValid runs the self-checks against the grammars they exist for.
//
// Validate and ValidateTargets describe themselves as run by the package's own
// tests, and until now nothing ran them on Schema or Declaration -- every case
// built a synthetic Def instead. A self-check nothing points at the real subject
// cannot catch the edit it was written for, which is an @annotation added to
// grammar.go in a shape the rest of the parser cannot read.
func TestGrammarsAreValid(t *testing.T) {
	grammars := []struct {
		name string
		root *Def
	}{
		{"Schema", Schema},
		{"Declaration", Declaration},
	}

	for _, g := range grammars {
		t.Run(g.name, func(t *testing.T) {
			if err := g.root.Validate(); err != nil {
				t.Errorf("%s.Validate() = %v, want nil", g.name, err)
			}

			// Only the doc-comment grammar's children are written on a
			// declaration; membership in Declaration is already the answer for
			// the in-function ones, so they carry no Target.
			if err := g.root.ValidateTargets(g.root == Schema); err != nil {
				t.Errorf("%s.ValidateTargets() = %v, want nil", g.name, err)
			}
		})
	}
}

func TestGrammarError_Error(t *testing.T) {
	err := &GrammarError{
		Node:    "@test",
		Message: "test error",
	}

	expected := "grammar error for @test: test error"
	if got := err.Error(); got != expected {
		t.Errorf("GrammarError.Error() = %v, want %v", got, expected)
	}
}
