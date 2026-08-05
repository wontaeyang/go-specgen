package main

import (
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/generator"
)

// The pipeline itself is covered end to end by TestGoldenFiles. What is left
// to check here is that a failing stage is named in the error.
func TestBuildSpec_UnreadablePackage(t *testing.T) {
	_, err := buildSpec("./does-not-exist", "3.1", generator.FormatYAML)
	if err == nil {
		t.Fatal("building a spec from a missing package should fail")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should name the stage that failed, got: %v", err)
	}
}
