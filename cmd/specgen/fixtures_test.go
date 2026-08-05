package main

import (
	"path/filepath"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/generator"
)

// TestFeatureFixtures covers output the examples cannot reach. TestGoldenFiles
// renders every example at every OpenAPI version, so version differences are
// covered there; what remains here is JSON rendering plus edge cases that would
// read as noise in a documentation example: empty components, block/inline
// response merge order, the exhaustive primitive type matrix, and inline-body
// builder behavior.
//
// Annotation features belong in examples/, which is the coverage surface.
//
// Expected files are updated with the same -update flag as TestGoldenFiles.
func TestFeatureFixtures(t *testing.T) {
	fixtures := []struct {
		name     string
		version  string
		format   generator.OutputFormat
		expected string
	}{
		{"jsonoutput", "3.1", generator.FormatJSON, "expected_31.json"},
		{"emptycomponents", "3.1", generator.FormatYAML, "expected_31.yaml"},
		{"mixedresponses", "3.1", generator.FormatYAML, "expected_31.yaml"},
		{"primitivearrays", "3.1", generator.FormatYAML, "expected_31.yaml"},
		{"anyconstraints", "3.1", generator.FormatYAML, "expected_31.yaml"},
		{"inlineschemaref", "3.1", generator.FormatYAML, "expected_31.yaml"},
	}

	for _, fx := range fixtures {
		t.Run(fx.name+"_"+fx.expected, func(t *testing.T) {
			// The "./" prefix makes go/packages treat this as a directory
			// path, not an import path.
			dir := "./" + filepath.Join("testdata", fx.name)

			got, err := buildSpec(dir, fx.version, fx.format)
			if err != nil {
				t.Fatalf("buildSpec: %v", err)
			}

			compareGolden(t, filepath.Join(dir, fx.expected), got)
		})
	}
}
