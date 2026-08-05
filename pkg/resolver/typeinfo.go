package resolver

import (
	"go/types"
	"reflect"
	"strings"
)

// This file turns Go types into the shapes the generator emits. It is the one
// place that knows the primitive table, the standard library types that
// serialize as scalars, and how struct tags name and qualify a field.

// specialType is the scalar shape a standard library struct serializes as.
type specialType struct {
	openAPI string
	format  string
}

// specialTypes are standard library structs that serialize as scalars. They
// resolve as primitives instead of being walked as structs, and they never
// count as unresolved schema references.
var specialTypes = map[string]map[string]specialType{
	"time": {
		"Time": {openAPI: "string", format: "date-time"},
	},
	"net/url": {
		"URL": {openAPI: "string", format: "uri"},
	},
	"net/netip": {
		"Addr":     {openAPI: "string"},
		"AddrPort": {openAPI: "string"},
		"Prefix":   {openAPI: "string"},
	},
	"math/big": {
		"Int":   {openAPI: "string"},
		"Float": {openAPI: "string"},
		"Rat":   {openAPI: "string"},
	},
	"regexp": {
		"Regexp": {openAPI: "string"},
	},
}

// specialFor returns the scalar mapping of a named standard library type.
func specialFor(named *types.Named) (specialType, bool) {
	obj := named.Obj()
	if obj.Pkg() == nil {
		return specialType{}, false
	}
	byName, ok := specialTypes[obj.Pkg().Path()]
	if !ok {
		return specialType{}, false
	}
	special, ok := byName[obj.Name()]
	return special, ok
}

// typeInfo resolves a Go type into the shape the generator emits, the format
// that goes with it, and whether it can carry null. Types that name a @schema
// resolve to references; everything else resolves structurally.
func (r *resolver) typeInfo(t types.Type) (info TypeInfo, format string, nullable bool) {
	if ptr, ok := t.(*types.Pointer); ok {
		nullable = true
		t = ptr.Elem()
	}

	if ref := r.schemaRef(t); ref != "" {
		return TypeInfo{Ref: ref}, "", nullable
	}

	switch u := t.(type) {
	case *types.Slice:
		if isByteSlice(u) {
			// encoding/json writes []byte as base64 text, so it is a string
			// with the byte format, not an array of anything.
			return TypeInfo{OpenAPI: "string"}, "byte", nullable
		}
		return r.arrayInfo(u.Elem()), "", nullable

	case *types.Array:
		return r.arrayInfo(u.Elem()), "", nullable

	case *types.Map:
		return r.mapInfo(u.Elem()), "", nullable

	case *types.Alias:
		info, format, _ = r.typeInfo(u.Rhs())
		return info, format, nullable

	case *types.Named:
		if special, ok := specialFor(u); ok {
			return TypeInfo{OpenAPI: special.openAPI}, special.format, nullable
		}
		info, format, _ = r.typeInfo(u.Underlying())
		return info, format, nullable

	case *types.Basic:
		openAPI, format := basicType(u)
		return TypeInfo{OpenAPI: openAPI}, format, nullable

	case *types.Interface:
		return TypeInfo{IsAny: true}, "", nullable
	}

	// Anything left (a bare struct, a type parameter, a channel) has no
	// OpenAPI shape of its own; "string" is the long-standing fallback.
	return TypeInfo{OpenAPI: "string"}, "", nullable
}

// arrayInfo describes a slice or array with the given element type. Items
// keeps the element's scalar type even when ItemsRef is set, because parameter
// schemas ignore references and emit the scalar instead.
func (r *resolver) arrayInfo(elem types.Type) TypeInfo {
	openAPI, format := scalarShape(elem)
	return TypeInfo{
		IsArray:     true,
		Items:       openAPI,
		ItemsFormat: format,
		ItemsRef:    r.schemaRef(elem),
	}
}

// mapInfo describes a map with the given value type. A value that collapses to
// nothing usable falls back to "string".
func (r *resolver) mapInfo(value types.Type) TypeInfo {
	if ref := r.schemaRef(value); ref != "" {
		return TypeInfo{IsMap: true, MapValueRef: ref}
	}

	openAPI, format := scalarShape(value)
	if openAPI == "" {
		openAPI = "string"
	}
	return TypeInfo{IsMap: true, MapValue: openAPI, MapValueFormat: format}
}

// schemaRef returns the name of the @schema type t refers to, or "" when it
// refers to none. Generic templates are not referenceable: they are never
// emitted to components, so a field typed with one resolves structurally.
func (r *resolver) schemaRef(t types.Type) string {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	var name string
	switch u := t.(type) {
	case *types.Named:
		name = u.Obj().Name()
	case *types.Alias:
		name = u.Obj().Name()
	default:
		return ""
	}

	if schema, ok := r.schemas[name]; ok && !schema.IsGeneric {
		return name
	}
	return ""
}

// scalarShape is the OpenAPI type and format a Go type collapses to where only
// a scalar fits: array items and map values. Structs collapse to "string", the
// long-standing fallback for anything that is not a primitive.
func scalarShape(t types.Type) (openAPI, format string) {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	switch u := t.(type) {
	case *types.Slice:
		if isByteSlice(u) {
			return "string", "byte"
		}
		return "array", ""
	case *types.Array:
		return "array", ""
	case *types.Alias:
		return scalarShape(u.Rhs())
	case *types.Named:
		if special, ok := specialFor(u); ok {
			return special.openAPI, special.format
		}
		return scalarShape(u.Underlying())
	case *types.Basic:
		return basicType(u)
	case *types.Interface:
		return "", ""
	}
	return "string", ""
}

