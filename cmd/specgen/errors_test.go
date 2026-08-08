package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// This is the error harness: every package under testdata/errors/ must fail,
// and the message must match expected_error.txt. The golden harness, for
// packages that must generate, is in golden_test.go.
//
// It exists because a success-only suite cannot notice a check that stopped
// firing. Every golden asserts "this input produces this output"; none can
// assert "this input produces no output", so deleting a validation rule leaves
// every golden passing. That is not hypothetical — validateEndpointTags was
// guarded by len(api.Tags) > 0 and never ran for an API that declared no tags,
// and nothing failed.
//
// The message text is product surface, not internals: when specgen cannot
// generate, the message is the entire output.
//
// These packages live under testdata/ rather than beside examples/ because they
// are wrong on purpose. testdata is the one directory name `go build ./...` and
// `go vet ./...` refuse to descend into, which is exactly right for a package
// with a chan field or a mis-tagged struct.
//
// All three stages accumulate to the same depth: every bad annotation is
// reported, not just the first one in a declaration. So a package here can show
// as many errors as it has mistakes, and a fixture that shows one usually has
// one. What still stops the run is the stage boundary -- the resolver never
// sees a package that failed to parse -- so no fixture can show parse and
// resolve errors together.
const errorsDir = "testdata/errors"

// TestErrorFixtures pins the message text of every failure specgen is supposed
// to produce.
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

// pkgDir builds a package path for buildSpec.
//
// The "./" prefix is no longer required — ExtractComments adds one to any bare
// relative path, since packages.Load would otherwise read it as an import path.
// It stays because the prefix is what the fixture messages are normalized
// against, and a package path that says it is a directory reads better here.
func pkgDir(root, name string) string {
	return "./" + filepath.Join(root, name)
}
