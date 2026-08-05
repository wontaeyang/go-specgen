package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wontaeyang/go-specgen/pkg/generator"
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
	fmt.Printf("Generating OpenAPI %s spec from %s...\n", openapiVersion, packagePath)

	data, err := buildSpec(packagePath, openapiVersion, format)
	if err != nil {
		return err
	}

	fmt.Printf("Writing to %s...\n", outputPath)

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
	// PrintDefaults writes to stderr by default; asked-for help belongs on
	// stdout with the rest of the text.
	flag.CommandLine.SetOutput(os.Stdout)

	fmt.Println("specgen - Generate OpenAPI specifications from Go code")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  specgen [options]")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
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
