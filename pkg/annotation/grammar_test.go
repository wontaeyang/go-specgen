package annotation

import (
	"slices"
	"testing"
)

func TestSchema_Structure(t *testing.T) {
	// Test that Schema is properly initialized
	if Schema == nil {
		t.Fatal("Schema is nil")
	}

	if Schema.Name != "root" {
		t.Errorf("Schema.Name = %v, want root", Schema.Name)
	}

	// Test top-level annotations exist
	topLevel := []string{"@api", "@endpoint", "@field", "@schema", "@path", "@query", "@header", "@cookie"}
	for _, name := range topLevel {
		if !Schema.HasChild(name) {
			t.Errorf("Schema missing top-level annotation: %s", name)
		}
	}
}

func TestSchema_API(t *testing.T) {
	api := Schema.GetChild("@api")
	if api == nil {
		t.Fatal("@api annotation not found")
	}

	// Check required fields
	if !api.Required {
		t.Error("@api should be required")
	}

	// Check children
	expectedChildren := []string{
		"@title", "@version", "@description", "@termsOfService",
		"@contact", "@license", "@server", "@securityScheme", "@security",
	}

	for _, child := range expectedChildren {
		if !api.HasChild(child) {
			t.Errorf("@api missing child: %s", child)
		}
	}

	// Check required children
	if title := api.GetChild("@title"); title == nil || !title.Required {
		t.Error("@title should be required child of @api")
	}

	if version := api.GetChild("@version"); version == nil || !version.Required {
		t.Error("@version should be required child of @api")
	}
}

func TestSchema_Contact(t *testing.T) {
	api := Schema.GetChild("@api")
	contact := api.GetChild("@contact")

	if contact == nil {
		t.Fatal("@contact annotation not found")
	}

	// All block annotations support inline format
	if contact.Kind != Block {
		t.Error("@contact should be a Block")
	}

	// @contact can be empty (no required children)
	if !contact.CanBeEmpty() {
		t.Error("@contact should allow empty (no required children)")
	}

	// Check children
	expectedChildren := []string{"@name", "@email", "@url"}
	for _, child := range expectedChildren {
		if !contact.HasChild(child) {
			t.Errorf("@contact missing child: %s", child)
		}
	}
}

func TestSchema_Endpoint(t *testing.T) {
	endpoint := Schema.GetChild("@endpoint")
	if endpoint == nil {
		t.Fatal("@endpoint annotation not found")
	}

	if !endpoint.HasMetadata {
		t.Error("@endpoint should have metadata (METHOD /path)")
	}

	// Check parameter references
	paramRefs := []string{"@path", "@query", "@header", "@cookie"}
	for _, ref := range paramRefs {
		node := endpoint.GetChild(ref)
		if node == nil {
			t.Errorf("@endpoint missing parameter reference: %s", ref)
			continue
		}

		if node.Kind != Reference {
			t.Errorf("@endpoint.%s should be Reference", ref)
		}

		if !node.Repeatable {
			t.Errorf("@endpoint.%s should be repeatable", ref)
		}
	}
}

func TestSchema_Response(t *testing.T) {
	endpoint := Schema.GetChild("@endpoint")
	response := endpoint.GetChild("@response")

	if response == nil {
		t.Fatal("@response annotation not found")
	}

	if !response.HasMetadata {
		t.Error("@response should have metadata (status code)")
	}

	if !response.Repeatable {
		t.Error("@response should be repeatable")
	}

	// Check children
	if !response.HasChild("@contentType") {
		t.Error("@response missing @contentType")
	}

	if !response.HasChild("@body") {
		t.Error("@response missing @body")
	}
}

func TestSchema_Field(t *testing.T) {
	field := Schema.GetChild("@field")
	if field == nil {
		t.Fatal("@field annotation not found")
	}

	// All block annotations support inline format
	if field.Kind != Block {
		t.Error("@field should be a Block")
	}

	// @field can be empty (no required children)
	if !field.CanBeEmpty() {
		t.Error("@field should allow empty (no required children)")
	}

	// Check validation children
	validationChildren := []string{
		"@description", "@format", "@example", "@enum", "@default",
		"@minimum", "@maximum", "@minLength", "@maxLength", "@pattern",
	}

	for _, child := range validationChildren {
		if !field.HasChild(child) {
			t.Errorf("@field missing validation child: %s", child)
		}
	}

	// Check @deprecated is flag
	deprecated := field.GetChild("@deprecated")
	if deprecated == nil {
		t.Fatal("@field missing @deprecated")
	}

	if deprecated.Kind != Flag {
		t.Error("@deprecated should be Flag")
	}
}

