package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const version = "1.0.0"

func main() {
	packagePath := flag.String("package", ".", "Path to the Go package to parse")
	yamlPath := flag.String("yaml", "", "Write the spec as YAML to this path")
	jsonPath := flag.String("json", "", "Write the spec as JSON to this path")
	openapiVersion := flag.String("openapi", "3.1", "OpenAPI version: 3.1 or 3.2")
	showVersion := flag.Bool("version", false, "Show version")
	showHelp := flag.Bool("help", false, "Show help")

	flag.Parse()

	if *showVersion {
		fmt.Printf("specgen version %s\n", version)
		os.Exit(0)
	}

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	if err := run(*packagePath, *yamlPath, *jsonPath, *openapiVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(packagePath, yamlPath, jsonPath, openapiVersion string) error {
	// Output lands only where it was named. Defaulting to openapi.yaml in the
	// working directory writes a file the caller never asked for.
	if yamlPath == "" && jsonPath == "" {
		return fmt.Errorf("specify -yaml and/or -json to say where the spec should be written")
	}

	if openapiVersion != "3.1" && openapiVersion != "3.2" {
		return fmt.Errorf("invalid OpenAPI version %q: must be 3.1 or 3.2", openapiVersion)
	}

	// One pipeline pass feeds both files, so they can never disagree.
	spec, err := buildSpec(packagePath, openapiVersion)
	if err != nil {
		return err
	}

	for _, out := range []struct {
		path string
		data []byte
	}{
		{yamlPath, spec.YAML},
		{jsonPath, spec.JSON},
	} {
		if out.path == "" {
			continue
		}
		if err := writeFile(out.path, out.data); err != nil {
			return err
		}
		fmt.Printf("Wrote %s\n", out.path)
	}

	return nil
}

// writeFile writes data to path, creating the parent directory if needed.
func writeFile(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create output directory %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
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
	fmt.Println("  -yaml string")
	fmt.Println("        Write the spec as YAML to this path")
	fmt.Println("  -json string")
	fmt.Println("        Write the spec as JSON to this path")
	fmt.Println("  -openapi string")
	fmt.Println("        OpenAPI version: 3.1 or 3.2 (default \"3.1\")")
	fmt.Println("  -version")
	fmt.Println("        Show version")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("At least one of -yaml or -json is required; the spec is written")
	fmt.Println("only to the paths you name.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate both formats from one pass")
	fmt.Println("  specgen -package ./api -yaml openapi.yaml -json openapi.json")
	fmt.Println()
	fmt.Println("  # YAML only")
	fmt.Println("  specgen -package ./api -yaml docs/openapi.yaml")
}
