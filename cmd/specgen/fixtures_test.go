package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/generator"
)

// TestFeatureFixtures is a mini-golden harness over testdata/ packages that
// lock features the examples/ golden files do not cover: readOnly/writeOnly,
// array constraints, exclusive bounds, deprecation, $ref emission variants,
// OpenAPI 3.0 output, JSON output, empty components, mixed block+inline
// responses, the primitive type matrix, and inline-body builder behavior.
//
// Expected files are updated with the same -update flag as TestGoldenFiles.
func TestFeatureFixtures(t *testing.T) {
	fixtures := []struct {
		name     string
		version  string
		format   generator.OutputFormat
		expected string
	}{
		{"readwrite", "3.1", generator.FormatYAML, "expected_31.yaml"},
		{"arrayconstraints", "3.1", generator.FormatYAML, "expected_31.yaml"},
		{"exclusivebounds", "3.1", generator.FormatYAML, "expected_31.yaml"},
		{"exclusivebounds", "3.0", generator.FormatYAML, "expected_30.yaml"},
		{"deprecated", "3.1", generator.FormatYAML, "expected_31.yaml"},
		{"refvariants", "3.1", generator.FormatYAML, "expected_31.yaml"},
		{"refvariants", "3.0", generator.FormatYAML, "expected_30.yaml"},
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

			expectedFile := filepath.Join(dir, fx.expected)

			if *update {
				if err := os.WriteFile(expectedFile, got, 0644); err != nil {
					t.Fatalf("update expected file: %v", err)
				}
				return
			}

			want, err := os.ReadFile(expectedFile)
			if err != nil {
				t.Fatalf("read expected file: %v", err)
			}

			if string(got) != string(want) {
				t.Errorf("output differs from %s", expectedFile)
				gotLines := splitLines(string(got))
				wantLines := splitLines(string(want))
				for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
					if gotLines[i] != wantLines[i] {
						t.Errorf("first diff at line %d:\n  got:  %s\n  want: %s", i+1, gotLines[i], wantLines[i])
						break
					}
				}
				if len(gotLines) != len(wantLines) {
					t.Errorf("line count differs: got %d, want %d", len(gotLines), len(wantLines))
				}
			}
		})
	}
}