func TestSchema_MarkerAnnotations(t *testing.T) {
	// @schema is now Block with optional children
	markers := []string{"@path", "@query", "@header", "@cookie"}

	for _, name := range markers {
		node := Schema.GetChild(name)
		if node == nil {
			t.Errorf("%s annotation not found", name)
			continue
		}

		if node.Kind != Marker {
			t.Errorf("%s should be Marker, got %v", name, node.Kind)
		}

		// Marker annotations have no children, so CanBeEmpty is true
		if !node.CanBeEmpty() {
			t.Errorf("%s should allow empty (no required children)", name)
		}

		if len(node.Children) > 0 {
			t.Errorf("%s should not have children (marker annotation)", name)
		}
	}
}

func TestSchema_SchemaBlock(t *testing.T) {
	schema := Schema.GetChild("@schema")
	if schema == nil {
		t.Fatal("@schema annotation not found")
	}

	if schema.Kind != Block {
		t.Errorf("@schema should be Block, got %v", schema.Kind)
	}

	// @schema can be empty (no required children)
	if !schema.CanBeEmpty() {
		t.Error("@schema should allow empty (no required children)")
	}

	desc := schema.GetChild("@description")
	if desc == nil {
		t.Error("@schema should have @description child")
	}

	deprecated := schema.GetChild("@deprecated")
	if deprecated == nil {
		t.Error("@schema should have @deprecated child")
	}
}

func TestSchema_Security(t *testing.T) {
	api := Schema.GetChild("@api")
	security := api.GetChild("@security")

	if security == nil {
		t.Fatal("@security annotation not found")
	}

	if !security.Repeatable {
		t.Error("@security should be repeatable (for OR logic)")
	}

	// Check @with sub-command
	with := security.GetChild("@with")
	if with == nil {
		t.Fatal("@security missing @with")
	}

	if with.Kind != SubCommand {
		t.Error("@with should be SubCommand type")
	}

	// All SubCommand types support inline format

	if !with.Repeatable {
		t.Error("@with should be repeatable (for AND logic)")
	}

	// Check @scope
	scope := with.GetChild("@scope")
	if scope == nil {
		t.Fatal("@with missing @scope")
	}

	if !scope.Repeatable {
		t.Error("@scope should be repeatable (multiple scopes for OAuth2)")
	}
}

func TestSchema_ParentReferences(t *testing.T) {
	// Test that parent references were initialized
	api := Schema.GetChild("@api")
	if api.Parent != Schema {
		t.Error("@api.Parent should be Schema")
	}

	title := api.GetChild("@title")
	if title.Parent != api {
		t.Error("@title.Parent should be @api")
	}

	contact := api.GetChild("@contact")
	name := contact.GetChild("@name")
	if name.Parent != contact {
		t.Error("@contact.@name.Parent should be @contact")
	}
}

func TestSchema_Endpoint_AllChildren(t *testing.T) {
	endpoint := Schema.GetChild("@endpoint")
	if endpoint == nil {
		t.Fatal("@endpoint annotation not found")
	}

	tests := []struct {
		name       string
		annType    Kind
		repeatable bool
	}{
		{"@operationID", Value, false},
		{"@summary", Value, false},
		{"@description", Value, false},
		{"@tag", Reference, true},
		{"@deprecated", Flag, false},
		{"@auth", Value, false},
		{"@path", Reference, true},
		{"@query", Reference, true},
		{"@header", Reference, true},
		{"@cookie", Reference, true},
		{"@request", Block, false},
		{"@response", Block, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := endpoint.GetChild(tt.name)
			if node == nil {
				t.Fatalf("@endpoint missing child: %s", tt.name)
			}
			if node.Kind != tt.annType {
				t.Errorf("@endpoint.%s type = %v, want %v", tt.name, node.Kind, tt.annType)
			}
			if node.Repeatable != tt.repeatable {
				t.Errorf("@endpoint.%s repeatable = %v, want %v", tt.name, node.Repeatable, tt.repeatable)
			}
		})
	}
}

func TestSchema_Request(t *testing.T) {
	endpoint := Schema.GetChild("@endpoint")
	request := endpoint.GetChild("@request")
	if request == nil {
		t.Fatal("@request annotation not found")
	}

	if request.Kind != Block {
		t.Errorf("@request type = %v, want Block", request.Kind)
	}

	children := []struct {
		name        string
		annType     Kind
		hasMetadata bool
	}{
		{"@contentType", Value, false},
		{"@body", Value, true},
		{"@bind", Value, false},
	}

	for _, tt := range children {
		t.Run(tt.name, func(t *testing.T) {
			node := request.GetChild(tt.name)
			if node == nil {
				t.Fatalf("@request missing child: %s", tt.name)
			}
			if node.Kind != tt.annType {
				t.Errorf("@request.%s type = %v, want %v", tt.name, node.Kind, tt.annType)
			}
			if node.HasMetadata != tt.hasMetadata {
				t.Errorf("@request.%s hasMetadata = %v, want %v", tt.name, node.HasMetadata, tt.hasMetadata)
			}
		})
	}
}

