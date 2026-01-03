package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wontaeyang/go-specgen/pkg/generator"
	"github.com/wontaeyang/go-specgen/pkg/parser"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
	"github.com/wontaeyang/go-specgen/pkg/validator"
)

const (
	version = "1.0.0"
)

func main() {
	// Define flags
	packagePath := flag.String("package", ".", "Path to the Go package to parse")
	outputPath := flag.String("output", "openapi.yaml", "Output file path")
	format := flag.String("format", "yaml", "Output format: json or yaml")
	openapiVersion := flag.String("openapi", "3.0", "OpenAPI version: 3.0, 3.1, or 3.2")
	showVersion := flag.Bool("version", false, "Show version")
	showHelp := flag.Bool("help", false, "Show help")

	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("specgen version %s\n", version)
		os.Exit(0)
	}

	// Show help
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Validate format
	var outputFormat generator.OutputFormat
	switch *format {
	case "json":
		outputFormat = generator.FormatJSON
	case "yaml", "yml":
		outputFormat = generator.FormatYAML
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid format '%s'. Must be 'json' or 'yaml'\n", *format)
		os.Exit(1)
	}

	// Validate OpenAPI version
	if *openapiVersion != "3.0" && *openapiVersion != "3.1" && *openapiVersion != "3.2" {
		fmt.Fprintf(os.Stderr, "Error: invalid OpenAPI version '%s'. Must be '3.0', '3.1', or '3.2'\n", *openapiVersion)
		os.Exit(1)
	}

	// Run the generation
	if err := generate(*packagePath, *outputPath, outputFormat, *openapiVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated OpenAPI spec: %s\n", *outputPath)
}

func generate(packagePath, outputPath string, format generator.OutputFormat, openapiVersion string) error {
	// Step 1: Parse the package
	fmt.Println("Parsing package...")
	p := parser.NewParser(packagePath)
	parsed, err := p.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse package: %w", err)
	}

	// Step 2: Resolve types
	fmt.Println("Resolving types...")
	r, err := resolver.NewResolver(packagePath, p.Comments())
	if err != nil {
		return fmt.Errorf("failed to create resolver: %w", err)
	}

	resolved, err := r.Resolve(parsed)
	if err != nil {
		return fmt.Errorf("failed to resolve types: %w", err)
	}

	// Step 3: Validate
	fmt.Println("Validating...")
	v := validator.NewValidator()
	if err := v.Validate(resolved); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Step 4: Generate OpenAPI spec
	fmt.Println("Generating OpenAPI spec...")
	gen := generator.NewGenerator(openapiVersion)
	spec, err := gen.Generate(resolved)
	if err != nil {
		return fmt.Errorf("failed to generate spec: %w", err)
	}

	// Step 5: Render to output format
	fmt.Println("Rendering output...")
	data, err := gen.Render(spec, format)
	if err != nil {
		return fmt.Errorf("failed to render spec: %w", err)
	}

	// Step 6: Write to file
	fmt.Printf("Writing to %s...\n", outputPath)

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

func printHelp() {
	fmt.Println("specgen - Generate OpenAPI specifications from Go code")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  specgen [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -package string")
	fmt.Println("        Path to the Go package to parse (default \".\")")
	fmt.Println("  -output string")
	fmt.Println("        Output file path (default \"openapi.yaml\")")
	fmt.Println("  -format string")
	fmt.Println("        Output format: json or yaml (default \"yaml\")")
	fmt.Println("  -openapi string")
	fmt.Println("        OpenAPI version: 3.0, 3.1, or 3.2 (default \"3.0\")")
	fmt.Println("  -version")
	fmt.Println("        Show version")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate OpenAPI spec from current directory")
	fmt.Println("  specgen")
	fmt.Println()
	fmt.Println("  # Generate JSON spec from a specific package")
	fmt.Println("  specgen -package ./api/handlers -format json -output openapi.json")
	fmt.Println()
	fmt.Println("  # Generate OpenAPI 3.1 spec")
	fmt.Println("  specgen -openapi 3.1 -output openapi-3.1.yaml")
}
