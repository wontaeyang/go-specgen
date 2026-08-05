package parser

// This file is the annotation grammar: the single source of truth for which
// annotations exist, where they may appear, and how their text is read.
//
// There are two grammars, because there are honestly two contexts:
//
//   - Grammar covers declaration comments — the annotations attached to the
//     package clause, a type declaration or a function declaration.
//   - InlineGrammar covers in-function declarations, where the annotated
//     struct IS the body, so there is no @body child to name one.
//
// They share child nodes wherever the child is genuinely the same annotation.

// AnnotationType says how an annotation's source text is read.
type AnnotationType int

const (
	// BlockAnnotation has a { } block containing child annotations.
	BlockAnnotation AnnotationType = iota

	// ValueAnnotation has a simple value: @title My API.
	ValueAnnotation

	// FlagAnnotation is a bare marker with no value: @deprecated, @path.
	FlagAnnotation

	// ReferenceAnnotation names another annotated declaration: @path UserPath.
	// It is read exactly like a ValueAnnotation; the separate type documents
	// that the value is resolved against other declarations.
	ReferenceAnnotation
)

// GrammarNode is one annotation in the grammar tree.
type GrammarNode struct {
	// Name is the annotation name, including the @.
	Name string

	Type AnnotationType

	// Required marks a child that must be present for its parent block to be
	// valid (see CanBeEmpty).
	Required bool

	// HasMetadata marks annotations whose opening line carries a value before
	// the block: "@endpoint GET /users" or "@securityScheme bearerAuth {".
	HasMetadata bool

	// Repeatable marks annotations that may appear more than once within
	// their parent.
	Repeatable bool

	// SupportsMultiline marks annotations whose value continues on the
	// following comment lines until the next sibling annotation.
	SupportsMultiline bool

	// RawValue marks values that pass through verbatim: no escape validation
	// and no unescaping. Used where the value is itself a DSL with its own
	// escape grammar (@pattern).
	RawValue bool

	Children map[string]*GrammarNode
}

// GetChild returns a child node by name, or nil if there is none.
func (n *GrammarNode) GetChild(name string) *GrammarNode {
	if n == nil {
		return nil
	}
	return n.Children[name]
}

// HasChild reports whether a child with the given name exists.
func (n *GrammarNode) HasChild(name string) bool {
	return n.GetChild(name) != nil
}

// CanBeEmpty reports whether the block may be written with no content. A block
// can be empty when none of its children are required.
func (n *GrammarNode) CanBeEmpty() bool {
	for _, child := range n.Children {
		if child.Required {
			return false
		}
	}
	return true
}

// Shared child nodes. Nodes carry no parent pointer, so the same node can be
// referenced from several parents — and from both grammars — without aliasing.
var (
	descriptionChild = &GrammarNode{
		Name:              "@description",
		Type:              ValueAnnotation,
		SupportsMultiline: true,
	}

	contentTypeChild = &GrammarNode{
		Name: "@contentType",
		Type: ValueAnnotation,
	}

	bindChild = &GrammarNode{
		Name: "@bind",
		Type: ValueAnnotation,
	}

	// headerRefChild names a @header parameter struct. It is repeatable both
	// on an endpoint (request headers) and on a response (response headers).
	headerRefChild = &GrammarNode{
		Name:       "@header",
		Type:       ReferenceAnnotation,
		Repeatable: true,
	}

	// bodyChild names the schema a request or response body carries:
	// "@body User", "@body []User", "@body map[string]User".
	bodyChild = &GrammarNode{
		Name: "@body",
		Type: ValueAnnotation,
	}
)

// ResponseNode is the @response block that names its body with @body. It is
// the same annotation in both places it may be written: in an @endpoint block,
// and on its own inside a handler body, where no struct declaration follows to
// serve as the body.
var ResponseNode = &GrammarNode{
	Name:        "@response",
	Type:        BlockAnnotation,
	HasMetadata: true,
	Repeatable:  true,
	Children: map[string]*GrammarNode{
		"@contentType": contentTypeChild,
		"@body":        bodyChild,
		"@bind":        bindChild,
		"@description": descriptionChild,
		"@header":      headerRefChild,
	},
}

