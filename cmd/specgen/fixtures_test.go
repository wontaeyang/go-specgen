package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// Fixture tests cover behavior the examples/ goldens never reach. The examples
// are documentation first and happen to be tests; these are tests first, and
// exist to make every behavior change in the refactor show up as a reviewable
// diff rather than a surprise.
//
//	testdata/features/<case>/  a Go package + expected.yaml     — must build
//	testdata/errors/<case>/    a Go package + expected_error.txt — must fail
//	testdata/json/<name>.json  JSON rendering of examples/<name>
//
// Regenerate all of them with:
//
//	go test ./cmd/specgen -update
const (
	featuresDir = "testdata/features"
	errorsDir   = "testdata/errors"
	jsonDir     = "testdata/json"
)

// pkgDir builds a package path for buildSpec. The "./" prefix matters:
// packages.Load reads an unprefixed relative path as an import path, not a
// directory. examples/ escapes this because "../../examples/x" already starts
// with a dot.
func pkgDir(root, name string) string {
	return "./" + filepath.Join(root, name)
}

// TestFeatureFixtures renders each feature package and compares it against the
// expected.yaml beside it. A fixture that fails to build is a harness failure —
// packages that are supposed to fail belong in testdata/errors.
func TestFeatureFixtures(t *testing.T) {
	for _, name := range subdirs(t, featuresDir) {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(featuresDir, name)

			spec, err := buildSpec(pkgDir(featuresDir, name), goldenVersion)
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			compareGolden(t, filepath.Join(dir, "expected.yaml"), spec.YAML)
		})
	}
}

// TestErrorFixtures pins the message text of every failure specgen is supposed
// to produce. Without this, an error path can silently stop firing.
func TestErrorFixtures(t *testing.T) {
	for _, name := range subdirs(t, errorsDir) {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(errorsDir, name)

			if _, err := buildSpec(pkgDir(errorsDir, name), goldenVersion); err != nil {
				// Paths appear in some messages; keep the fixture machine-independent.
				got := strings.ReplaceAll(err.Error(), pkgDir(errorsDir, name), "<pkg>")
				compareGolden(t, filepath.Join(dir, "expected_error.txt"), []byte(got+"\n"))
				return
			}

			t.Fatalf("expected this package to fail, but it generated a spec")
		})
	}
}

// jsonExamples are the examples rendered to JSON as well as YAML. Both formats
// serialize the same *v3.Document, so the YAML goldens already prove specgen's
// correctness — these exist to catch serializer breakage, such as a libopenapi
// upgrade changing JSON output while leaving YAML alone.
var jsonExamples = []string{"nested", "petstore", "responses"}

func TestJSONRendering(t *testing.T) {
	for _, name := range jsonExamples {
		t.Run(name, func(t *testing.T) {
			spec, err := buildSpec(filepath.Join(examplesDir, name), goldenVersion)
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			compareGolden(t, filepath.Join(jsonDir, name+".json"), spec.JSON)
		})
	}
}

// TestVersionGuard pins the fact that specgen has no 3.2-specific behavior:
// 3.2 output is 3.1 output with a different version line. If a 3.2 branch is
// ever added, this fails loudly instead of the difference drifting in unseen.
func TestVersionGuard(t *testing.T) {
	for _, name := range subdirs(t, examplesDir) {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(examplesDir, name)

			v31, err := buildSpec(dir, "3.1")
			if err != nil {
				t.Fatalf("build 3.1: %v", err)
			}

			v32, err := buildSpec(dir, "3.2")
			if err != nil {
				t.Fatalf("build 3.2: %v", err)
			}

			want := strings.Replace(string(v31.YAML), "openapi: 3.1.0", "openapi: 3.2.0", 1)
			if got := string(v32.YAML); got != want {
				t.Errorf("3.2 output differs from 3.1 beyond the version line\n%s", firstDiff(want, got))
			}
		})
	}
}
