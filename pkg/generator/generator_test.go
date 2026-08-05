package generator

import (
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/parser"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
)

// Feature-level output is locked by the golden files and the fixture harness
// in cmd/specgen. These tests cover the seams that neither reaches: version
// validation, the bare-ref emptiness check, enum tagging asymmetry, and the
// no-aliasing guarantee.

func renderYAML(t *testing.T, version string, pkg *resolver.Package) string {
	t.Helper()
	gen, err := NewGenerator(version)
	if err != nil {
		t.Fatalf("NewGenerator(%q): %v", version, err)
	}
	out, err := gen.Render(gen.Generate(pkg), FormatYAML)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return string(out)
}

func minimalPackage(schemas ...*resolver.Schema) *resolver.Package {
	return &resolver.Package{
		Name:    "test",
		API:     &parser.APIInfo{Title: "Test", Version: "1.0.0"},
		Schemas: schemas,
	}
}

func TestNewGenerator_RejectsUnknownVersion(t *testing.T) {
	for _, version := range []string{"", "2.0", "3.5", "3.0.3"} {
		if _, err := NewGenerator(version); err == nil {
			t.Errorf("NewGenerator(%q) should fail", version)
		}
	}
	for _, version := range []string{"3.0", "3.1", "3.2"} {
		if _, err := NewGenerator(version); err != nil {
			t.Errorf("NewGenerator(%q): %v", version, err)
		}
	}
}

func TestRefSchema_30_BareRefWithoutSiblings(t *testing.T) {
	// An unannotated, non-nullable ref must stay a bare $ref in 3.0; any
	// annotation must trigger the allOf wrapper instead.
	pkg := minimalPackage(
		&resolver.Schema{Name: "Owner", Fields: []*resolver.Field{
			{Name: "name", Required: true, Type: resolver.TypeInfo{OpenAPI: "string"}},
		}},
		&resolver.Schema{Name: "Item", Fields: []*resolver.Field{
			{Name: "bare", Required: true, Type: resolver.TypeInfo{Ref: "Owner"}},
			{Name: "documented", Description: "with docs", Required: true, Type: resolver.TypeInfo{Ref: "Owner"}},
		}},
	)

	out := renderYAML(t, "3.0", pkg)

	bare := `                bare:
                    $ref: '#/components/schemas/Owner'`
	if !strings.Contains(out, bare) {
		t.Errorf("bare field should emit a bare $ref, got:\n%s", out)
	}
	if !strings.Contains(out, "allOf:") {
		t.Errorf("documented field should emit an allOf wrapper, got:\n%s", out)
	}
}

func TestEnumTagging_Asymmetry(t *testing.T) {
	// Integer enums on a scalar field render unquoted (tagged !!int). Array
	// field enums land in items untagged; parameter array enums land in
	// items tagged by the item type.
	fieldScalar := &resolver.Field{
		Name: "level", Required: true,
		Type:        resolver.TypeInfo{OpenAPI: "integer"},
		Constraints: resolver.Constraints{Enum: []string{"1", "2"}},
	}
	fieldArray := &resolver.Field{
		Name: "levels", Required: true,
		Type:        resolver.TypeInfo{IsArray: true, Items: "integer"},
		Constraints: resolver.Constraints{Enum: []string{"1", "2"}},
	}
	pkg := minimalPackage(&resolver.Schema{Name: "S", Fields: []*resolver.Field{fieldScalar, fieldArray}})
	pkg.Endpoints = []*resolver.Endpoint{{
		Method: "GET", Path: "/x",
		Parameters: []*resolver.Param{{In: "query", Field: fieldArray}},
		Responses:  []*resolver.Response{{Status: "204", Description: "done"}},
	}}

	out := renderYAML(t, "3.1", pkg)

	if strings.Count(out, "- 1") != 3 {
		t.Errorf("expected all three enum emissions to render unquoted integers, got:\n%s", out)
	}
}

func TestScalarTagging_DefaultAndExample(t *testing.T) {
	// A default or example is annotation text. Emitted untagged, YAML would
	// retype it: "true" on a string field would come back a boolean.
	pkg := minimalPackage(&resolver.Schema{Name: "Flags", Fields: []*resolver.Field{
		{Name: "text", Required: true, Type: resolver.TypeInfo{OpenAPI: "string"},
			Constraints: resolver.Constraints{Default: "true", Example: "42"}},
		{Name: "count", Required: true, Type: resolver.TypeInfo{OpenAPI: "integer"},
			Constraints: resolver.Constraints{Default: "7"}},
		{Name: "on", Required: true, Type: resolver.TypeInfo{OpenAPI: "boolean"},
			Constraints: resolver.Constraints{Default: "true"}},
	}})

	out := renderYAML(t, "3.1", pkg)
	for _, want := range []string{`default: "true"`, `example: "42"`, "default: 7", "default: true"} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}
}

func TestGenerate_DoesNotAliasInput(t *testing.T) {
	field := &resolver.Field{
		Name: "id", Required: true,
		Type:        resolver.TypeInfo{OpenAPI: "string"},
		Constraints: resolver.Constraints{ReadOnly: true, UniqueItems: true},
	}
	pkg := minimalPackage(&resolver.Schema{Name: "S", Fields: []*resolver.Field{field}})

	gen, err := NewGenerator("3.1")
	if err != nil {
		t.Fatal(err)
	}
	doc := gen.Generate(pkg)

	// Mutating the input after generation must not change the document.
	field.Required = false
	field.ReadOnly = false
	field.UniqueItems = false

	out, err := gen.Render(doc, FormatYAML)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"readOnly: true", "uniqueItems: true", "required:"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output should contain %q independent of later input mutation, got:\n%s", want, out)
		}
	}
}

func TestRender_UnsupportedFormat(t *testing.T) {
	gen, err := NewGenerator("3.1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gen.Render(gen.Generate(minimalPackage()), OutputFormat("xml")); err == nil {
		t.Error("Render should fail for unsupported format")
	}
}
