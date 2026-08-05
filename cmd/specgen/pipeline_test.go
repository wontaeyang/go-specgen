package main

import (
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/generator"
)

func TestBuildSpec_UnreadablePackage(t *testing.T) {
	_, err := buildSpec("./testdata/does-not-exist", "3.1", generator.FormatYAML)
	if err == nil {
		t.Fatal("building a spec from a missing package should fail")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should name the stage that failed, got: %v", err)
	}
}

func TestBuildSpec_UnknownVersion(t *testing.T) {
	if _, err := buildSpec("./testdata/readwrite", "4.0", generator.FormatYAML); err == nil {
		t.Fatal("an unknown OpenAPI version should fail")
	}
}
