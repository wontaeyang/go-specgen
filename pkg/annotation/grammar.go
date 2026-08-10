package annotation

import (
	"maps"
	"slices"
)

// There are two grammars because the two contexts genuinely differ.
//
// Schema covers annotations written in doc comments, above a type or a
// function. Declaration covers annotations written inside a function body,
// above a var declaration. The difference that forces the split is @body: a
// doc-comment @response names its body, while an in-function @response is
// attached to a struct that already is the body, so @body has nothing to say.

// Schema is the grammar for doc-comment annotations.
var Schema = &Def{
	Name: "root",
	Kind: Block,
	Children: map[string]*Def{
		"@api": {
			Name:     "@api",
			Kind:     Block,
			Target:   OnPackage,
			Required: true,
			Children: map[string]*Def{
				"@title": {
					Name:     "@title",
					Kind:     Value,
					Required: true,
				},
				"@version": {
					Name:     "@version",
					Kind:     Value,
					Required: true,
				},
				"@description": {
					Name:              "@description",
					Kind:              Value,
					SupportsMultiline: true,
				},
				"@termsOfService": {
					Name: "@termsOfService",
					Kind: Value,
				},
				"@contact": {
					Name: "@contact",
					Kind: Block,
					Children: map[string]*Def{
						"@name": {
							Name: "@name",
							Kind: Value,
						},
						"@email": {
							Name: "@email",
							Kind: Value,
						},
						"@url": {
							Name: "@url",
							Kind: Value,
						},
					},
				},
				"@license": {
					Name: "@license",
					Kind: Block,
					Children: map[string]*Def{
						"@name": {
							Name: "@name",
							Kind: Value,
						},
						"@url": {
							Name: "@url",
							Kind: Value,
						},
					},
				},
				"@server": {
					Name:        "@server",
					Kind:        Block,
					HasMetadata: true,
					Repeatable:  true,
					Children: map[string]*Def{
						"@description": {
							Name:              "@description",
							Kind:              Value,
							SupportsMultiline: true,
						},
					},
				},
				"@securityScheme": {
					Name:        "@securityScheme",
					Kind:        Block,
					HasMetadata: true,
					Repeatable:  true,
					Children: map[string]*Def{
						"@type": {
							Name:     "@type",
							Kind:     Value,
							Required: true,
						},
						"@scheme": {
							Name: "@scheme",
							Kind: Value,
						},
						"@bearerFormat": {
							Name: "@bearerFormat",
							Kind: Value,
						},
						"@in": {
							Name: "@in",
							Kind: Value,
						},
						"@name": {
							Name: "@name",
							Kind: Value,
						},
						"@description": {
							Name:              "@description",
							Kind:              Value,
							SupportsMultiline: true,
						},
					},
				},
				"@security": {
					Name:       "@security",
					Kind:       Block,
					Repeatable: true,
					Children: map[string]*Def{
						"@with": {
							Name:        "@with",
							Kind:        SubCommand,
							HasMetadata: true,
							Repeatable:  true,
							Children: map[string]*Def{
								"@scope": {
									Name:       "@scope",
									Kind:       Value,
									Repeatable: true,
								},
							},
						},
					},
				},
				"@tag": {
					Name:        "@tag",
					Kind:        Block,
					HasMetadata: true,
					Repeatable:  true,
					Children: map[string]*Def{
						"@description": {
							Name:              "@description",
							Kind:              Value,
							SupportsMultiline: true,
						},
					},
				},
				"@defaultContentType": {
					Name: "@defaultContentType",
					Kind: Value,
				},
			},
		},
		"@endpoint": {
			Name:        "@endpoint",
			Kind:        Block,
			Target:      OnFunc,
			HasMetadata: true,
			Children: map[string]*Def{
				"@operationID": {
					Name: "@operationID",
					Kind: Value,
				},
				"@summary": {
					Name: "@summary",
					Kind: Value,
				},
				"@description": {
					Name:              "@description",
					Kind:              Value,
					SupportsMultiline: true,
				},
				"@tag": {
					Name:       "@tag",
					Kind:       Reference,
					Repeatable: true,
				},
				"@deprecated": {
					Name: "@deprecated",
					Kind: Flag,
				},
				"@auth": {
					Name: "@auth",
					Kind: Value,
				},
				"@path": {
					Name:       "@path",
					Kind:       Reference,
					Repeatable: true,
				},
				"@query": {
					Name:       "@query",
					Kind:       Reference,
					Repeatable: true,
				},
				"@header": {
					Name:       "@header",
					Kind:       Reference,
					Repeatable: true,
				},
				"@cookie": {
					Name:       "@cookie",
					Kind:       Reference,
					Repeatable: true,
				},
				"@request": {
					Name: "@request",
					Kind: Block,
					Children: map[string]*Def{
						"@contentType": {
							Name: "@contentType",
							Kind: Value,
						},
						"@body": {
							Name:        "@body",
							Kind:        Value,
							HasMetadata: true,
						},
						"@bind": {
							Name: "@bind",
							Kind: Value,
						},
					},
				},
				"@response": {
					Name:        "@response",
					Kind:        Block,
					HasMetadata: true,
					Repeatable:  true,
					Children: map[string]*Def{
						"@contentType": {
							Name: "@contentType",
							Kind: Value,
						},
						"@body": {
							Name:        "@body",
							Kind:        Value,
							HasMetadata: true,
						},
						"@bind": {
							Name: "@bind",
							Kind: Value,
						},
						"@description": {
							Name:              "@description",
							Kind:              Value,
							SupportsMultiline: true,
						},
						"@header": {
							Name:       "@header",
							Kind:       Value,
							Repeatable: true,
						},
					},
				},
			},
		},
		"@field": {
			Name:   "@field",
			Kind:   Block,
			Target: OnField,
			Children: map[string]*Def{
				"@description": {
					Name:              "@description",
					Kind:              Value,
					SupportsMultiline: true,
				},
				"@format": {
					Name: "@format",
					Kind: Value,
				},
				"@example": {
					Name: "@example",
					Kind: Value,
				},
				"@enum": {
					Name: "@enum",
					Kind: Value,
				},
				"@default": {
					Name: "@default",
					Kind: Value,
				},
				"@minimum": {
					Name: "@minimum",
					Kind: Value,
				},
				"@maximum": {
					Name: "@maximum",
					Kind: Value,
				},
				"@exclusiveMinimum": {
					Name: "@exclusiveMinimum",
					Kind: Value,
				},
				"@exclusiveMaximum": {
					Name: "@exclusiveMaximum",
					Kind: Value,
				},
				"@minLength": {
					Name: "@minLength",
					Kind: Value,
				},
				"@maxLength": {
					Name: "@maxLength",
					Kind: Value,
				},
				"@minItems": {
					Name: "@minItems",
					Kind: Value,
				},
				"@maxItems": {
					Name: "@maxItems",
					Kind: Value,
				},
				"@uniqueItems": {
					Name: "@uniqueItems",
					Kind: Flag,
				},
				"@pattern": {
					Name:     "@pattern",
					Kind:     Value,
					RawValue: true,
				},
				"@deprecated": {
					Name: "@deprecated",
					Kind: Flag,
				},
				"@readOnly": {
					Name: "@readOnly",
					Kind: Flag,
				},
				"@writeOnly": {
					Name: "@writeOnly",
					Kind: Flag,
				},
				"@required": {
					Name: "@required",
					Kind: Value,
				},
				"@nullable": {
					Name: "@nullable",
					Kind: Value,
				},
			},
		},
		"@schema": {
			Name:   "@schema",
			Kind:   Block,
			Target: OnType,
			Children: map[string]*Def{
				"@description": {
					Name:              "@description",
					Kind:              Value,
					SupportsMultiline: true,
				},
				"@deprecated": {
					Name: "@deprecated",
					Kind: Flag,
				},
			},
		},
		"@path":   {Name: "@path", Kind: Marker, Target: OnType},
		"@query":  {Name: "@query", Kind: Marker, Target: OnType},
		"@header": {Name: "@header", Kind: Marker, Target: OnType},
		"@cookie": {Name: "@cookie", Kind: Marker, Target: OnType},
	},
}

