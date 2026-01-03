package schema

// AnnotationSchema is the complete annotation schema tree
// This is the single source of truth for all annotations
var AnnotationSchema = &SchemaNode{
	Name: "root",
	Type: BlockAnnotation,
	Children: map[string]*SchemaNode{
		"@api": {
			Name:     "@api",
			Type:     BlockAnnotation,
			Required: true,
			Children: map[string]*SchemaNode{
				"@title": {
					Name:     "@title",
					Type:     ValueAnnotation,
					Required: true,
				},
				"@version": {
					Name:     "@version",
					Type:     ValueAnnotation,
					Required: true,
				},
				"@description": {
					Name:              "@description",
					Type:              ValueAnnotation,
					SupportsMultiline: true,
				},
				"@termsOfService": {
					Name: "@termsOfService",
					Type: ValueAnnotation,
				},
				"@contact": {
					Name: "@contact",
					Type: BlockAnnotation,
					Children: map[string]*SchemaNode{
						"@name": {
							Name: "@name",
							Type: ValueAnnotation,
						},
						"@email": {
							Name: "@email",
							Type: ValueAnnotation,
						},
						"@url": {
							Name: "@url",
							Type: ValueAnnotation,
						},
					},
				},
				"@license": {
					Name: "@license",
					Type: BlockAnnotation,
					Children: map[string]*SchemaNode{
						"@name": {
							Name: "@name",
							Type: ValueAnnotation,
						},
						"@url": {
							Name: "@url",
							Type: ValueAnnotation,
						},
					},
				},
				"@server": {
					Name:        "@server",
					Type:        BlockAnnotation,
					HasMetadata: true,
					Repeatable:  true,
					Children: map[string]*SchemaNode{
						"@description": {
							Name:              "@description",
							Type:              ValueAnnotation,
							SupportsMultiline: true,
						},
					},
				},
				"@securityScheme": {
					Name:        "@securityScheme",
					Type:        BlockAnnotation,
					HasMetadata: true,
					Repeatable:  true,
					Children: map[string]*SchemaNode{
						"@type": {
							Name:     "@type",
							Type:     ValueAnnotation,
							Required: true,
						},
						"@scheme": {
							Name: "@scheme",
							Type: ValueAnnotation,
						},
						"@bearerFormat": {
							Name: "@bearerFormat",
							Type: ValueAnnotation,
						},
						"@in": {
							Name: "@in",
							Type: ValueAnnotation,
						},
						"@name": {
							Name: "@name",
							Type: ValueAnnotation,
						},
						"@description": {
							Name:              "@description",
							Type:              ValueAnnotation,
							SupportsMultiline: true,
						},
					},
				},
				"@security": {
					Name:       "@security",
					Type:       BlockAnnotation,
					Repeatable: true,
					Children: map[string]*SchemaNode{
						"@with": {
							Name:        "@with",
							Type:        SubCommand,
							HasMetadata: true,
							Repeatable:  true,
							Children: map[string]*SchemaNode{
								"@scope": {
									Name:       "@scope",
									Type:       ValueAnnotation,
									Repeatable: true,
								},
							},
						},
					},
				},
				"@tag": {
					Name:        "@tag",
					Type:        BlockAnnotation,
					HasMetadata: true,
					Repeatable:  true,
					Children: map[string]*SchemaNode{
						"@description": {
							Name:              "@description",
							Type:              ValueAnnotation,
							SupportsMultiline: true,
						},
					},
				},
				"@defaultContentType": {
					Name: "@defaultContentType",
					Type: ValueAnnotation,
				},
			},
		},
		"@endpoint": {
			Name:        "@endpoint",
			Type:        BlockAnnotation,
			HasMetadata: true,
			Children: map[string]*SchemaNode{
				"@operationID": {
					Name: "@operationID",
					Type: ValueAnnotation,
				},
				"@summary": {
					Name: "@summary",
					Type: ValueAnnotation,
				},
				"@description": {
					Name:              "@description",
					Type:              ValueAnnotation,
					SupportsMultiline: true,
				},
				"@tag": {
					Name:       "@tag",
					Type:       ReferenceAnnotation,
					Repeatable: true,
				},
				"@deprecated": {
					Name: "@deprecated",
					Type: FlagAnnotation,
				},
				"@auth": {
					Name: "@auth",
					Type: ValueAnnotation,
				},
				"@path": {
					Name:       "@path",
					Type:       ReferenceAnnotation,
					Repeatable: true,
				},
				"@query": {
					Name:       "@query",
					Type:       ReferenceAnnotation,
					Repeatable: true,
				},
				"@header": {
					Name:       "@header",
					Type:       ReferenceAnnotation,
					Repeatable: true,
				},
				"@cookie": {
					Name:       "@cookie",
					Type:       ReferenceAnnotation,
					Repeatable: true,
				},
				"@request": {
					Name: "@request",
					Type: BlockAnnotation,
					Children: map[string]*SchemaNode{
						"@contentType": {
							Name: "@contentType",
							Type: ValueAnnotation,
						},
						"@body": {
							Name:        "@body",
							Type:        ValueAnnotation,
							HasMetadata: true,
						},
						"@bind": {
							Name: "@bind",
							Type: ValueAnnotation,
						},
					},
				},
				"@response": {
					Name:        "@response",
					Type:        BlockAnnotation,
					HasMetadata: true,
					Repeatable:  true,
					Children: map[string]*SchemaNode{
						"@contentType": {
							Name: "@contentType",
							Type: ValueAnnotation,
						},
						"@body": {
							Name:        "@body",
							Type:        ValueAnnotation,
							HasMetadata: true,
						},
						"@bind": {
							Name: "@bind",
							Type: ValueAnnotation,
						},
						"@description": {
							Name:              "@description",
							Type:              ValueAnnotation,
							SupportsMultiline: true,
						},
						"@header": {
							Name:       "@header",
							Type:       ValueAnnotation,
							Repeatable: true,
						},
					},
				},
			},
		},
		"@field": {
			Name: "@field",
			Type: BlockAnnotation,
			Children: map[string]*SchemaNode{
				"@description": {
					Name:              "@description",
					Type:              ValueAnnotation,
					SupportsMultiline: true,
				},
				"@format": {
					Name: "@format",
					Type: ValueAnnotation,
				},
				"@example": {
					Name: "@example",
					Type: ValueAnnotation,
				},
				"@enum": {
					Name: "@enum",
					Type: ValueAnnotation,
				},
				"@default": {
					Name: "@default",
					Type: ValueAnnotation,
				},
				"@minimum": {
					Name: "@minimum",
					Type: ValueAnnotation,
				},
				"@maximum": {
					Name: "@maximum",
					Type: ValueAnnotation,
				},
				"@minLength": {
					Name: "@minLength",
					Type: ValueAnnotation,
				},
				"@maxLength": {
					Name: "@maxLength",
					Type: ValueAnnotation,
				},
				"@minItems": {
					Name: "@minItems",
					Type: ValueAnnotation,
				},
				"@maxItems": {
					Name: "@maxItems",
					Type: ValueAnnotation,
				},
				"@uniqueItems": {
					Name: "@uniqueItems",
					Type: FlagAnnotation,
				},
				"@pattern": {
					Name: "@pattern",
					Type: ValueAnnotation,
				},
				"@deprecated": {
					Name: "@deprecated",
					Type: FlagAnnotation,
				},
			},
		},
		"@schema": {
			Name: "@schema",
			Type: BlockAnnotation,
			Children: map[string]*SchemaNode{
				"@description": {
					Name:              "@description",
					Type:              ValueAnnotation,
					SupportsMultiline: true,
				},
				"@deprecated": {
					Name: "@deprecated",
					Type: FlagAnnotation,
				},
			},
		},
		"@path": {
			Name: "@path",
			Type: MarkerAnnotation,
		},
		"@query": {
			Name: "@query",
			Type: MarkerAnnotation,
		},
		"@header": {
			Name: "@header",
			Type: MarkerAnnotation,
		},
		"@cookie": {
			Name: "@cookie",
			Type: MarkerAnnotation,
		},
	},
}

func init() {
	// Initialize parent references in the schema tree
	AnnotationSchema.InitializeParents()
}
