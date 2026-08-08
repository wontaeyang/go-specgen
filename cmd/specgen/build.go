package main

import (
	"fmt"

	"github.com/wontaeyang/go-specgen/pkg/generator"
	"github.com/wontaeyang/go-specgen/pkg/parser"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
	"github.com/wontaeyang/go-specgen/pkg/validator"
)

// Spec holds one pipeline pass rendered in every output format.
type Spec struct {
	YAML []byte
	JSON []byte
}

// buildSpec runs parse -> resolve -> validate -> generate over a package
// directory and renders the resulting document.
//
// This is the only definition of the pipeline. The CLI, the golden tests, and
// the fixture tests all go through it, so none of them can drift from the
// others or from each other's notion of what specgen does.
func buildSpec(packagePath, openapiVersion string) (*Spec, error) {
	parsed, err := parser.Parse(packagePath)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	resolved, err := resolver.NewResolver(parsed).Resolve()
	if err != nil {
		return nil, fmt.Errorf("resolve: %w", err)
	}

	if err := validator.NewValidator().Validate(resolved); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	gen := generator.NewGenerator(openapiVersion)
	doc, err := gen.Generate(resolved)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}

	spec := &Spec{}
	if spec.YAML, err = gen.RenderYAML(doc); err != nil {
		return nil, fmt.Errorf("render yaml: %w", err)
	}
	if spec.JSON, err = gen.RenderJSON(doc); err != nil {
		return nil, fmt.Errorf("render json: %w", err)
	}

	return spec, nil
}