// apiNode is the @api block: document-level metadata, declared on the package.
var apiNode = &GrammarNode{
	Name: "@api",
	Type: BlockAnnotation,
	Children: map[string]*GrammarNode{
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
		"@description": descriptionChild,
		"@termsOfService": {
			Name: "@termsOfService",
			Type: ValueAnnotation,
		},
		"@contact": {
			Name: "@contact",
			Type: BlockAnnotation,
			Children: map[string]*GrammarNode{
				"@name":  {Name: "@name", Type: ValueAnnotation},
				"@email": {Name: "@email", Type: ValueAnnotation},
				"@url":   {Name: "@url", Type: ValueAnnotation},
			},
		},
		"@license": {
			Name: "@license",
			Type: BlockAnnotation,
			Children: map[string]*GrammarNode{
				"@name": {Name: "@name", Type: ValueAnnotation},
				"@url":  {Name: "@url", Type: ValueAnnotation},
			},
		},
		"@server": {
			Name:        "@server",
			Type:        BlockAnnotation,
			HasMetadata: true,
			Repeatable:  true,
			Children: map[string]*GrammarNode{
				"@description": descriptionChild,
			},
		},
		"@securityScheme": {
			Name:        "@securityScheme",
			Type:        BlockAnnotation,
			HasMetadata: true,
			Repeatable:  true,
			Children: map[string]*GrammarNode{
				"@type": {
					Name:     "@type",
					Type:     ValueAnnotation,
					Required: true,
				},
				"@scheme":       {Name: "@scheme", Type: ValueAnnotation},
				"@bearerFormat": {Name: "@bearerFormat", Type: ValueAnnotation},
				"@in":           {Name: "@in", Type: ValueAnnotation},
				"@name":         {Name: "@name", Type: ValueAnnotation},
				"@description":  descriptionChild,
			},
		},
		"@security": {
			Name:       "@security",
			Type:       BlockAnnotation,
			Repeatable: true,
			Children: map[string]*GrammarNode{
				"@with": {
					Name:        "@with",
					Type:        BlockAnnotation,
					HasMetadata: true,
					Repeatable:  true,
					Children: map[string]*GrammarNode{
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
			Children: map[string]*GrammarNode{
				"@description": descriptionChild,
			},
		},
		"@defaultContentType": {
			Name: "@defaultContentType",
			Type: ValueAnnotation,
		},
	},
}

// endpointNode is the @endpoint block, declared on a handler function.
var endpointNode = &GrammarNode{
	Name:        "@endpoint",
	Type:        BlockAnnotation,
	HasMetadata: true,
	Children: map[string]*GrammarNode{
		"@operationID": {Name: "@operationID", Type: ValueAnnotation},
		"@summary":     {Name: "@summary", Type: ValueAnnotation},
		"@description": descriptionChild,
		"@tag": {
			Name:       "@tag",
			Type:       ReferenceAnnotation,
			Repeatable: true,
		},
		"@deprecated": {Name: "@deprecated", Type: FlagAnnotation},
		"@auth":       {Name: "@auth", Type: ValueAnnotation},
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
		"@header": headerRefChild,
		"@cookie": {
			Name:       "@cookie",
			Type:       ReferenceAnnotation,
			Repeatable: true,
		},
		"@request": {
			Name: "@request",
			Type: BlockAnnotation,
			Children: map[string]*GrammarNode{
				"@contentType": contentTypeChild,
				"@body":        bodyChild,
				"@bind":        bindChild,
			},
		},
		"@response": ResponseNode,
	},
}

// fieldNode is the @field block, declared on a struct field.
var fieldNode = &GrammarNode{
	Name: "@field",
	Type: BlockAnnotation,
	Children: map[string]*GrammarNode{
		"@description":      descriptionChild,
		"@format":           {Name: "@format", Type: ValueAnnotation},
		"@example":          {Name: "@example", Type: ValueAnnotation},
		"@enum":             {Name: "@enum", Type: ValueAnnotation},
		"@default":          {Name: "@default", Type: ValueAnnotation},
		"@minimum":          {Name: "@minimum", Type: ValueAnnotation},
		"@maximum":          {Name: "@maximum", Type: ValueAnnotation},
		"@exclusiveMinimum": {Name: "@exclusiveMinimum", Type: ValueAnnotation},
		"@exclusiveMaximum": {Name: "@exclusiveMaximum", Type: ValueAnnotation},
		"@minLength":        {Name: "@minLength", Type: ValueAnnotation},
		"@maxLength":        {Name: "@maxLength", Type: ValueAnnotation},
		"@minItems":         {Name: "@minItems", Type: ValueAnnotation},
		"@maxItems":         {Name: "@maxItems", Type: ValueAnnotation},
		"@uniqueItems":      {Name: "@uniqueItems", Type: FlagAnnotation},
		"@pattern": {
			Name:     "@pattern",
			Type:     ValueAnnotation,
			RawValue: true,
		},
		"@deprecated": {Name: "@deprecated", Type: FlagAnnotation},
		"@readOnly":   {Name: "@readOnly", Type: FlagAnnotation},
		"@writeOnly":  {Name: "@writeOnly", Type: FlagAnnotation},
		"@required":   {Name: "@required", Type: ValueAnnotation},
		"@nullable":   {Name: "@nullable", Type: ValueAnnotation},
	},
}

// schemaNode is the @schema block, declared on a struct type.
var schemaNode = &GrammarNode{
	Name: "@schema",
	Type: BlockAnnotation,
	Children: map[string]*GrammarNode{
		"@description": descriptionChild,
		"@deprecated":  {Name: "@deprecated", Type: FlagAnnotation},
	},
}

// Grammar is the grammar for declaration comments: every annotation that may
// open a comment on a package clause, a type declaration or a function.
//
// The parameter markers carry no value — the struct's fields are the
// parameters — so they are flags.
var Grammar = map[string]*GrammarNode{
	"@api":      apiNode,
	"@endpoint": endpointNode,
	"@field":    fieldNode,
	"@schema":   schemaNode,
	"@path":     {Name: "@path", Type: FlagAnnotation},
	"@query":    {Name: "@query", Type: FlagAnnotation},
	"@header":   {Name: "@header", Type: FlagAnnotation},
	"@cookie":   {Name: "@cookie", Type: FlagAnnotation},
}

// InlineGrammar is the grammar for in-function declarations: a var or type
// declaration inside a handler body, annotated to become a parameter group, a
// request body or a response body.
//
// The struct being declared IS the body here, which is why @request and
// @response have no @body child — naming a second body is an error, not a
// choice. Parameter markers take no options at all.
//
// An @response written on its own, with no declaration under it, is parsed
// against ResponseNode instead: with no struct to be the body, it has to name
// one.
var InlineGrammar = map[string]*GrammarNode{
	"@path":   {Name: "@path", Type: FlagAnnotation},
	"@query":  {Name: "@query", Type: FlagAnnotation},
	"@header": {Name: "@header", Type: FlagAnnotation},
	"@cookie": {Name: "@cookie", Type: FlagAnnotation},
	"@request": {
		Name: "@request",
		Type: BlockAnnotation,
		Children: map[string]*GrammarNode{
			"@contentType": contentTypeChild,
			"@description": descriptionChild,
			"@bind":        bindChild,
		},
	},
	"@response": {
		Name:        "@response",
		Type:        BlockAnnotation,
		HasMetadata: true, // status code
		Children: map[string]*GrammarNode{
			"@contentType": contentTypeChild,
			"@description": descriptionChild,
			"@header":      headerRefChild,
			"@bind":        bindChild,
		},
	},
}
