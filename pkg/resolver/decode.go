package resolver

// Schemas are directionless: a named @schema always describes the encoded
// (response) shape, whichever side of the wire references it. The one place
// specgen knows a struct is decoded rather than encoded is an in-function
// @request block — those fields exist only as a request body — so only they
// carry the decode rule below.

// applyDecodeRule replaces the marshal-rule Required/Nullable on an inline
// @request block's fields with what json.Unmarshal actually tolerates:
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
