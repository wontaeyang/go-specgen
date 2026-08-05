package main

import (
	"fmt"

	"github.com/wontaeyang/go-specgen/pkg/generator"
	"github.com/wontaeyang/go-specgen/pkg/parser"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
	"github.com/wontaeyang/go-specgen/pkg/validator"
)

// buildSpec runs the whole pipeline over one package: parse the annotations,
// resolve them against the Go types, validate the result, then generate and
// render the document. It is the single path the CLI and the tests share.
func buildSpec(packageDir, openapiVersion string, format generator.OutputFormat) ([]byte, error) {
	parsed, err := parser.Parse(packageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse package: %w", err)
	}

	resolved, err := resolver.Resolve(parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve types: %w", err)
	}

	if err := validator.Validate(resolved); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	gen, err := generator.NewGenerator(openapiVersion)
	if err != nil {
		return nil, err
	}

	data, err := gen.Render(gen.Generate(resolved), format)
	if err != nil {
		return nil, fmt.Errorf("failed to render spec: %w", err)
	}
	return data, nil
}
