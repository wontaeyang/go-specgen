package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This is the log harness: what specgen says about a package it generated
// anyway. The golden harness pins the document, the error harness pins the
// message a failure produces, and neither can see stderr — so a rule whose whole
// effect is a line on stderr has no other place to be asserted.
//
// It lives here rather than in pkg/parser because the subject is a package under
// examples/, and examples/ belongs to this harness. A unit test reaching two
// directories up to read another tree's fixtures couples the two the wrong way
// round.

// TestSkippedAnnotationsAreLogged covers the declarations specgen passes over
// because their doc comment opens with a name it does not recognize.
//
// These are reported rather than raised. A name the grammar has never heard of
// cannot be told apart from prose, so failing the run would reject a package that
// merely documents itself with a line starting @ — but saying nothing is how a
// misspelled @endpoint used to cost a whole operation in silence.
//
// The other half of the rule is pinned by examples/skipped/skipped.yaml, where
// Widget and /widgets are absent precisely because @schemaa and @endpoin were
// passed over. The document shows what was lost; only the log says why.
func TestSkippedAnnotationsAreLogged(t *testing.T) {
	var out bytes.Buffer

	flags, prefix := log.Flags(), log.Prefix()
	log.SetOutput(&out)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(flags)
		log.SetPrefix(prefix)
	})

	// Through buildSpec, so this exercises the same pipeline the CLI and the
	// golden harness do rather than a stage of it in isolation.
	if _, err := buildSpec(filepath.Join(examplesDir, "skipped"), goldenVersion); err != nil {
		t.Fatalf("build: %v; the package is meant to generate", err)
	}

	logged := out.String()

	// Order is not meaningful: these are written as each declaration is passed
	// over, across two stages of the parser.
	want := []string{
		"type Widget: @schemaa is not an annotation; it was skipped",
		"func ListWidgets: @endpoin is not an annotation; it was skipped",
		"field Widget.ID: @fild is not an annotation; it was skipped",
		"declaration ListGadgets.filters: @quer is not an annotation; it was skipped",
	}
	for _, line := range want {
		if !strings.Contains(logged, line) {
			t.Errorf("missing from the log:\n  %s\ngot:\n%s", line, logged)
		}
	}

	// Everything the package documents in prose must stay quiet: Gadget names
	// @schema only after discussing it, Helper carries no @ at all, and Escaped
	// opens with \@ — the escape the rest of the syntax uses for a literal @.
	for _, quiet := range []string{"type Gadget", "func Helper", "func Escaped", "@author"} {
		if strings.Contains(logged, quiet) {
			t.Errorf("%q should not be reported; got:\n%s", quiet, logged)
		}
	}
}
