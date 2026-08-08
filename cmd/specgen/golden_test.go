package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update golden and fixture files")

const (
	// examplesDir holds the documented, hand-written example packages. Every
	// subdirectory is a golden case; there is no list to keep in sync.
	examplesDir = "../../examples"

	// goldenVersion is the OpenAPI version all goldens and fixtures are
	// rendered at. 3.2 differs only in the version string — see TestVersionGuard.
	goldenVersion = "3.1"
)

// TestGoldenFiles renders every package under examples/ and compares it
// byte-for-byte against the <name>.yaml checked in beside it.
func TestGoldenFiles(t *testing.T) {
	for _, name := range subdirs(t, examplesDir) {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(examplesDir, name)

			spec, err := buildSpec(dir, goldenVersion)
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			compareGolden(t, filepath.Join(dir, name+".yaml"), spec.YAML)
		})
	}
}

// subdirs returns the names of every directory directly under root, sorted.
func subdirs(t *testing.T, root string) []string {
	t.Helper()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}

	// os.ReadDir already sorts by filename, so subtest order is deterministic.
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}

	if len(names) == 0 {
		t.Fatalf("no packages found under %s", root)
	}

	return names
}

// compareGolden compares got against the file at path, or overwrites it when
// the test binary was run with -update.
func compareGolden(t *testing.T, path string, got []byte) {
	t.Helper()

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatalf("update %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run the test with -update to create it)", path, err)
	}

	if string(got) == string(want) {
		return
	}

	t.Errorf("output differs from %s\n%s", path, firstDiff(string(want), string(got)))
	t.Errorf("re-run with -update and review the diff if the change is intended")
}

// firstDiff describes the first line where want and got disagree. The full
// change is meant to be reviewed with -update plus `git diff`; this is just
// enough to recognize a failure without regenerating anything.
func firstDiff(want, got string) string {
	wantLines, gotLines := splitLines(want), splitLines(got)

	for i := 0; i < len(wantLines) && i < len(gotLines); i++ {
		if wantLines[i] != gotLines[i] {
			return fmt.Sprintf("first difference at line %d:\n  want: %s\n  got:  %s\n(%d want lines, %d got lines)",
				i+1, wantLines[i], gotLines[i], len(wantLines), len(gotLines))
		}
	}

	return fmt.Sprintf("common prefix matches; line count differs: want %d, got %d", len(wantLines), len(gotLines))
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
