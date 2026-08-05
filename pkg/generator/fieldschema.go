package generator

import (
	"fmt"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/wontaeyang/go-specgen/pkg/resolver"
	"go.yaml.in/yaml/v4"
)

// fieldSchema builds the schema for a field. It is the single field-schema
// builder: schema properties, inline bodies, and bind-wrapper fields all go
// through it. Anonymous-struct slices take precedence over TypeInfo, in the
// same order the legacy builder checked them.
func (g *Generator) fieldSchema(field *resolver.Field) *base.SchemaProxy {
	if len(field.Inline) > 0 {
		proxy := g.inlineObjectSchema(field.Inline)
		g.applyConstraints(proxy.Schema(), field)
		return proxy
	}

	if len(field.ItemsInline) > 0 {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "array")
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: g.inlineObjectSchema(field.ItemsInline),
		}
		g.applyConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	if len(field.MapValueInline) > 0 {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "object")
		schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: g.inlineObjectSchema(field.MapValueInline),
		}
		g.applyConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	ti := field.Type

	if ti.IsArray {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "array")
		var items *base.SchemaProxy
		if ti.ItemsRef != "" {
			items = base.CreateSchemaProxyRef(componentRef(ti.ItemsRef))
		} else {
			itemSchema := g.schemaBuilder.NewSchema()
			g.schemaBuilder.SetType(itemSchema, ti.Items)
			itemSchema.Format = ti.ItemsFormat
			items = base.CreateSchemaProxy(itemSchema)
		}
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{A: items}
		g.applyConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	if ti.IsMap {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "object")
		var values *base.SchemaProxy
		if ti.MapValueRef != "" {
			values = base.CreateSchemaProxyRef(componentRef(ti.MapValueRef))
		} else {
			valueSchema := g.schemaBuilder.NewSchema()
			g.schemaBuilder.SetType(valueSchema, ti.MapValue)
			valueSchema.Format = ti.MapValueFormat
			values = base.CreateSchemaProxy(valueSchema)
		}
		schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{A: values}
		g.applyConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	if ti.Ref != "" {
		return g.refSchema(componentRef(ti.Ref), field)
	}

	if ti.IsAny {
		schema := g.schemaBuilder.NewSchema()
		g.applyConstraints(schema, field)
		return base.CreateSchemaProxy(schema)
	}

	schema := g.schemaBuilder.NewSchema()
	g.schemaBuilder.SetType(schema, ti.OpenAPI)
	g.applyConstraints(schema, field)
	return base.CreateSchemaProxy(schema)
}

// refSchema builds the schema for a field whose type is a named @schema.
// A bare $ref is emitted when the field has no annotation keywords and is not
// nullable. Otherwise sibling keywords are attached: natively in OpenAPI 3.1+
// ($ref can carry siblings per JSON Schema 2020-12), or via an allOf wrapper
// in 3.0 (where $ref siblings are forbidden). Nullable refs always need a
// composition wrapper, since a $ref cannot also be typed "null".
func (g *Generator) refSchema(refPath string, field *resolver.Field) *base.SchemaProxy {
	if field.Nullable {
		wrapper := g.schemaBuilder.NewSchema()
		if g.schemaBuilder.Is31Plus() {
			// OpenAPI 3.1+: express null via oneOf (siblings would intersect, not union).
			wrapper.OneOf = []*base.SchemaProxy{
				base.CreateSchemaProxyRef(refPath),
				base.CreateSchemaProxy(&base.Schema{Type: []string{"null"}}),
			}
		} else {
			// OpenAPI 3.0: $ref cannot have siblings; wrap in allOf and mark nullable.
			wrapper.AllOf = []*base.SchemaProxy{base.CreateSchemaProxyRef(refPath)}
		}
		g.applyConstraints(wrapper, field)
		return base.CreateSchemaProxy(wrapper)
	}

	siblings := g.schemaBuilder.NewSchema()
	g.applyConstraints(siblings, field)

	if g.schemaBuilder.Is31Plus() {
		// An empty siblings schema renders as a bare $ref, so no special-casing.
		return base.CreateSchemaProxyRefWithSchema(refPath, siblings)
	}

	// OpenAPI 3.0: wrap in allOf only when there is something to carry; an
	// unannotated ref stays a bare $ref rather than a noisy allOf wrapper.
	if isEmptySchema(siblings) {
		return base.CreateSchemaProxyRef(refPath)
	}
	siblings.AllOf = []*base.SchemaProxy{base.CreateSchemaProxyRef(refPath)}
	return base.CreateSchemaProxy(siblings)
}

// isEmptySchema reports whether applyConstraints left the schema untouched.
// It checks exactly the fields applyConstraints can set.
func isEmptySchema(s *base.Schema) bool {
	return s.Description == "" && s.Format == "" && s.Enum == nil &&
		s.Example == nil && s.Default == nil && s.Pattern == "" &&
		s.MinLength == nil && s.MaxLength == nil && s.MinItems == nil &&
		s.MaxItems == nil && s.UniqueItems == nil && s.Minimum == nil &&
		s.Maximum == nil && s.ExclusiveMinimum == nil && s.ExclusiveMaximum == nil &&
		s.Nullable == nil && s.Type == nil && s.Deprecated == nil &&
		s.ReadOnly == nil && s.WriteOnly == nil
}

