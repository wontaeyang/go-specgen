package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	switch *format {
	case "json", "yaml", "yml":
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid format '%s'. Must be 'json' or 'yaml'\n", *format)
		os.Exit(1)
	}

	// Validate OpenAPI version
	if *openapiVersion != "3.0" && *openapiVersion != "3.1" && *openapiVersion != "3.2" {
		fmt.Fprintf(os.Stderr, "Error: invalid OpenAPI version '%s'. Must be '3.0', '3.1', or '3.2'\n", *openapiVersion)
		os.Exit(1)
	}

	spec, err := buildSpec(*packagePath, *openapiVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	data := spec.YAML
	if *format == "json" {
		data = spec.JSON
	}

	if err := writeFile(*outputPath, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated OpenAPI spec: %s\n", *outputPath)
}

// writeFile writes data to path, creating the parent directory if needed.
func writeFile(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
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
