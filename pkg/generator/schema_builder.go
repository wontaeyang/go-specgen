package generator

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// SchemaBuilder provides version-aware schema construction for OpenAPI 3.0, 3.1, and 3.2.
// It handles the differences between versions, particularly around nullable handling
// and exclusive bounds.
type SchemaBuilder struct {
	version string // "3.0", "3.1", "3.2"
}

// NewSchemaBuilder creates a new SchemaBuilder for the specified OpenAPI version.
func NewSchemaBuilder(version string) *SchemaBuilder {
	return &SchemaBuilder{version: version}
}

// NewSchema creates a new empty Schema instance.
func (sb *SchemaBuilder) NewSchema() *base.Schema {
	return &base.Schema{}
}

// SetType sets the type field of the schema.
// In OpenAPI, Type is always an array of strings.
func (sb *SchemaBuilder) SetType(schema *base.Schema, t string) {
	schema.Type = []string{t}
}

// SetTypes sets multiple types for the schema (3.1+ feature).
func (sb *SchemaBuilder) SetTypes(schema *base.Schema, types []string) {
	schema.Type = types
}

// SetNullable handles nullable differently per OpenAPI version:
// - OpenAPI 3.0: Sets nullable: true
// - OpenAPI 3.1+: Appends "null" to the type array
func (sb *SchemaBuilder) SetNullable(schema *base.Schema, nullable bool) {
	if !nullable {
		return
	}
	switch sb.version {
	case "3.0":
		schema.Nullable = &nullable
	case "3.1", "3.2":
		// 3.1+ uses type array: ["string", "null"]
		if len(schema.Type) > 0 {
			schema.Type = append(schema.Type, "null")
		}
	}
}

// SetExclusiveMinimum sets exclusive minimum constraint (version-aware).
// - OpenAPI 3.0: Sets minimum + exclusiveMinimum as boolean (true)
// - OpenAPI 3.1+: Sets exclusiveMinimum as the numeric value itself
//
// Note: Not currently used - resolver doesn't support @exclusiveMinimum yet.
func (sb *SchemaBuilder) SetExclusiveMinimum(schema *base.Schema, value float64) {
	switch sb.version {
	case "3.0":
		// 3.0: set minimum + exclusiveMinimum as boolean
		schema.Minimum = &value
		t := true
		schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{A: t}
	case "3.1", "3.2":
		// 3.1+: exclusiveMinimum is the value itself
		schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{N: 1, B: value}
	}
}

// SetExclusiveMaximum sets exclusive maximum constraint (version-aware).
// - OpenAPI 3.0: Sets maximum + exclusiveMaximum as boolean (true)
// - OpenAPI 3.1+: Sets exclusiveMaximum as the numeric value itself
//
// Note: Not currently used - resolver doesn't support @exclusiveMaximum yet.
func (sb *SchemaBuilder) SetExclusiveMaximum(schema *base.Schema, value float64) {
	switch sb.version {
	case "3.0":
		// 3.0: set maximum + exclusiveMaximum as boolean
		schema.Maximum = &value
		t := true
		schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{A: t}
	case "3.1", "3.2":
		// 3.1+: exclusiveMaximum is the value itself
		schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{N: 1, B: value}
	}
}

// Is30 returns true if the target version is OpenAPI 3.0.
func (sb *SchemaBuilder) Is30() bool {
	return sb.version == "3.0"
}

// Is31Plus returns true if the target version is OpenAPI 3.1 or later.
func (sb *SchemaBuilder) Is31Plus() bool {
	return sb.version == "3.1" || sb.version == "3.2"
}

// Version returns the OpenAPI version string.
func (sb *SchemaBuilder) Version() string {
	return sb.version
}
