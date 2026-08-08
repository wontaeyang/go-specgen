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
	p := parser.NewParser(packagePath)
	parsed, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	r, err := resolver.NewResolver(packagePath, p.Comments())
	if err != nil {
		return nil, fmt.Errorf("resolve: %w", err)
	}

	resolved, err := r.Resolve(parsed)
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
	if spec.YAML, err = gen.Render(doc, generator.FormatYAML); err != nil {
		return nil, fmt.Errorf("render yaml: %w", err)
	}
	if spec.JSON, err = gen.Render(doc, generator.FormatJSON); err != nil {
		return nil, fmt.Errorf("render json: %w", err)
	}

	return spec, nil
}