// Declaration is the grammar for annotations written inside a function body,
// above a var declaration. The struct being declared is the body, so @body is
// deliberately absent — naming one would contradict the struct it sits on.
var Declaration = &Def{
	Name: "root",
	Kind: Block,
	Children: map[string]*Def{
		"@request": {
			Name: "@request",
			Kind: Block,
			Children: map[string]*Def{
				"@contentType": {
					Name: "@contentType",
					Kind: Value,
				},
				"@description": {
					Name:              "@description",
					Kind:              Value,
					SupportsMultiline: true,
				},
				"@bind": {
					Name: "@bind",
					Kind: Value,
				},
			},
		},
		"@response": {
			Name:        "@response",
			Kind:        Block,
			HasMetadata: true, // status code
			Children: map[string]*Def{
				"@contentType": {
					Name: "@contentType",
					Kind: Value,
				},
				"@description": {
					Name:              "@description",
					Kind:              Value,
					SupportsMultiline: true,
				},
				"@header": {
					Name:       "@header",
					Kind:       Value,
					Repeatable: true,
				},
				"@bind": {
					Name: "@bind",
					Kind: Value,
				},
			},
		},
		"@path":   {Name: "@path", Kind: Marker},
		"@query":  {Name: "@query", Kind: Marker},
		"@header": {Name: "@header", Kind: Marker},
		"@cookie": {Name: "@cookie", Kind: Marker},
	},
}

func init() {
	Schema.InitializeParents()
	Declaration.InitializeParents()

	collectNames(Schema, knownNames)
	collectNames(Declaration, knownNames)
}

// knownNames is every annotation name either grammar defines, at any depth.
//
// It answers a question no single node can: whether a name means anything at
// all. A caller that finds @summary above a type knows it is not valid there
// from its own list of what is; this says whether the user wrote a real
// annotation in the wrong place or a name that exists nowhere, which is the
// difference between a mistake worth failing on and one worth reporting.
var knownNames = map[string]bool{}

// collectNames walks a grammar tree, recording every name in it.
func collectNames(node *Def, into map[string]bool) {
	for name, child := range node.Children {
		into[name] = true
		collectNames(child, into)
	}
}

// IsKnown reports whether name is an annotation either grammar defines, in any
// position. It says nothing about whether the name is legal where it was
// written.
func IsKnown(name string) bool { return knownNames[name] }

// WrittenOn is every annotation that may open the doc comment of a declaration
// of this kind, in name order.
//
// Callers used to hold this as a list of their own, one per pass, which is the
// arrangement pkg/annotation exists to prevent: adding an annotation is meant
// to be an edit to grammar.go and nothing else.
func WrittenOn(target Target) []string {
	var names []string
	for name, def := range Schema.Children {
		if def.Target == target {
			names = append(names, name)
		}
	}

	slices.Sort(names)
	return names
}

// InFunction is every annotation that may open the doc comment of a
// declaration inside a function body, in name order.
//
// It needs no Target: the Declaration grammar covers exactly that context, so
// being a child of its root is already the answer.
func InFunction() []string {
	names := slices.Collect(maps.Keys(Declaration.Children))

	slices.Sort(names)
	return names
}
