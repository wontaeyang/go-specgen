package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/generator"
)

var update = flag.Bool("update", false, "update golden files")

// examples are the packages under examples/. Each one is rendered at every
// version that has its own emission behavior, so version-specific emission
// (nullability, exclusive bounds, $ref siblings) is covered by the same
// sources that document the annotations.
var examples = []string{
	"block",
	"closure",
	"constraints",
	"customtypes",
	"enum",
	"overrides",
	"generics",
	"inline",
	"inlineonly",
	"nested",
	"parameters",
	"petstore",
	"refs",
	"responses",
	"security",
	"standalone_api",
	"tags",
}

// openAPIVersions are the versions with golden files. The suffix names the
// file: examples/petstore/petstore_31.yaml.
//
// 3.2 is absent on purpose. The generator has no 3.2-specific emission —
// SchemaBuilder treats "3.1" and "3.2" identically in every method, and the
// only thing that differs is the version header — so a 3.2 golden could never
// hold anything its 3.1 twin does not. Add it here when 3.2 grows emission of
// its own; until then there is nothing for it to lock.
var openAPIVersions = []struct {
	version string
	suffix  string
}{
	{"3.0", "30"},
	{"3.1", "31"},
}

func TestGoldenFiles(t *testing.T) {
	for _, name := range examples {
		for _, v := range openAPIVersions {
			t.Run(name+"_"+v.suffix, func(t *testing.T) {
				dir := filepath.Join("..", "..", "examples", name)

				got, err := buildSpec(dir, v.version, generator.FormatYAML)
				if err != nil {
					t.Fatalf("buildSpec: %v", err)
				}

				goldenFile := filepath.Join(dir, name+"_"+v.suffix+".yaml")
				compareGolden(t, goldenFile, got)
			})
		}
	}
}

// compareGolden compares generated output against a golden file, or rewrites
// the golden file when -update is set.
func compareGolden(t *testing.T, goldenFile string, got []byte) {
	t.Helper()

	if *update {
		if err := os.WriteFile(goldenFile, got, 0644); err != nil {
			t.Fatalf("update golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}

	if string(got) == string(want) {
		return
	}

	t.Errorf("output differs from %s", goldenFile)
	gotLines := strings.Split(string(got), "\n")
	wantLines := strings.Split(string(want), "\n")
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
