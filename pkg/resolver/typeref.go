package resolver

import (
	"fmt"
	"go/types"

	"github.com/wontaeyang/go-specgen/pkg/parser"
)

// resolveTypeRef describes a Go type as an emission-ready shape.
//
// This one traversal replaces four probes that each looked exactly one level
// deep: "is it an anonymous struct", "is it a slice of anonymous struct", "is
// it a map of anonymous struct", and a flat type mapper for everything else.
// Because it recurses, map[string][]User is describable at all — the flat form
// had one element-type slot and had to pick a level to remember.
//
// annotations are the @field annotations written on the fields of this type,
// when it is (or contains) an anonymous struct. They travel through containers
// unchanged, because an annotation inside `Items []struct{...}` describes a
// field of the element, not of the slice.
func (r *Resolver) resolveTypeRef(t types.Type, annotations ...[]*parser.Field) *TypeRef {
	var nested []*parser.Field
	if len(annotations) > 0 {
		nested = annotations[0]
	}

	switch typ := t.(type) {
	case *types.Pointer:
		// A pointer changes whether a field may be null, not what shape it has.
		return r.resolveTypeRef(typ.Elem(), nested)

	case *types.Alias:
		return r.resolveTypeRef(typ.Rhs(), nested)

	case *types.Named:
		return r.resolveNamedTypeRef(typ)

	case *types.Struct:
		// Reached only for an anonymous struct: a named one is *types.Named,
		// handled above, and becomes a $ref or an error.
		return &TypeRef{Shape: ShapeObject, Fields: r.resolveAnonymousFields(typ, nested)}

	case *types.Slice:
		// []byte is base64 text on the wire, not a JSON array. Deliberately
		// matched on *types.Basic rather than its underlying: a named byte type
		// is a distinct type and keeps array semantics.
		if basic, ok := typ.Elem().(*types.Basic); ok && basic.Kind() == types.Byte {
			return &TypeRef{Shape: ShapeScalar, Type: "string", Format: "byte"}
		}
		return &TypeRef{Shape: ShapeArray, Elem: r.resolveTypeRef(typ.Elem(), nested)}

	case *types.Array:
		return &TypeRef{Shape: ShapeArray, Elem: r.resolveTypeRef(typ.Elem(), nested)}

	case *types.Map:
		// OpenAPI keys are always strings; a non-string Go key still marshals
		// as one, so only the value type matters here.
		return &TypeRef{Shape: ShapeMap, Elem: r.resolveTypeRef(typ.Elem(), nested)}

	case *types.Interface:
		return &TypeRef{Shape: ShapeAny}

	case *types.TypeParam:
		return &TypeRef{Shape: ShapeTypeParam}

	case *types.Basic:
		return basicTypeRef(typ)
	}

	return &TypeRef{Shape: ShapeUnsupported, Reason: "an unsupported type (" + t.String() + ")"}
}

// resolveNamedTypeRef handles a named type: a special standard-library type, a
// @schema reference, an unrepresentable struct, or a name that is transparent
// to whatever it is defined as.
func (r *Resolver) resolveNamedTypeRef(named *types.Named) *TypeRef {
	obj := named.Obj()

	pkgPath := ""
	if obj.Pkg() != nil {
		pkgPath = obj.Pkg().Path()
	}

	// time.Time, url.URL and friends are structs that serialize as scalars.
	if special := resolveSpecialType(pkgPath, obj.Name()); special != nil {
		return &TypeRef{Shape: ShapeScalar, Type: special.openAPIType, Format: special.format}
	}

	if _, isStruct := named.Underlying().(*types.Struct); isStruct {
		if r.schemaNames[obj.Name()] {
			return &TypeRef{Shape: ShapeRef, Ref: obj.Name()}
		}
		return &TypeRef{
			Shape:  ShapeUnsupported,
			Ref:    obj.Name(),
			Reason: fmt.Sprintf("a struct (%s) with no @schema annotation", obj.Name()),
		}
	}

	// type Addresses []Address — the name adds nothing the spec can express.
	return r.resolveTypeRef(named.Underlying())
}

// basicTypeRef maps a Go basic type to its OpenAPI primitive.
//
// Unsigned integers deliberately carry no format: OpenAPI has no unsigned
// formats, and claiming int32/int64 would misstate the range.
func basicTypeRef(basic *types.Basic) *TypeRef {
	switch basic.Kind() {
	case types.Bool:
		return &TypeRef{Shape: ShapeScalar, Type: "boolean"}
	case types.Int, types.Int8, types.Int16:
		return &TypeRef{Shape: ShapeScalar, Type: "integer"}
	case types.Int32:
		return &TypeRef{Shape: ShapeScalar, Type: "integer", Format: "int32"}
	case types.Int64:
		return &TypeRef{Shape: ShapeScalar, Type: "integer", Format: "int64"}
	case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
		return &TypeRef{Shape: ShapeScalar, Type: "integer"}
	case types.Float32:
		return &TypeRef{Shape: ShapeScalar, Type: "number", Format: "float"}
	case types.Float64:
		return &TypeRef{Shape: ShapeScalar, Type: "number", Format: "double"}
	case types.String:
		return &TypeRef{Shape: ShapeScalar, Type: "string"}
	}

	// Complex numbers, uintptr and unsafe.Pointer are real Go types that
	// encoding/json refuses to marshal, so there is nothing truthful to emit.
	return &TypeRef{Shape: ShapeUnsupported, Reason: "an unsupported type (" + basic.Name() + ")"}
}

// isNullable reports whether a nil value of this type reaches the wire as null.
//
// Only pointers do. Slices and maps can be nil, and encoding/json does write
// them as null, but a handler that returns one is nearly always reporting an
// empty collection rather than a null — see the nullability note in the README.
// @nullable overrides either way.
func isNullable(t types.Type) bool {
	switch typ := t.(type) {
	case *types.Pointer:
		return true
	case *types.Alias:
		return isNullable(typ.Rhs())
	case *types.Named:
		return isNullable(typ.Underlying())
	}
	return false
}

// primitiveByName maps a Go type name written in an annotation to its OpenAPI
// primitive. Unlike basicTypeRef this works from a name rather than a
// *types.Basic, because @body text is not attached to a Go declaration.
func primitiveByName(name string) (primitive, format string, ok bool) {
	switch name {
	case "string":
		return "string", "", true
	case "bool":
		return "boolean", "", true
	case "int", "int8", "int16", "uint", "uint8", "uint16", "uint32", "uint64", "byte":
		return "integer", "", true
	case "int32":
		return "integer", "int32", true
	case "int64":
		return "integer", "int64", true
	case "float32":
		return "number", "float", true
	case "float64":
		return "number", "double", true
	}
	return "", "", false
}
