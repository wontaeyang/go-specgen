package schema

import (
	"testing"
)

func TestAnnotationSchema_Structure(t *testing.T) {
	// Test that AnnotationSchema is properly initialized
	if AnnotationSchema == nil {
		t.Fatal("AnnotationSchema is nil")
	}

	if AnnotationSchema.Name != "root" {
		t.Errorf("AnnotationSchema.Name = %v, want root", AnnotationSchema.Name)
	}

	// Test top-level annotations exist
	topLevel := []string{"@api", "@endpoint", "@field", "@schema", "@path", "@query", "@header", "@cookie"}
	for _, name := range topLevel {
		if !AnnotationSchema.HasChild(name) {
			t.Errorf("AnnotationSchema missing top-level annotation: %s", name)
		}
	}
}

func TestAnnotationSchema_API(t *testing.T) {
	api := AnnotationSchema.GetChild("@api")
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

func TestAnnotationSchema_Contact(t *testing.T) {
	api := AnnotationSchema.GetChild("@api")
	contact := api.GetChild("@contact")

	if contact == nil {
		t.Fatal("@contact annotation not found")
	}

	// All block annotations support inline format
	if contact.Type != BlockAnnotation {
		t.Error("@contact should be a BlockAnnotation")
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

func TestAnnotationSchema_Endpoint(t *testing.T) {
	endpoint := AnnotationSchema.GetChild("@endpoint")
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

		if node.Type != ReferenceAnnotation {
			t.Errorf("@endpoint.%s should be ReferenceAnnotation", ref)
		}

		if !node.Repeatable {
			t.Errorf("@endpoint.%s should be repeatable", ref)
		}
	}
}

func TestAnnotationSchema_Response(t *testing.T) {
	endpoint := AnnotationSchema.GetChild("@endpoint")
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

func TestAnnotationSchema_Field(t *testing.T) {
	field := AnnotationSchema.GetChild("@field")
	if field == nil {
		t.Fatal("@field annotation not found")
	}

	// All block annotations support inline format
	if field.Type != BlockAnnotation {
		t.Error("@field should be a BlockAnnotation")
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

	if deprecated.Type != FlagAnnotation {
		t.Error("@deprecated should be FlagAnnotation")
	}
}

func TestAnnotationSchema_MarkerAnnotations(t *testing.T) {
	// @schema is now BlockAnnotation with optional children
	markers := []string{"@path", "@query", "@header", "@cookie"}

	for _, name := range markers {
		node := AnnotationSchema.GetChild(name)
		if node == nil {
			t.Errorf("%s annotation not found", name)
			continue
		}

		if node.Type != MarkerAnnotation {
			t.Errorf("%s should be MarkerAnnotation, got %v", name, node.Type)
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

func TestAnnotationSchema_SchemaBlock(t *testing.T) {
	schema := AnnotationSchema.GetChild("@schema")
	if schema == nil {
		t.Fatal("@schema annotation not found")
	}

	if schema.Type != BlockAnnotation {
		t.Errorf("@schema should be BlockAnnotation, got %v", schema.Type)
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

func TestAnnotationSchema_Security(t *testing.T) {
	api := AnnotationSchema.GetChild("@api")
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

	if with.Type != SubCommand {
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

func TestAnnotationSchema_ParentReferences(t *testing.T) {
	// Test that parent references were initialized
	api := AnnotationSchema.GetChild("@api")
	if api.Parent != AnnotationSchema {
		t.Error("@api.Parent should be AnnotationSchema")
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

func TestAnnotationSchema_Integrity(t *testing.T) {
	if err := ValidateSchemaIntegrity(); err != nil {
		t.Errorf("Schema integrity validation failed: %v", err)
	}
}