func TestSchema_Response_AllChildren(t *testing.T) {
	endpoint := Schema.GetChild("@endpoint")
	response := endpoint.GetChild("@response")
	if response == nil {
		t.Fatal("@response annotation not found")
	}

	children := []struct {
		name       string
		annType    Kind
		repeatable bool
	}{
		{"@contentType", Value, false},
		{"@body", Value, false},
		{"@bind", Value, false},
		{"@description", Value, false},
		{"@header", Value, true},
	}

	for _, tt := range children {
		t.Run(tt.name, func(t *testing.T) {
			node := response.GetChild(tt.name)
			if node == nil {
				t.Fatalf("@response missing child: %s", tt.name)
			}
			if node.Kind != tt.annType {
				t.Errorf("@response.%s type = %v, want %v", tt.name, node.Kind, tt.annType)
			}
			if node.Repeatable != tt.repeatable {
				t.Errorf("@response.%s repeatable = %v, want %v", tt.name, node.Repeatable, tt.repeatable)
			}
		})
	}

	// @description should support multiline
	desc := response.GetChild("@description")
	if !desc.SupportsMultiline {
		t.Error("@response.@description should support multiline")
	}

	// @body should have metadata
	body := response.GetChild("@body")
	if !body.HasMetadata {
		t.Error("@response.@body should have metadata")
	}
}

func TestSchema_Field_AllChildren(t *testing.T) {
	field := Schema.GetChild("@field")
	if field == nil {
		t.Fatal("@field annotation not found")
	}

	valueChildren := []string{
		"@description", "@format", "@example", "@enum", "@default",
		"@minimum", "@maximum", "@exclusiveMinimum", "@exclusiveMaximum",
		"@minLength", "@maxLength", "@minItems", "@maxItems", "@pattern",
	}
	for _, name := range valueChildren {
		t.Run(name, func(t *testing.T) {
			node := field.GetChild(name)
			if node == nil {
				t.Fatalf("@field missing child: %s", name)
			}
			if node.Kind != Value {
				t.Errorf("@field.%s type = %v, want Value", name, node.Kind)
			}
		})
	}

	flagChildren := []string{"@uniqueItems", "@deprecated", "@readOnly", "@writeOnly"}
	for _, name := range flagChildren {
		t.Run(name, func(t *testing.T) {
			node := field.GetChild(name)
			if node == nil {
				t.Fatalf("@field missing child: %s", name)
			}
			if node.Kind != Flag {
				t.Errorf("@field.%s type = %v, want Flag", name, node.Kind)
			}
		})
	}

	// @description should support multiline
	desc := field.GetChild("@description")
	if !desc.SupportsMultiline {
		t.Error("@field.@description should support multiline")
	}
}

func TestSchema_API_AllChildren(t *testing.T) {
	api := Schema.GetChild("@api")
	if api == nil {
		t.Fatal("@api annotation not found")
	}

	expectedChildren := []struct {
		name       string
		annType    Kind
		repeatable bool
	}{
		{"@title", Value, false},
		{"@version", Value, false},
		{"@description", Value, false},
		{"@termsOfService", Value, false},
		{"@contact", Block, false},
		{"@license", Block, false},
		{"@server", Block, true},
		{"@securityScheme", Block, true},
		{"@security", Block, true},
		{"@tag", Block, true},
		{"@defaultContentType", Value, false},
	}

	for _, tt := range expectedChildren {
		t.Run(tt.name, func(t *testing.T) {
			node := api.GetChild(tt.name)
			if node == nil {
				t.Fatalf("@api missing child: %s", tt.name)
			}
			if node.Kind != tt.annType {
				t.Errorf("@api.%s type = %v, want %v", tt.name, node.Kind, tt.annType)
			}
			if node.Repeatable != tt.repeatable {
				t.Errorf("@api.%s repeatable = %v, want %v", tt.name, node.Repeatable, tt.repeatable)
			}
		})
	}

	// @description should support multiline
	desc := api.GetChild("@description")
	if !desc.SupportsMultiline {
		t.Error("@api.@description should support multiline")
	}
}

func TestSchema_SecurityScheme(t *testing.T) {
	api := Schema.GetChild("@api")
	scheme := api.GetChild("@securityScheme")
	if scheme == nil {
		t.Fatal("@securityScheme annotation not found")
	}

	if !scheme.HasMetadata {
		t.Error("@securityScheme should have metadata (scheme name)")
	}

	// @type is required
	typeNode := scheme.GetChild("@type")
	if typeNode == nil {
		t.Fatal("@securityScheme missing @type")
	}
	if !typeNode.Required {
		t.Error("@securityScheme.@type should be required")
	}

	optionalChildren := []string{"@scheme", "@bearerFormat", "@in", "@name", "@description"}
	for _, name := range optionalChildren {
		node := scheme.GetChild(name)
		if node == nil {
			t.Errorf("@securityScheme missing child: %s", name)
			continue
		}
		if node.Required {
			t.Errorf("@securityScheme.%s should not be required", name)
		}
	}
}

