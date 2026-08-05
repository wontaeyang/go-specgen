package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/generator"
	"github.com/wontaeyang/go-specgen/pkg/parser"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
	"github.com/wontaeyang/go-specgen/pkg/validator"
)

var update = flag.Bool("update", false, "update golden files")

func TestGoldenFiles(t *testing.T) {
	examples := []struct {
		name string
		yaml string
	}{
		{"block", "block.yaml"},
		{"closure", "closure.yaml"},
		{"customtypes", "customtypes.yaml"},
		{"enum", "enum.yaml"},
		{"overrides", "overrides.yaml"},
		{"generics", "generics.yaml"},
		{"inline", "inline.yaml"},
		{"nested", "nested.yaml"},
		{"parameters", "parameters.yaml"},
		{"petstore", "petstore.yaml"},
		{"responses", "responses.yaml"},
		{"security", "security.yaml"},
		{"standalone_api", "standalone_api.yaml"},
		{"tags", "tags.yaml"},
	}

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			dir := filepath.Join("..", "..", "examples", ex.name)

			// Step 1: Parse
			p := parser.NewParser(dir)
			parsed, err := p.Parse()
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			// Step 2: Resolve
			r, err := resolver.NewResolver(dir, p.Comments())
			if err != nil {
				t.Fatalf("resolver: %v", err)
			}

			resolved, err := r.Resolve(parsed)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}

			// Step 3: Validate
			v := validator.NewValidator()
			if err := v.Validate(resolved); err != nil {
				t.Fatalf("validate: %v", err)
			}

			// Step 4: Generate
			gen, err := generator.NewGenerator("3.1")
			if err != nil {
				t.Fatalf("generator: %v", err)
			}
			spec := gen.Generate(resolver.ConvertLegacy(resolved))

			got, err := gen.Render(spec, generator.FormatYAML)
			if err != nil {
				t.Fatalf("render: %v", err)
			}

			goldenFile := filepath.Join(dir, ex.yaml)

			// Update golden files if -update flag is set
			if *update {
				if err := os.WriteFile(goldenFile, got, 0644); err != nil {
					t.Fatalf("update golden file: %v", err)
				}
				return
			}

			// Compare against golden file
			want, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("read golden file: %v", err)
			}

			if string(got) != string(want) {
				t.Errorf("output differs from %s", goldenFile)
				// Show first differing line for easier debugging
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

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
