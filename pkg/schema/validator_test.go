package schema

import (
	"testing"
)

func TestGetTopLevelAnnotations(t *testing.T) {
	annotations := GetTopLevelAnnotations()

	if len(annotations) == 0 {
		t.Fatal("GetTopLevelAnnotations() returned empty slice")
	}

	// Check that expected top-level annotations are present
	expected := map[string]bool{
		"@api":      true,
		"@endpoint": true,
		"@field":    true,
		"@schema":   true,
		"@path":     true,
		"@query":    true,
		"@header":   true,
		"@cookie":   true,
	}

	for _, name := range annotations {
		if !expected[name] {
			t.Errorf("unexpected top-level annotation: %s", name)
		}
	}
}

func TestIsTopLevelAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"@api", true},
		{"@endpoint", true},
		{"@field", true},
		{"@schema", true},
		{"@title", false},    // child of @api
		{"@contact", false},  // child of @api
		{"@response", false}, // child of @endpoint
		{"@nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTopLevelAnnotation(tt.name); got != tt.expected {
				t.Errorf("IsTopLevelAnnotation(%s) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestIsMarkerAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"@schema", false},
		{"@path", true},
		{"@query", true},
		{"@header", true},
		{"@cookie", true},
		{"@api", false},
		{"@field", false},
		{"@endpoint", false},
		{"@nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMarkerAnnotation(tt.name); got != tt.expected {
				t.Errorf("IsMarkerAnnotation(%s) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestIsBlockAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"@api", true},
		{"@endpoint", true},
		{"@field", true},
		{"@schema", true},
		{"@nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBlockAnnotation(tt.name); got != tt.expected {
				t.Errorf("IsBlockAnnotation(%s) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}

	// Test nested block annotations
	api := AnnotationSchema.GetChild("@api")
	if contact := api.GetChild("@contact"); contact != nil && contact.Type != BlockAnnotation {
		t.Errorf("@contact should be BlockAnnotation")
	}

	endpoint := AnnotationSchema.GetChild("@endpoint")
	if response := endpoint.GetChild("@response"); response != nil && response.Type != BlockAnnotation {
		t.Errorf("@response should be BlockAnnotation")
	}
}

func TestIsValueAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"@api", false},    // BlockAnnotation
		{"@schema", false}, // MarkerAnnotation
		{"@field", false},  // BlockAnnotation
		{"@nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValueAnnotation(tt.name); got != tt.expected {
				t.Errorf("IsValueAnnotation(%s) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}

	// Test nested value annotations
	api := AnnotationSchema.GetChild("@api")
	valueTests := []string{"@title", "@version", "@description", "@termsOfService"}
	for _, name := range valueTests {
		if node := api.GetChild(name); node == nil {
			t.Errorf("%s not found in @api", name)
		} else if node.Type != ValueAnnotation {
			t.Errorf("%s should be ValueAnnotation, got %v", name, node.Type)
		}
	}
}

func TestIsReferenceAnnotation(t *testing.T) {
	// Get reference annotations from @endpoint
	endpoint := AnnotationSchema.GetChild("@endpoint")

	tests := []struct {
		name     string
		expected bool
	}{
		// Parameter references in @endpoint
		{"@api", false},
		{"@title", false},
		{"@nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReferenceAnnotation(tt.name); got != tt.expected {
				t.Errorf("IsReferenceAnnotation(%s) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}

	// Check parameter references within @endpoint context
	paramRefs := []string{"@path", "@query", "@header", "@cookie"}
	for _, ref := range paramRefs {
		node := endpoint.GetChild(ref)
		if node == nil || node.Type != ReferenceAnnotation {
			t.Errorf("@endpoint.%s should be ReferenceAnnotation", ref)
		}
	}
}

func TestHasMetadata(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"@endpoint", true},
		{"@api", false},
		{"@field", false},
		{"@contact", false},
		{"@nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasMetadata(tt.name); got != tt.expected {
				t.Errorf("HasMetadata(%s) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}

	// Test nested annotations separately
	api := AnnotationSchema.GetChild("@api")
	apiTests := []struct {
		name     string
		expected bool
	}{
		{"@server", true},
		{"@securityScheme", true},
	}

	for _, tt := range apiTests {
		t.Run(tt.name, func(t *testing.T) {
			node := api.GetChild(tt.name)
			if node == nil {
				t.Fatalf("annotation %s not found in @api", tt.name)
			}
			if got := node.HasMetadata; got != tt.expected {
				t.Errorf("%s.HasMetadata = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}

	endpoint := AnnotationSchema.GetChild("@endpoint")
	endpointTests := []struct {
		name     string
		expected bool
	}{
		{"@response", true},
	}

	for _, tt := range endpointTests {
		t.Run(tt.name, func(t *testing.T) {
			node := endpoint.GetChild(tt.name)
			if node == nil {
				t.Fatalf("annotation %s not found in @endpoint", tt.name)
			}
			if got := node.HasMetadata; got != tt.expected {
				t.Errorf("%s.HasMetadata = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestIsRepeatable(t *testing.T) {
	api := AnnotationSchema.GetChild("@api")
	endpoint := AnnotationSchema.GetChild("@endpoint")

	tests := []struct {
		name     string
		parent   *SchemaNode
		expected bool
	}{
		{"@server", api, true},
		{"@securityScheme", api, true},
		{"@security", api, true},
		{"@response", endpoint, true},
		{"@path", endpoint, true}, // repeatable for multiple param structs
		{"@query", endpoint, true},
		{"@header", endpoint, true},
		{"@cookie", endpoint, true},
		{"@title", api, false},
		{"@version", api, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fullName string
			if tt.parent != nil {
				node := tt.parent.GetChild(tt.name)
				if node == nil {
					t.Fatalf("annotation %s not found in parent", tt.name)
				}
				fullName = tt.name
				if got := node.Repeatable; got != tt.expected {
					t.Errorf("%s.Repeatable = %v, want %v", fullName, got, tt.expected)
				}
			} else {
				if got := IsRepeatable(tt.name); got != tt.expected {
					t.Errorf("IsRepeatable(%s) = %v, want %v", tt.name, got, tt.expected)
				}
			}
		})
	}
}

func TestAllowsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"@schema", true},
		{"@path", true},
		{"@query", true},
		{"@header", true},
		{"@cookie", true},
		{"@field", true},
		{"@api", false},
		{"@endpoint", true},
		{"@nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AllowsEmpty(tt.name); got != tt.expected {
				t.Errorf("AllowsEmpty(%s) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}

	// Test nested annotations separately (children of @api)
	api := AnnotationSchema.GetChild("@api")
	nestedTests := []struct {
		name     string
		expected bool
	}{
		{"@contact", true}, // no required children
		{"@license", true}, // no required children
		{"@server", true},  // no required children
	}

	for _, tt := range nestedTests {
		t.Run(tt.name, func(t *testing.T) {
			node := api.GetChild(tt.name)
			if node == nil {
				t.Fatalf("annotation %s not found in @api", tt.name)
			}
			if got := node.CanBeEmpty(); got != tt.expected {
				t.Errorf("%s.CanBeEmpty() = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestSupportsInline(t *testing.T) {
	// All block annotations support inline format now
	tests := []struct {
		name     string
		expected bool
	}{
		{"@field", true},    // BlockAnnotation
		{"@api", true},      // BlockAnnotation
		{"@endpoint", true}, // BlockAnnotation
		{"@schema", true},   // BlockAnnotation
		{"@path", false},    // MarkerAnnotation (not a block)
		{"@nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SupportsInline(tt.name); got != tt.expected {
				t.Errorf("SupportsInline(%s) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}

	// Test nested annotations separately (children of @api)
	api := AnnotationSchema.GetChild("@api")
	nestedTests := []struct {
		name     string
		expected bool
	}{
		{"@contact", true}, // BlockAnnotation
		{"@license", true}, // BlockAnnotation
	}

	for _, tt := range nestedTests {
		t.Run(tt.name, func(t *testing.T) {
			node := api.GetChild(tt.name)
			if node == nil {
				t.Fatalf("annotation %s not found in @api", tt.name)
			}
			// All block annotations support inline
			if node.Type != BlockAnnotation {
				t.Errorf("%s should be BlockAnnotation", tt.name)
			}
		})
	}
}

func TestGetChildrenNames(t *testing.T) {
	// Test @api children
	apiChildren := GetChildrenNames("@api")
	if len(apiChildren) == 0 {
		t.Error("@api should have children")
	}

	expectedAPI := map[string]bool{
		"@title": true, "@version": true, "@description": true,
		"@termsOfService": true, "@contact": true, "@license": true,
		"@server": true, "@securityScheme": true, "@security": true,
		"@tag": true, "@defaultContentType": true,
	}

	for _, child := range apiChildren {
		if !expectedAPI[child] {
			t.Errorf("unexpected child of @api: %s", child)
		}
	}

	// Test @schema (block annotation with children)
	schemaChildren := GetChildrenNames("@schema")
	if len(schemaChildren) != 2 {
		t.Errorf("@schema should have 2 children, got %d", len(schemaChildren))
	}

	// Test marker annotation (should have no children)
	pathChildren := GetChildrenNames("@path")
	if len(pathChildren) != 0 {
		t.Error("@path (marker) should have no children")
	}

	// Test non-existent annotation
	nonexistent := GetChildrenNames("@nonexistent")
	if nonexistent != nil {
		t.Error("non-existent annotation should return nil")
	}
}

func TestValidateNoNestedBraces(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"no braces", "simple content", false},
		{"single level braces", "content { inner }", false},
		{"nested braces", "content { outer { inner } }", true},
		{"unbalanced opening", "content { { inner }", true},
		{"unbalanced closing", "content } }", true},
		{"multiple single level", "{ one } and { two }", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNoNestedBraces(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNoNestedBraces(%q) error = %v, wantErr %v", tt.content, err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateAnnotation(t *testing.T) {
	validator := NewValidator(AnnotationSchema)

	tests := []struct {
		name           string
		annotationName string
		data           map[string]interface{}
		wantErr        bool
	}{
		{
			name:           "valid field annotation",
			annotationName: "@field",
			data: map[string]interface{}{
				"@description": "test description",
				"@format":      "email",
			},
			wantErr: false,
		},
		{
			name:           "valid empty field (marker pattern)",
			annotationName: "@field",
			data:           map[string]interface{}{},
			wantErr:        false,
		},
		{
			name:           "unknown annotation",
			annotationName: "@nonexistent",
			data:           map[string]interface{}{},
			wantErr:        true,
		},
		{
			name:           "unknown field in annotation",
			annotationName: "@field",
			data: map[string]interface{}{
				"@unknown": "value",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateAnnotation(tt.annotationName, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAnnotation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatAnnotationPath(t *testing.T) {
	tests := []struct {
		name     string
		path     []string
		expected string
	}{
		{"empty path", []string{}, ""},
		{"single element", []string{"@api"}, "@api"},
		{"nested path", []string{"@api", "@contact", "@name"}, "@api > @contact > @name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatAnnotationPath(tt.path); got != tt.expected {
				t.Errorf("FormatAnnotationPath() = %v, want %v", got, tt.expected)
			}
		})
	}
}
