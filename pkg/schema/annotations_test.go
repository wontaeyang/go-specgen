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

func TestAnnotationSchema_Endpoint_AllChildren(t *testing.T) {
	endpoint := AnnotationSchema.GetChild("@endpoint")
	if endpoint == nil {
		t.Fatal("@endpoint annotation not found")
	}

	tests := []struct {
		name       string
		annType    AnnotationType
		repeatable bool
	}{
		{"@operationID", ValueAnnotation, false},
		{"@summary", ValueAnnotation, false},
		{"@description", ValueAnnotation, false},
		{"@tag", ReferenceAnnotation, true},
		{"@deprecated", FlagAnnotation, false},
		{"@auth", ValueAnnotation, false},
		{"@path", ReferenceAnnotation, true},
		{"@query", ReferenceAnnotation, true},
		{"@header", ReferenceAnnotation, true},
		{"@cookie", ReferenceAnnotation, true},
		{"@request", BlockAnnotation, false},
		{"@response", BlockAnnotation, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := endpoint.GetChild(tt.name)
			if node == nil {
				t.Fatalf("@endpoint missing child: %s", tt.name)
			}
			if node.Type != tt.annType {
				t.Errorf("@endpoint.%s type = %v, want %v", tt.name, node.Type, tt.annType)
			}
			if node.Repeatable != tt.repeatable {
				t.Errorf("@endpoint.%s repeatable = %v, want %v", tt.name, node.Repeatable, tt.repeatable)
			}
		})
	}
}

func TestAnnotationSchema_Request(t *testing.T) {
	endpoint := AnnotationSchema.GetChild("@endpoint")
	request := endpoint.GetChild("@request")
	if request == nil {
		t.Fatal("@request annotation not found")
	}

	if request.Type != BlockAnnotation {
		t.Errorf("@request type = %v, want BlockAnnotation", request.Type)
	}

	children := []struct {
		name        string
		annType     AnnotationType
		hasMetadata bool
	}{
		{"@contentType", ValueAnnotation, false},
		{"@body", ValueAnnotation, true},
		{"@bind", ValueAnnotation, false},
	}

	for _, tt := range children {
		t.Run(tt.name, func(t *testing.T) {
			node := request.GetChild(tt.name)
			if node == nil {
				t.Fatalf("@request missing child: %s", tt.name)
			}
			if node.Type != tt.annType {
				t.Errorf("@request.%s type = %v, want %v", tt.name, node.Type, tt.annType)
			}
			if node.HasMetadata != tt.hasMetadata {
				t.Errorf("@request.%s hasMetadata = %v, want %v", tt.name, node.HasMetadata, tt.hasMetadata)
			}
		})
	}
}

func TestAnnotationSchema_Response_AllChildren(t *testing.T) {
	endpoint := AnnotationSchema.GetChild("@endpoint")
	response := endpoint.GetChild("@response")
	if response == nil {
		t.Fatal("@response annotation not found")
	}

	children := []struct {
		name       string
		annType    AnnotationType
		repeatable bool
	}{
		{"@contentType", ValueAnnotation, false},
		{"@body", ValueAnnotation, false},
		{"@bind", ValueAnnotation, false},
		{"@description", ValueAnnotation, false},
		{"@header", ValueAnnotation, true},
	}

	for _, tt := range children {
		t.Run(tt.name, func(t *testing.T) {
			node := response.GetChild(tt.name)
			if node == nil {
				t.Fatalf("@response missing child: %s", tt.name)
			}
			if node.Type != tt.annType {
				t.Errorf("@response.%s type = %v, want %v", tt.name, node.Type, tt.annType)
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

func TestAnnotationSchema_Field_AllChildren(t *testing.T) {
	field := AnnotationSchema.GetChild("@field")
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
			if node.Type != ValueAnnotation {
				t.Errorf("@field.%s type = %v, want ValueAnnotation", name, node.Type)
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
			if node.Type != FlagAnnotation {
				t.Errorf("@field.%s type = %v, want FlagAnnotation", name, node.Type)
			}
		})
	}

	// @description should support multiline
	desc := field.GetChild("@description")
	if !desc.SupportsMultiline {
		t.Error("@field.@description should support multiline")
	}
}

func TestAnnotationSchema_API_AllChildren(t *testing.T) {
	api := AnnotationSchema.GetChild("@api")
	if api == nil {
		t.Fatal("@api annotation not found")
	}

	expectedChildren := []struct {
		name       string
		annType    AnnotationType
		repeatable bool
	}{
		{"@title", ValueAnnotation, false},
		{"@version", ValueAnnotation, false},
		{"@description", ValueAnnotation, false},
		{"@termsOfService", ValueAnnotation, false},
		{"@contact", BlockAnnotation, false},
		{"@license", BlockAnnotation, false},
		{"@server", BlockAnnotation, true},
		{"@securityScheme", BlockAnnotation, true},
		{"@security", BlockAnnotation, true},
		{"@tag", BlockAnnotation, true},
		{"@defaultContentType", ValueAnnotation, false},
	}

	for _, tt := range expectedChildren {
		t.Run(tt.name, func(t *testing.T) {
			node := api.GetChild(tt.name)
			if node == nil {
				t.Fatalf("@api missing child: %s", tt.name)
			}
			if node.Type != tt.annType {
				t.Errorf("@api.%s type = %v, want %v", tt.name, node.Type, tt.annType)
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

func TestAnnotationSchema_SecurityScheme(t *testing.T) {
	api := AnnotationSchema.GetChild("@api")
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

func TestAnnotationSchema_License(t *testing.T) {
	api := AnnotationSchema.GetChild("@api")
	license := api.GetChild("@license")
	if license == nil {
		t.Fatal("@license annotation not found")
	}

	if license.Type != BlockAnnotation {
		t.Errorf("@license type = %v, want BlockAnnotation", license.Type)
	}

	expectedChildren := []string{"@name", "@url"}
	for _, name := range expectedChildren {
		if !license.HasChild(name) {
			t.Errorf("@license missing child: %s", name)
		}
	}
}

func TestAnnotationSchema_Tag(t *testing.T) {
	api := AnnotationSchema.GetChild("@api")
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

func TestAnnotationSchema_Server(t *testing.T) {
	api := AnnotationSchema.GetChild("@api")
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

func TestAnnotationSchema_Integrity(t *testing.T) {
	if err := AnnotationSchema.Validate(); err != nil {
		t.Errorf("Schema integrity validation failed: %v", err)
	}
}