// inlineObjectSchema builds an object schema from resolved fields. With no
// fields it stays a bare object schema (no properties key), matching the
// legacy inline-schema behavior.
func (g *Generator) inlineObjectSchema(fields []*resolver.Field) *base.SchemaProxy {
	schema := g.schemaBuilder.NewSchema()
	g.schemaBuilder.SetType(schema, "object")

	if len(fields) == 0 {
		return base.CreateSchemaProxy(schema)
	}

	props := orderedmap.New[string, *base.SchemaProxy]()
	var required []string

	for _, field := range fields {
		props.Set(field.Name, g.fieldSchema(field))
		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema.Properties = props
	if len(required) > 0 {
		schema.Required = required
	}

	return base.CreateSchemaProxy(schema)
}

// applyConstraints applies description, format, and every @field constraint
// to a schema. It is the single source of truth for which keywords exist.
// Array enums land inside the items schema; everything else lands on the
// schema itself.
func (g *Generator) applyConstraints(schema *base.Schema, field *resolver.Field) {
	if field.Description != "" {
		schema.Description = field.Description
	}
	if field.Format != "" {
		schema.Format = field.Format
	}
	if len(field.Enum) > 0 {
		// The enum type governs !!int tagging; for arrays the field-level
		// type is not "integer", so array enums stay untagged here (legacy
		// behavior — parameter schemas tag by item type instead).
		enumValues := enumNodes(field.Enum, field.Type.OpenAPI)
		if field.Type.IsArray {
			if schema.Items != nil && schema.Items.A != nil {
				itemSchema, _ := schema.Items.A.BuildSchema()
				if itemSchema != nil {
					itemSchema.Enum = enumValues
				}
			}
		} else {
			schema.Enum = enumValues
		}
	}
	if field.Example != "" {
		schema.Example = scalarNode(field.Example, field.Type.OpenAPI)
	}
	if field.Default != "" {
		schema.Default = scalarNode(field.Default, field.Type.OpenAPI)
	}
	if field.Pattern != "" {
		schema.Pattern = field.Pattern
	}
	if field.MinLength != nil {
		val := int64(*field.MinLength)
		schema.MinLength = &val
	}
	if field.MaxLength != nil {
		val := int64(*field.MaxLength)
		schema.MaxLength = &val
	}
	if field.MinItems != nil {
		val := int64(*field.MinItems)
		schema.MinItems = &val
	}
	if field.MaxItems != nil {
		val := int64(*field.MaxItems)
		schema.MaxItems = &val
	}
	if field.UniqueItems {
		t := true
		schema.UniqueItems = &t
	}
	if field.Minimum != nil {
		schema.Minimum = field.Minimum
	}
	if field.Maximum != nil {
		schema.Maximum = field.Maximum
	}
	if field.ExclusiveMinimum != nil {
		g.schemaBuilder.SetExclusiveMinimum(schema, *field.ExclusiveMinimum)
	}
	if field.ExclusiveMaximum != nil {
		g.schemaBuilder.SetExclusiveMaximum(schema, *field.ExclusiveMaximum)
	}
	if field.Nullable {
		g.schemaBuilder.SetNullable(schema, true)
	}
	if field.Constraints.Deprecated {
		t := true
		schema.Deprecated = &t
	}
	if field.ReadOnly {
		t := true
		schema.ReadOnly = &t
	}
	if field.WriteOnly {
		t := true
		schema.WriteOnly = &t
	}
}

// parameterFieldSchema builds the schema for an operation parameter. It
// deliberately differs from fieldSchema: no description (it lives on the
// parameter), no inline or ref support, and array enums are typed by the
// item type (so integer items get !!int tags, unlike field schemas).
func (g *Generator) parameterFieldSchema(field *resolver.Field) *base.SchemaProxy {
	schema := g.schemaBuilder.NewSchema()

	if field.Type.IsArray {
		g.schemaBuilder.SetType(schema, "array")
		itemSchema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(itemSchema, field.Type.Items)
		itemSchema.Format = field.Type.ItemsFormat
		if len(field.Enum) > 0 {
			itemSchema.Enum = enumNodes(field.Enum, field.Type.Items)
		}
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: base.CreateSchemaProxy(itemSchema),
		}
	} else {
		g.schemaBuilder.SetType(schema, field.Type.OpenAPI)
		if len(field.Enum) > 0 {
			schema.Enum = enumNodes(field.Enum, field.Type.OpenAPI)
		}
	}

	if field.Format != "" {
		schema.Format = field.Format
	}
	if field.Example != "" {
		schema.Example = scalarNode(field.Example, field.Type.OpenAPI)
	}
	if field.Default != "" {
		schema.Default = scalarNode(field.Default, field.Type.OpenAPI)
	}
	if field.Pattern != "" {
		schema.Pattern = field.Pattern
	}
	if field.MinLength != nil {
		val := int64(*field.MinLength)
		schema.MinLength = &val
	}
	if field.MaxLength != nil {
		val := int64(*field.MaxLength)
		schema.MaxLength = &val
	}
	if field.MinItems != nil {
		val := int64(*field.MinItems)
		schema.MinItems = &val
	}
	if field.MaxItems != nil {
		val := int64(*field.MaxItems)
		schema.MaxItems = &val
	}
	if field.UniqueItems {
		t := true
		schema.UniqueItems = &t
	}
	if field.Minimum != nil {
		schema.Minimum = field.Minimum
	}
	if field.Maximum != nil {
		schema.Maximum = field.Maximum
	}
	if field.ExclusiveMinimum != nil {
		g.schemaBuilder.SetExclusiveMinimum(schema, *field.ExclusiveMinimum)
	}
	if field.ExclusiveMaximum != nil {
		g.schemaBuilder.SetExclusiveMaximum(schema, *field.ExclusiveMaximum)
	}
	if field.Nullable {
		g.schemaBuilder.SetNullable(schema, true)
	}
	if field.Constraints.Deprecated {
		t := true
		schema.Deprecated = &t
	}
	if field.ReadOnly {
		t := true
		schema.ReadOnly = &t
	}
	if field.WriteOnly {
		t := true
		schema.WriteOnly = &t
	}

	return base.CreateSchemaProxy(schema)
}

