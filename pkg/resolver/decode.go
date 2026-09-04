package resolver

// Schemas are directionless: a named @schema always describes the encoded
// (response) shape, whichever side of the wire references it. The one place
// specgen knows a struct is decoded rather than encoded is an in-function
// @request block — those fields exist only as a request body — so only they
// carry the decode rule below.

// applyDecodeRule replaces the marshal-rule Required/Nullable on an inline
// @request block's fields with the decode rule.
//
// json.Unmarshal itself never rejects an absent field and never rejects null:
// null sets a pointer, slice or map to nil and is silently ignored on any
// other type. So the rule is about what the handler can observe, not what
// decoding tolerates. A pointer is the only way to tell an absent field apart
// from its zero value, so a pointer field is optional and everything else is
// required. Nothing is nullable: null is indistinguishable from absence on a
// pointer and invisible elsewhere, so accepting it would promise something the
// handler cannot see. omitempty/omitzero only affect encoding and are ignored.
// @required/@nullable overrides still win, so they are applied again on top.
func applyDecodeRule(fields []*Field) {
	for _, field := range fields {
		field.Required = !field.pointer
		field.Nullable = false
		overrideRequiredNullable(field, field.annotation)
		applyDecodeRuleType(field.Type)
	}
}

// applyDecodeRuleType carries the decode rule into fields nested inside
// anonymous objects and container elements. A $ref carries no Fields, so a
// named schema referenced from an inline request keeps the marshal rule.
func applyDecodeRuleType(t *TypeRef) {
	if t == nil {
		return
	}
	applyDecodeRuleType(t.Elem)
	applyDecodeRule(t.Fields)
}