func TestSchema_License(t *testing.T) {
	api := Schema.GetChild("@api")
	license := api.GetChild("@license")
	if license == nil {
		t.Fatal("@license annotation not found")
	}

	if license.Kind != Block {
		t.Errorf("@license type = %v, want Block", license.Kind)
	}

	expectedChildren := []string{"@name", "@url"}
	for _, name := range expectedChildren {
		if !license.HasChild(name) {
			t.Errorf("@license missing child: %s", name)
		}
	}
}

func TestSchema_Tag(t *testing.T) {
	api := Schema.GetChild("@api")
	tag := api.GetChild("@tag")
	if tag == nil {
		t.Fatal("@tag annotation not found")
	}

	if !tag.HasMetadata {
		t.Error("@tag should have metadata (tag name)")
	}

	if !tag.Repeatable {
		t.Error("@tag should be repeatable")
	}

	desc := tag.GetChild("@description")
	if desc == nil {
		t.Fatal("@tag missing @description")
	}
	if !desc.SupportsMultiline {
		t.Error("@tag.@description should support multiline")
	}
}

func TestSchema_Server(t *testing.T) {
	api := Schema.GetChild("@api")
	server := api.GetChild("@server")
	if server == nil {
		t.Fatal("@server annotation not found")
	}

	if !server.HasMetadata {
		t.Error("@server should have metadata (URL)")
	}

	if !server.Repeatable {
		t.Error("@server should be repeatable")
	}

	desc := server.GetChild("@description")
	if desc == nil {
		t.Fatal("@server missing @description")
	}
	if !desc.SupportsMultiline {
		t.Error("@server.@description should support multiline")
	}
}

func TestSchema_Integrity(t *testing.T) {
	if err := Schema.Validate(); err != nil {
		t.Errorf("Schema integrity validation failed: %v", err)
	}
}

// TestTargetIntegrity is what keeps Target from drifting. An annotation added
// to a root without one would be written on nothing, silently — the failure
// mode of the per-pass lists Target replaced. Declaration's children take no
// Target: that grammar covers exactly one context, so membership is the answer.
func TestTargetIntegrity(t *testing.T) {
	if err := Schema.ValidateTargets(true); err != nil {
		t.Errorf("Schema targets: %v", err)
	}
	if err := Declaration.ValidateTargets(false); err != nil {
		t.Errorf("Declaration targets: %v", err)
	}
}

// TestWrittenOn pins the sets the parser derives instead of holding its own
// copy of. Every top-level name lands under exactly one declaration.
func TestWrittenOn(t *testing.T) {
	want := map[Target][]string{
		OnPackage: {"@api"},
		OnType:    {"@cookie", "@header", "@path", "@query", "@schema"},
		OnFunc:    {"@endpoint"},
		OnField:   {"@field"},
	}

	var total int
	for target, names := range want {
		got := WrittenOn(target)
		if !slices.Equal(got, names) {
			t.Errorf("WrittenOn(%v) = %v, want %v", target, got, names)
		}
		total += len(names)
	}

	if total != len(Schema.Children) {
		t.Errorf("the targets cover %d names but Schema has %d children; one is written on nothing",
			total, len(Schema.Children))
	}
}

func TestInFunction(t *testing.T) {
	want := []string{"@cookie", "@header", "@path", "@query", "@request", "@response"}

	if got := InFunction(); !slices.Equal(got, want) {
		t.Errorf("InFunction() = %v, want %v", got, want)
	}
}

// TestIsKnown covers the question no single grammar node answers: whether a
// name means anything anywhere. It is what separates an annotation in the wrong
// place, which is a mistake with one correct reading, from a name that is
// indistinguishable from prose.
func TestIsKnown(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"@api", true},
		{"@endpoint", true},
		{"@schema", true},
		{"@field", true},
		{"@summary", true},   // nested: a child of @endpoint
		{"@minLength", true}, // nested: a child of @field
		{"@request", true},   // top level in Declaration only
		{"@bind", true},      // nested two levels down
		{"@endpoin", false},  // typo
		{"@schemaa", false},  // typo
		{"@route", false},    // plausible, but not this grammar
		{"@author", false},   // prose
		{"root", false},      // the tree's own name is not an annotation
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKnown(tt.name); got != tt.want {
				t.Errorf("IsKnown(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