// bodySchema builds the schema for a request or response body: a wrapper
// envelope when bound, otherwise the plain body.
func (g *Generator) bodySchema(content *resolver.Content) *base.SchemaProxy {
	if content.Bind != nil && content.Bind.Wrapper != nil {
		return g.wrappedSchema(content)
	}
	return g.plainBodySchema(content)
}

// plainBodySchema builds the unwrapped body: a type reference for named
// bodies, an inline object for inline bodies.
func (g *Generator) plainBodySchema(content *resolver.Content) *base.SchemaProxy {
	if content.Ref != nil {
		return g.typeRefSchema(content.Ref)
	}
	return g.inlineObjectSchema(content.Fields)
}

// wrappedSchema builds the wrapper envelope with the body substituted for
// the bound field.
func (g *Generator) wrappedSchema(content *resolver.Content) *base.SchemaProxy {
	wrapper := content.Bind.Wrapper

	schema := g.schemaBuilder.NewSchema()
	g.schemaBuilder.SetType(schema, "object")

	if wrapper.Description != "" {
		schema.Description = wrapper.Description
	}

	props := orderedmap.New[string, *base.SchemaProxy]()
	var required []string

	for _, field := range wrapper.Fields {
		if field.GoName == content.Bind.Field {
			props.Set(field.Name, g.plainBodySchema(content))
		} else {
			props.Set(field.Name, g.fieldSchema(field))
		}
		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema.Properties = props
	if len(required) > 0 {
		schema.Required = required
	}

	return base.CreateSchemaProxy(schema)
}

// typeRefSchema builds the schema for a named @body type: "User", "[]User",
// "map[string]int", or a bare primitive.
func (g *Generator) typeRefSchema(ref *resolver.TypeRef) *base.SchemaProxy {
	element := func() *base.SchemaProxy {
		if ref.Primitive != "" {
			s := g.schemaBuilder.NewSchema()
			g.schemaBuilder.SetType(s, ref.Primitive)
			return base.CreateSchemaProxy(s)
		}
		return base.CreateSchemaProxyRef(componentRef(ref.Schema))
	}

	if ref.IsArray {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "array")
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{A: element()}
		return base.CreateSchemaProxy(schema)
	}

	if ref.IsMap {
		schema := g.schemaBuilder.NewSchema()
		g.schemaBuilder.SetType(schema, "object")
		schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{A: element()}
		return base.CreateSchemaProxy(schema)
	}

	return element()
}

// scalarNode builds the yaml node of a default or example value, tagged by the
// type the field emits so the emitter cannot retype it: a string field with a
// default of "true" stays the text true, not the boolean.
func scalarNode(value, openAPIType string) *yaml.Node {
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: value}
	switch openAPIType {
	case "string":
		node.Tag = "!!str"
	case "integer":
		node.Tag = "!!int"
	case "number":
		node.Tag = "!!float"
	case "boolean":
		node.Tag = "!!bool"
	}
	return node
}

// enumNodes converts enum values to yaml nodes, tagging them as integers
// when the governing type is "integer" so the emitter renders them unquoted.
func enumNodes(values []string, openAPIType string) []*yaml.Node {
	result := make([]*yaml.Node, len(values))
	if openAPIType == "integer" {
		for i, v := range values {
			result[i] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: v}
		}
	} else {
		for i, v := range values {
			result[i] = &yaml.Node{Kind: yaml.ScalarNode, Value: v}
		}
	}
	return result
}

func componentRef(name string) string {
	return fmt.Sprintf("#/components/schemas/%s", name)
}