// basicType maps a Go basic type to its OpenAPI type and format. Unsigned
// integers get no format: OpenAPI has no unsigned formats.
func basicType(b *types.Basic) (openAPI, format string) {
	switch b.Kind() {
	case types.Bool:
		return "boolean", ""
	case types.Int, types.Int8, types.Int16:
		return "integer", ""
	case types.Int32:
		return "integer", "int32"
	case types.Int64:
		return "integer", "int64"
	case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
		return "integer", ""
	case types.Float32:
		return "number", "float"
	case types.Float64:
		return "number", "double"
	case types.String:
		return "string", ""
	}
	return "string", ""
}

// isByteSlice reports whether t is []byte, which serializes as base64 text
// rather than as an array of numbers. Fixed-size [N]byte arrays do not:
// encoding/json writes those as arrays.
func isByteSlice(t types.Type) bool {
	slice, ok := t.(*types.Slice)
	if !ok {
		return false
	}
	basic, ok := slice.Elem().(*types.Basic)
	return ok && basic.Kind() == types.Byte
}

// unresolvedStruct returns the name of the first named struct reachable
// through t that carries no @schema annotation, or "" when there is none.
// Such a struct has no components entry to reference, so the validator reports
// it instead of letting it silently emit as a string.
func (r *resolver) unresolvedStruct(t types.Type) string {
	switch u := t.(type) {
	case *types.Pointer:
		return r.unresolvedStruct(u.Elem())
	case *types.Slice:
		return r.unresolvedStruct(u.Elem())
	case *types.Array:
		return r.unresolvedStruct(u.Elem())
	case *types.Map:
		return r.unresolvedStruct(u.Elem())
	case *types.Named:
		if _, ok := specialFor(u); ok {
			return ""
		}
		if _, isStruct := u.Underlying().(*types.Struct); isStruct {
			if _, known := r.schemas[u.Obj().Name()]; known {
				return ""
			}
			return u.Obj().Name()
		}
		// A named slice or map (type Addresses []Address) still reaches a
		// struct through its underlying type.
		return r.unresolvedStruct(u.Underlying())
	}
	return ""
}

// anonymousStruct returns the anonymous struct t is, unwrapping one pointer.
// Named structs are not anonymous: they are referenced, not inlined.
func anonymousStruct(t types.Type) *types.Struct {
	if t == nil {
		return nil
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	st, _ := t.(*types.Struct)
	return st
}

// sliceElem returns the element type of a slice, unwrapping one pointer, or
// nil when t is not a slice.
func sliceElem(t types.Type) types.Type {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if slice, ok := t.(*types.Slice); ok {
		return slice.Elem()
	}
	return nil
}

// mapElem returns the value type of a map, unwrapping one pointer, or nil when
// t is not a map.
func mapElem(t types.Type) types.Type {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if m, ok := t.(*types.Map); ok {
		return m.Elem()
	}
	return nil
}

// SupportedTags are the struct tags consulted for a body field's wire name, in
// fallback order. Append to it to support another tag.
var SupportedTags = []string{"json", "xml"}

// bodyFieldName resolves the wire name of a body field: the first supported
// tag that names it, otherwise the Go field name. A tag of "-" skips the field
// entirely, reported as an empty name.
func bodyFieldName(tag reflect.StructTag, goName string) string {
	for _, key := range SupportedTags {
		value := tag.Get(key)
		if value == "" {
			continue
		}
		if value == "-" {
			return ""
		}
		name, _, _ := strings.Cut(value, ",")
		if name == "" {
			// A tag that carries only options (json:",omitempty") names
			// nothing, so the fallback chain continues.
			continue
		}
		return name
	}
	return goName
}

// paramFieldName resolves the wire name of a parameter field from the tag of
// its own kind. Unlike a body field a parameter is never skipped: a missing
// tag, and even "-", falls back to the Go field name.
func paramFieldName(tag reflect.StructTag, kind, goName string) string {
	name, _, _ := strings.Cut(tag.Get(kind), ",")
	if name == "" || name == "-" {
		return goName
	}
	return name
}

// hasTagOption reports whether the tag of the given key carries an option,
// e.g. `query:"limit,required"`.
func hasTagOption(tag reflect.StructTag, key, option string) bool {
	_, options, _ := strings.Cut(tag.Get(key), ",")
	for _, opt := range strings.Split(options, ",") {
		if opt == option {
			return true
		}
	}
	return false
}

// omitsWhenEmpty reports whether the json tag actually drops the field for
// empty or zero values of its type. omitzero omits any zero value, including
// zero structs. omitempty follows encoding/json's isEmptyValue, which never
// considers a struct empty (and an array only at length zero), so such fields
// stay on the wire despite the tag.
func omitsWhenEmpty(tag reflect.StructTag, fieldType types.Type) bool {
	if hasTagOption(tag, "json", "omitzero") {
		return true
	}
	return hasTagOption(tag, "json", "omitempty") && canBeEmpty(fieldType)
}

// canBeEmpty mirrors encoding/json's isEmptyValue: it reports whether any
// value of the type is ever considered empty by omitempty.
func canBeEmpty(fieldType types.Type) bool {
	switch u := fieldType.Underlying().(type) {
	case *types.Pointer, *types.Interface, *types.Map, *types.Slice:
		return true
	case *types.Basic:
		return u.Info()&(types.IsBoolean|types.IsNumeric|types.IsString) != 0
	case *types.Array:
		return u.Len() == 0
	default:
		return false
	}
}
