package resolver

import (
	"fmt"
	"maps"
	"slices"
)

// inferDirections classifies every schema by which side of the wire can reach
// it: request-reachable, response-reachable, both, or neither. Direction is
// never declared — it is a fact about how endpoints use a schema, so it is
// derived by walking references from every endpoint body (named @body schemas,
// in-function @request/@response struct fields, and @bind wrappers) through
// schema fields.
//
// The classification decides which required/nullable rule a schema's fields
// carry: response-reachable schemas keep the marshal rule resolveField already
// computed, request-reachable schemas are rewritten to the decode rule here.
// Both-direction and unreachable schemas are the validator's to reject — this
// pass only records the evidence, one reference chain per direction on the
// Schema, and leaves such schemas' fields alone (they never reach generation).
//
// Runs only on a fully resolved package: Resolve calls it after its own error
// check, so every body and field the walk touches exists. References to
// schemas that were never declared are skipped — the validator reports those
// by name.
func inferDirections(pkg *Package) {
	request := newWalker(pkg.Schemas)
	response := newWalker(pkg.Schemas)

	// Endpoints are in parser order (sorted by function name), responses
	// sorted by status code, so the first chain that reaches a schema — the
	// one the validator prints — is the same on every run.
	for _, endpoint := range pkg.Endpoints {
		path := fmt.Sprintf("@endpoint[%s %s]", endpoint.Method, endpoint.Path)
		if endpoint.Request != nil {
			request.seed(path+".@request", endpoint.Request.Body, endpoint.Request.Inline)
		}
		for _, resp := range endpoint.Responses {
			response.seed(fmt.Sprintf("%s.@response[%s]", path, resp.StatusCode), resp.Body, resp.Inline)
		}
	}

	request.run()
	response.run()

	for _, name := range slices.Sorted(maps.Keys(pkg.Schemas)) {
		schema := pkg.Schemas[name]
		schema.RequestChain = request.chains[name]
		schema.ResponseChain = response.chains[name]
		if !schema.IsGeneric && schema.RequestChain != "" && schema.ResponseChain == "" {
			applyDecodeRule(schema.Fields)
		}
	}

	// In-function @request structs are request bodies by position, so their
	// own fields carry the decode rule too. Response inlines keep the marshal
	// rule resolveField gave them.
	for _, endpoint := range pkg.Endpoints {
		if endpoint.Request != nil && endpoint.Request.Inline != nil {
			applyDecodeRule(endpoint.Request.Inline.Fields)
		}
	}
}

// walker performs one direction's reachability walk over the schema graph.
type walker struct {
	schemas map[string]*Schema

	// chains records, for each reached schema, the reference chain that
	// reached it first. Set exactly once per schema, which makes it double as
	// the visited set — cycles stop here — and makes the reported chain
	// deterministic: seeds arrive in endpoint order and the queue is FIFO, so
	// the recorded chain is a shortest one from the earliest declaration.
	chains map[string]string
	queue  []string
}

func newWalker(schemas map[string]*Schema) *walker {
	return &walker{
		schemas: schemas,
		chains:  make(map[string]string),
	}
}

// seed enqueues every schema one body references directly: the named @body
// type (through any array/map layers), each in-function struct field, and the
// @bind wrapper from whichever form carries it.
func (w *walker) seed(path string, body *Body, inline *InlineBody) {
	var bind *BindTarget
	switch {
	case body != nil:
		bind = body.Bind
		refsIn(body.Type, func(ref string) {
			w.reach(ref, fmt.Sprintf("%s → %s", path, ref))
		})
	case inline != nil:
		bind = inline.Bind
		w.reachFields(path, inline.Fields)
	}

	if bind != nil {
		w.reach(bind.Wrapper, fmt.Sprintf("%s.@bind → %s", path, bind.Wrapper))
	}
}

// run walks the queue to a fixed point, following schema fields into the
// schemas they reference.
func (w *walker) run() {
	for len(w.queue) > 0 {
		name := w.queue[0]
		w.queue = w.queue[1:]
		w.reachFields(w.chains[name], w.schemas[name].Fields)
	}
}

// reachFields follows every schema reference inside fields, extending chain by
// one " → field <GoName> → <SchemaName>" hop per reference. A reference nested
// inside an anonymous object, array, or map is attributed to the top-level
// field that owns the type — chains stay flat.
func (w *walker) reachFields(chain string, fields []*Field) {
	for _, field := range fields {
		refsIn(field.Type, func(ref string) {
			w.reach(ref, fmt.Sprintf("%s → field %s → %s", chain, field.GoName, ref))
		})
	}
}

// reach records a schema as reachable and queues it for expansion, once.
// Names that resolve to no schema are skipped; the validator reports them.
func (w *walker) reach(name, chain string) {
	if _, known := w.schemas[name]; !known {
		return
	}
	if _, seen := w.chains[name]; seen {
		return
	}
	w.chains[name] = chain
	w.queue = append(w.queue, name)
}

// refsIn calls visit for every schema name a type references, in declaration
// order: the ref itself, then through array/map elements and anonymous struct
// fields.
func refsIn(t *TypeRef, visit func(string)) {
	if t == nil {
		return
	}
	if t.Shape == ShapeRef {
		visit(t.Ref)
	}
	refsIn(t.Elem, visit)
	for _, field := range t.Fields {
		refsIn(field.Type, visit)
	}
}

// applyDecodeRule replaces the marshal-rule Required/Nullable on
// request-direction fields with what json.Unmarshal actually tolerates:
// absence is fine exactly when the field is a pointer, and null never
// round-trips into a non-pointer. omitempty/omitzero say nothing about
// decoding, so they are ignored. @required/@nullable overrides still win, so
// they are re-applied from the raw facts resolveField recorded.
func applyDecodeRule(fields []*Field) {
	for _, field := range fields {
		field.Required = !field.pointer
		field.Nullable = false
		if field.overrideRequired != nil {
			field.Required = *field.overrideRequired
		}
		if field.overrideNullable != nil {
			field.Nullable = *field.overrideNullable
		}
		applyDecodeRuleType(field.Type)
	}
}

// applyDecodeRuleType carries the decode rule into fields nested inside
// anonymous objects and container elements.
func applyDecodeRuleType(t *TypeRef) {
	if t == nil {
		return
	}
	applyDecodeRuleType(t.Elem)
	applyDecodeRule(t.Fields)
}
