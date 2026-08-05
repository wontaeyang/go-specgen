package resolver

import (
	"go/types"
	"reflect"
	"testing"
)

// The helpers below are unit-tested against synthetic types and tags, because
// the combinations they cover are cheaper to write here than as fixture
// packages. The type mapping itself is covered end to end in resolver_test.go.

var (
	stringType = types.Typ[types.String]
	structType = types.NewStruct(nil, nil)
)

func TestOmitsWhenEmpty(t *testing.T) {
	tests := []struct {
		name      string
		tag       string
		fieldType types.Type
		want      bool
	}{
		{"no tag", ``, stringType, false},
		{"named only", `json:"value"`, stringType, false},
		{"omitempty on a string", `json:"value,omitempty"`, stringType, true},
		{"omitzero on a string", `json:"value,omitzero"`, stringType, true},
		// encoding/json never considers a struct empty, so omitempty cannot
		// drop one; omitzero drops any zero value, structs included.
		{"omitempty on a struct", `json:"value,omitempty"`, structType, false},
		{"omitzero on a struct", `json:"value,omitzero"`, structType, true},
		{"omitempty on a pointer", `json:"value,omitempty"`, types.NewPointer(stringType), true},
		{"omitempty on a slice", `json:"value,omitempty"`, types.NewSlice(stringType), true},
		{"omitempty on an empty array", `json:"value,omitempty"`, types.NewArray(stringType, 0), true},
		{"omitempty on a sized array", `json:"value,omitempty"`, types.NewArray(stringType, 3), false},
		// Only the json tag decides: another tag that happens to spell the
		// option must not leak into the decision.
		{"omitempty in another tag", `yaml:"value,omitempty"`, stringType, false},
		{"option-shaped value elsewhere", `json:"value" validate:"required,omitempty"`, stringType, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := omitsWhenEmpty(reflect.StructTag(tt.tag), tt.fieldType); got != tt.want {
				t.Errorf("omitsWhenEmpty(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestBodyFieldName(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want string
	}{
		{"json tag", `json:"user_id"`, "user_id"},
		{"json tag with options", `json:"user_id,omitempty"`, "user_id"},
		{"json tag with several options", `json:"user_id,omitempty,string"`, "user_id"},
		{"xml fallback", `xml:"UserName"`, "UserName"},
		{"json wins over xml", `json:"email" xml:"EmailAddress"`, "email"},
		{"no tags", ``, "GoName"},
		{"unsupported tag only", `form:"value"`, "GoName"},
		{"skipped", `json:"-"`, ""},
		{"skipped despite xml", `json:"-" xml:"WontBeUsed"`, ""},
		{"skipped by xml", `xml:"-"`, ""},
		// `json:"-,"` is how encoding/json spells a field literally named "-".
		{"literal dash", `json:"-,"`, "-"},
		{"empty json falls through to xml", `json:"" xml:"from_xml"`, "from_xml"},
		{"options-only json falls through to xml", `json:",omitempty" xml:"from_xml"`, "from_xml"},
		{"options-only json falls back to the Go name", `json:",omitempty"`, "GoName"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bodyFieldName(reflect.StructTag(tt.tag), "GoName"); got != tt.want {
				t.Errorf("bodyFieldName(%q) = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}

func TestParamFieldName(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		kind string
		want string
	}{
		{"named", `query:"limit"`, "query", "limit"},
		{"named with options", `query:"limit,required"`, "query", "limit"},
		{"another kind's tag is ignored", `path:"id"`, "query", "GoName"},
		{"no tag", ``, "query", "GoName"},
		// Unlike a body field, a parameter is never skipped.
		{"dash falls back", `query:"-"`, "query", "GoName"},
		{"path kind", `path:"id"`, "path", "id"},
		{"header kind", `header:"X-Request-ID"`, "header", "X-Request-ID"},
		{"cookie kind", `cookie:"session"`, "cookie", "session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := paramFieldName(reflect.StructTag(tt.tag), tt.kind, "GoName"); got != tt.want {
				t.Errorf("paramFieldName(%q, %q) = %q, want %q", tt.tag, tt.kind, got, tt.want)
			}
		})
	}
}

func TestHasTagOption(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{`query:"q,required"`, true},
		{`query:"q,omitempty,required"`, true},
		{`query:"q"`, false},
		{``, false},
		// "required" as the name, not as an option.
		{`query:"required"`, false},
		// The option belongs to another tag.
		{`json:"q,required" query:"q"`, false},
	}

	for _, tt := range tests {
		if got := hasTagOption(reflect.StructTag(tt.tag), "query", "required"); got != tt.want {
			t.Errorf("hasTagOption(%q) = %v, want %v", tt.tag, got, tt.want)
		}
	}
}

func TestTypeRef(t *testing.T) {
	tests := []struct {
		body string
		want TypeRef
	}{
		{"User", TypeRef{Schema: "User"}},
		{"  User  ", TypeRef{Schema: "User"}},
		{"[]User", TypeRef{IsArray: true, Schema: "User"}},
		{"map[string]User", TypeRef{IsMap: true, Schema: "User"}},
		{"int", TypeRef{Primitive: "integer"}},
		{"float64", TypeRef{Primitive: "number"}},
		{"bool", TypeRef{Primitive: "boolean"}},
		{"[]string", TypeRef{IsArray: true, Primitive: "string"}},
		{"map[string]int", TypeRef{IsMap: true, Primitive: "integer"}},
	}

	for _, tt := range tests {
		if got := *typeRef(tt.body); got != tt.want {
			t.Errorf("typeRef(%q) = %+v, want %+v", tt.body, got, tt.want)
		}
	}
}

func TestScalarType(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
		want string
	}{
		{"string", stringType, "string"},
		{"pointer", types.NewPointer(types.Typ[types.Int]), "integer"},
		{"slice", types.NewSlice(stringType), "array"},
		// []byte is base64 text rather than an array of numbers.
		{"byte slice", types.NewSlice(types.Typ[types.Byte]), "string"},
		{"map", types.NewMap(stringType, stringType), "string"},
		{"struct", structType, "string"},
		{"interface", types.NewInterfaceType(nil, nil), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scalarType(tt.typ); got != tt.want {
				t.Errorf("scalarType(%s) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}
