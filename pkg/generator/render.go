package generator

import "fmt"

import v3 "github.com/pb33f/libopenapi/datamodel/high/v3"

// OutputFormat represents the output format.
type OutputFormat string

const (
	FormatJSON OutputFormat = "json"
	FormatYAML OutputFormat = "yaml"
)

// Generator generates OpenAPI specifications using libopenapi's v3 models.
// All output formatting (key order, indentation, quoting) is owned by
// libopenapi and go.yaml.in/yaml/v4; the golden files depend on the pinned
// versions of both, and on rendering through Document.Render().
type Generator struct {
	version       string // "3.0", "3.1", "3.2"
	schemaBuilder *SchemaBuilder
}

// NewGenerator creates a generator for the given OpenAPI version.
func NewGenerator(version string) (*Generator, error) {
	switch version {
	case "3.0", "3.1", "3.2":
	default:
		return nil, fmt.Errorf("unsupported OpenAPI version: %q (want 3.0, 3.1, or 3.2)", version)
	}
	return &Generator{
		version:       version,
		schemaBuilder: NewSchemaBuilder(version),
	}, nil
}

// openAPIVersion returns the full version string for the document header.
func (g *Generator) openAPIVersion() string {
	switch g.version {
	case "3.1":
		return "3.1.0"
	case "3.2":
		return "3.2.0"
	default:
		return "3.0.3"
	}
}

// Render renders the document in the requested format.
func (g *Generator) Render(doc *v3.Document, format OutputFormat) ([]byte, error) {
	switch format {
	case FormatJSON:
		return doc.RenderJSON("  ")
	case FormatYAML:
		return doc.Render()
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
