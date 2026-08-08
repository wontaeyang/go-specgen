package generator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/resolver"
	"go.yaml.in/yaml/v4"
)

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator("3.1")
	if gen == nil {
		t.Fatal("NewGenerator() returned nil")
	}

	if gen.version != "3.1" {
		t.Errorf("version = %s, want 3.1", gen.version)
	}
}

func TestGenerator_Generate(t *testing.T) {
	pkg := &resolver.Package{
		API: &resolver.API{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Schemas: map[string]*resolver.Schema{
			"User": {
				Name: "User",
				Fields: []*resolver.Field{
					{
						Name:     "id",
						GoName:   "ID",
						Type:     &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
						Required: true,
					},
				},
			},
		},
		Parameters: map[string]*resolver.ParameterStruct{},
		Endpoints: []*resolver.Endpoint{
			{
				Method: "GET",
				Path:   "/users",
				Responses: map[string]*resolver.Response{
					"200": {StatusCode: "200", Description: "Success"},
				},
			},
		},
	}

	gen := NewGenerator("3.1")
	spec, err := gen.Generate(pkg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if spec == nil {
		t.Fatal("Generate() returned nil spec")
	}

	// Verify openapi version
	if spec.Version != "3.1.0" {
		t.Errorf("openapi = %v, want 3.1.0", spec.Version)
	}

	// Verify info
	if spec.Info == nil {
		t.Fatal("info is nil")
	}

	// Verify paths
	if spec.Paths == nil {
		t.Fatal("paths is nil")
	}

	// Verify components
	if spec.Components == nil {
		t.Fatal("components is nil")
	}
}

func TestGenerator_GenerateInfo(t *testing.T) {
	api := &resolver.API{
		Title:          "Test API",
		Version:        "1.0.0",
		Description:    "Test description",
		TermsOfService: "https://example.com/terms",
		Contact: &resolver.Contact{
			Name:  "API Team",
			Email: "api@example.com",
			URL:   "https://example.com",
		},
		License: &resolver.License{
			Name: "MIT",
			URL:  "https://opensource.org/licenses/MIT",
		},
	}

	gen := NewGenerator("3.1")
	info := gen.generateInfo(api)

	if info.Title != "Test API" {
		t.Errorf("Title = %v, want Test API", info.Title)
	}

	if info.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", info.Version)
	}

	if info.Description != "Test description" {
		t.Errorf("Description = %v, want Test description", info.Description)
	}

	if info.Contact == nil {
		t.Fatal("Contact is nil")
	}

	if info.License == nil {
		t.Fatal("License is nil")
	}
}

func TestGenerator_GenerateServers(t *testing.T) {
	servers := []*resolver.Server{
		{
			URL:         "https://api.example.com",
			Description: "Production",
		},
		{
			URL:         "https://staging.example.com",
			Description: "Staging",
		},
	}

	gen := NewGenerator("3.1")
	result := gen.generateServers(servers)

	if len(result) != 2 {
		t.Fatalf("Expected 2 servers, got %d", len(result))
	}

	if result[0].URL != "https://api.example.com" {
		t.Errorf("First server URL = %v, want https://api.example.com", result[0].URL)
	}

	if result[1].Description != "Staging" {
		t.Errorf("Second server description = %v, want Staging", result[1].Description)
	}
}

func TestGenerator_GenerateSchemas(t *testing.T) {
	schemas := map[string]*resolver.Schema{
		"User": {
			Name:        "User",
			Description: "User model",
			Fields: []*resolver.Field{
				{
					Name:        "id",
					GoName:      "ID",
					Type:        &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
					Format:      "uuid",
					Required:    true,
					Description: "User ID",
				},
				{
					Name:     "email",
					GoName:   "Email",
					Type:     &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
					Format:   "email",
					Required: true,
				},
			},
		},
	}

	gen := NewGenerator("3.1")
	result := gen.generateSchemas(schemas)

	if result.Len() != 1 {
		t.Fatalf("Expected 1 schema, got %d", result.Len())
	}

	user := result.GetOrZero("User")
	if user == nil {
		t.Fatal("User schema not found")
	}
}

func TestGenerator_RenderJSON(t *testing.T) {
	pkg := &resolver.Package{
		API: &resolver.API{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Schemas:    map[string]*resolver.Schema{},
		Parameters: map[string]*resolver.ParameterStruct{},
		Endpoints: []*resolver.Endpoint{
			{
				Method: "GET",
				Path:   "/test",
				Responses: map[string]*resolver.Response{
					"200": {StatusCode: "200", Description: "OK"},
				},
			},
		},
	}

	gen := NewGenerator("3.1")
	doc, err := gen.Generate(pkg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := gen.RenderJSON(doc)
	if err != nil {
		t.Fatalf("render error = %v", err)
	}

	// Verify it's valid JSON
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Check openapi version
	if version, ok := obj["openapi"].(string); !ok || version == "" {
		t.Error("Missing or invalid openapi version")
	}

	// Check info
	if info, ok := obj["info"].(map[string]interface{}); !ok {
		t.Error("Missing info section")
	} else {
		if title, ok := info["title"].(string); !ok || title != "Test API" {
			t.Errorf("Title = %v, want Test API", title)
		}
	}

	// Check paths
	if paths, ok := obj["paths"].(map[string]interface{}); !ok {
		t.Error("Missing paths section")
	} else {
		if _, ok := paths["/test"]; !ok {
			t.Error("Missing /test path")
		}
	}
}

func TestGenerator_RenderYAML(t *testing.T) {
	pkg := &resolver.Package{
		API: &resolver.API{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Schemas:    map[string]*resolver.Schema{},
		Parameters: map[string]*resolver.ParameterStruct{},
		Endpoints: []*resolver.Endpoint{
			{
				Method: "GET",
				Path:   "/test",
				Responses: map[string]*resolver.Response{
					"200": {StatusCode: "200", Description: "OK"},
				},
			},
		},
	}

	gen := NewGenerator("3.1")
	doc, err := gen.Generate(pkg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := gen.RenderYAML(doc)
	if err != nil {
		t.Fatalf("render error = %v", err)
	}

	// Verify it contains YAML-like content
	yamlStr := string(data)
	if !strings.Contains(yamlStr, "openapi:") {
		t.Error("YAML should contain 'openapi:'")
	}

	if !strings.Contains(yamlStr, "info:") {
		t.Error("YAML should contain 'info:'")
	}

	if !strings.Contains(yamlStr, "Test API") {
		t.Error("YAML should contain 'Test API'")
	}

	// Verify it's valid YAML
	var obj map[string]interface{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		t.Fatalf("Generated YAML is invalid: %v", err)
	}
}

func TestGenerator_GeneratePaths(t *testing.T) {
	endpoints := []*resolver.Endpoint{
		{
			Method:  "GET",
			Path:    "/users",
			Summary: "List users",
			Responses: map[string]*resolver.Response{
				"200": {StatusCode: "200", Description: "Success"},
			},
		},
		{
			Method:  "POST",
			Path:    "/users",
			Summary: "Create user",
			Responses: map[string]*resolver.Response{
				"201": {StatusCode: "201", Description: "Created"},
			},
		},
	}

	gen := NewGenerator("3.1")
	paths := gen.generatePaths(endpoints, map[string]*resolver.ParameterStruct{}, map[string]*resolver.Schema{})

	if paths.PathItems.Len() != 1 {
		t.Fatalf("Expected 1 path item, got %d", paths.PathItems.Len())
	}

	usersPath := paths.PathItems.GetOrZero("/users")
	if usersPath == nil {
		t.Fatal("/users path not found")
	}

	// Check that both GET and POST operations exist
	if usersPath.Get == nil {
		t.Error("GET operation not found")
	}
	if usersPath.Post == nil {
		t.Error("POST operation not found")
	}
}

func TestGenerator_GenerateOperation(t *testing.T) {
	endpoint := &resolver.Endpoint{
		Method:      "GET",
		Path:        "/users/{id}",
		Summary:     "Get user",
		Description: "Get user by ID",
		OperationID: "getUser",
		Tags:        []string{"users"},
		PathParams: []*resolver.ParameterStruct{
			{
				Name: "UserIDPath",
				Fields: []*resolver.Field{
					{
						Name:     "id",
						GoName:   "ID",
						Type:     &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
						Required: true,
					},
				},
			},
		},
		Responses: map[string]*resolver.Response{
			"200": {
				StatusCode:  "200",
				Description: "Success",
				ContentType: "application/json",
				Body:        &resolver.Body{Schema: "User", Type: &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "User"}},
			},
		},
	}

	paramMap := map[string]*resolver.ParameterStruct{
		"UserIDPath": endpoint.PathParams[0],
	}

	schemas := map[string]*resolver.Schema{
		"User": {Name: "User"},
	}

	gen := NewGenerator("3.1")
	op := gen.generateOperation(endpoint, paramMap, schemas)

	if op.Summary != "Get user" {
		t.Errorf("Summary = %v, want Get user", op.Summary)
	}

	if op.OperationId != "getUser" {
		t.Errorf("OperationId = %v, want getUser", op.OperationId)
	}

	if len(op.Tags) == 0 {
		t.Error("Tags not set")
	}

	if len(op.Parameters) == 0 {
		t.Fatal("Expected parameters")
	}

	if op.Responses == nil {
		t.Fatal("Expected responses")
	}
}

func TestGenerator_HTTPMethodsLowercase(t *testing.T) {
	endpoints := []*resolver.Endpoint{
		{
			Method: "GET",
			Path:   "/test",
			Responses: map[string]*resolver.Response{
				"200": {StatusCode: "200", Description: "OK"},
			},
		},
		{
			Method: "POST",
			Path:   "/test",
			Responses: map[string]*resolver.Response{
				"201": {StatusCode: "201", Description: "Created"},
			},
		},
	}

	gen := NewGenerator("3.1")
	paths := gen.generatePaths(endpoints, map[string]*resolver.ParameterStruct{}, map[string]*resolver.Schema{})

	testPath := paths.PathItems.GetOrZero("/test")
	if testPath == nil {
		t.Fatal("/test path not found")
	}

	// Check that methods are set correctly
	if testPath.Get == nil {
		t.Error("GET operation not found")
	}

	if testPath.Post == nil {
		t.Error("POST operation not found")
	}
}

func TestGenerator_GenerateTags(t *testing.T) {
	gen := NewGenerator("3.1")

	tests := []struct {
		name string
		tags []*resolver.Tag
		want int // Expected number of tags
	}{
		{
			name: "single tag",
			tags: []*resolver.Tag{
				{Name: "pets", Description: "Pet operations"},
			},
			want: 1,
		},
		{
			name: "multiple tags",
			tags: []*resolver.Tag{
				{Name: "pets", Description: "Pet operations"},
				{Name: "users", Description: "User operations"},
			},
			want: 2,
		},
		{
			name: "tag without description",
			tags: []*resolver.Tag{
				{Name: "pets", Description: ""},
			},
			want: 1,
		},
		{
			name: "empty tags",
			tags: []*resolver.Tag{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateTags(tt.tags)

			if len(result) != tt.want {
				t.Errorf("generateTags() returned %d tags, want %d", len(result), tt.want)
			}

			// Validate structure of each tag
			for i, tag := range result {
				if tag.Name != tt.tags[i].Name {
					t.Errorf("tag[%d] name = %v, want %s", i, tag.Name, tt.tags[i].Name)
				}

				if tt.tags[i].Description != "" && tag.Description != tt.tags[i].Description {
					t.Errorf("tag[%d] description = %v, want %s", i, tag.Description, tt.tags[i].Description)
				}
			}
		})
	}
}

func TestGenerator_GenerateWithTags(t *testing.T) {
	gen := NewGenerator("3.1")

	pkg := &resolver.Package{
		API: &resolver.API{
			Title:   "Test API",
			Version: "1.0.0",
			Tags: []*resolver.Tag{
				{Name: "pets", Description: "Pet operations"},
				{Name: "users", Description: "User operations"},
			},
		},
		Schemas:    map[string]*resolver.Schema{},
		Parameters: map[string]*resolver.ParameterStruct{},
		Endpoints: []*resolver.Endpoint{
			{
				Method: "GET",
				Path:   "/pets",
				Tags:   []string{"pets"},
				Responses: map[string]*resolver.Response{
					"200": {StatusCode: "200", Description: "Success"},
				},
			},
		},
	}

	spec, err := gen.Generate(pkg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check tags section exists
	if spec.Tags == nil {
		t.Fatal("spec missing 'tags' field")
	}

	if len(spec.Tags) != 2 {
		t.Errorf("tags array has %d items, want 2", len(spec.Tags))
	}

	// Verify first tag
	if spec.Tags[0].Name != "pets" {
		t.Errorf("tags[0].name = %v, want 'pets'", spec.Tags[0].Name)
	}

	if spec.Tags[0].Description != "Pet operations" {
		t.Errorf("tags[0].description = %v, want 'Pet operations'", spec.Tags[0].Description)
	}

	// Verify second tag
	if spec.Tags[1].Name != "users" {
		t.Errorf("tags[1].name = %v, want 'users'", spec.Tags[1].Name)
	}

	if spec.Tags[1].Description != "User operations" {
		t.Errorf("tags[1].description = %v, want 'User operations'", spec.Tags[1].Description)
	}
}

func TestGenerator_GenerateBodySchema_Ref(t *testing.T) {
	// Test body without bind - should use $ref
	gen := NewGenerator("3.1")

	body := &resolver.Body{
		Schema: "User",
		Type:   &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "User"},
	}

	schemas := map[string]*resolver.Schema{}

	result := gen.generateBodySchema(body, schemas)

	if !result.IsReference() {
		t.Error("Expected schema reference")
	}

	if result.GetReference() != "#/components/schemas/User" {
		t.Errorf("$ref = %v, want #/components/schemas/User", result.GetReference())
	}
}

func TestGenerator_GenerateBodySchema_Wrapped(t *testing.T) {
	// Test body with bind - should be wrapped
	gen := NewGenerator("3.1")

	wrapperSchema := &resolver.Schema{
		Name: "DataResponse",
		Fields: []*resolver.Field{
			{Name: "status", GoName: "Status", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}, Required: true},
			{Name: "data", GoName: "Data", Type: &resolver.TypeRef{Shape: resolver.ShapeObject}, Required: true},
		},
	}

	body := &resolver.Body{
		Schema: "User",
		Type:   &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "User"},
		Bind: &resolver.BindTarget{
			Wrapper:       "DataResponse",
			Field:         "Data",
			WrapperSchema: wrapperSchema,
		},
	}

	schemas := map[string]*resolver.Schema{
		"DataResponse": wrapperSchema,
	}

	result := gen.generateBodySchema(body, schemas)

	// Should not be a reference (wrapped schema is inlined)
	if result.IsReference() {
		t.Error("Should not be a reference for wrapped schema")
	}

	// Build and check the schema
	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	if len(schema.Type) == 0 || schema.Type[0] != "object" {
		t.Errorf("type = %v, want object", schema.Type)
	}

	// Should have properties
	if schema.Properties == nil {
		t.Fatal("properties not found")
	}
}

func TestGenerator_BuildTypeProxy_Simple(t *testing.T) {
	gen := NewGenerator("3.1")

	result := gen.buildTypeProxy(&resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "User"})

	if !result.IsReference() {
		t.Error("Expected schema reference")
	}

	if result.GetReference() != "#/components/schemas/User" {
		t.Errorf("$ref = %v, want #/components/schemas/User", result.GetReference())
	}
}

func TestGenerator_BuildTypeProxy_Array(t *testing.T) {
	gen := NewGenerator("3.1")

	result := gen.buildTypeProxy(&resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "User"}})

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	if len(schema.Type) == 0 || schema.Type[0] != "array" {
		t.Errorf("type = %v, want array", schema.Type)
	}

	if schema.Items == nil {
		t.Fatal("items not found")
	}
}

func TestGenerator_BuildTypeProxy_Map(t *testing.T) {
	gen := NewGenerator("3.1")

	result := gen.buildTypeProxy(&resolver.TypeRef{Shape: resolver.ShapeMap, Elem: &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "User"}})

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	if len(schema.Type) == 0 || schema.Type[0] != "object" {
		t.Errorf("type = %v, want object", schema.Type)
	}

	if schema.AdditionalProperties == nil {
		t.Fatal("additionalProperties not found")
	}
}

func TestGenerator_BuildTypeProxy_Primitive(t *testing.T) {
	gen := NewGenerator("3.1")

	result := gen.buildTypeProxy(&resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "integer"})

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	if len(schema.Type) == 0 || schema.Type[0] != "integer" {
		t.Errorf("type = %v, want integer", schema.Type)
	}
}

// renderRefField renders a $ref field through generateFieldSchemaWithRefs and
// returns the YAML so sibling keywords (which live on the proxy, not the built
// schema) can be asserted.
func renderRefField(t *testing.T, version string, field *resolver.Field) string {
	t.Helper()
	gen := NewGenerator(version)
	proxy := gen.generateFieldSchema(field)
	out, err := yaml.Marshal(proxy)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	return string(out)
}

func TestGenerator_RefField_BareWhenNoSiblings(t *testing.T) {
	field := &resolver.Field{Name: "home_address", GoName: "HomeAddress", GoType: "Address",
		Type: &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "Address"}}
	got := renderRefField(t, "3.1", field)

	if !strings.Contains(got, "$ref: '#/components/schemas/Address'") {
		t.Errorf("expected bare $ref, got:\n%s", got)
	}
	if strings.Contains(got, "allOf") || strings.Contains(got, "oneOf") {
		t.Errorf("unannotated non-nullable ref should not be wrapped, got:\n%s", got)
	}
}

func TestGenerator_RefField_Deprecated_31Siblings(t *testing.T) {
	field := &resolver.Field{
		Name: "home_address", GoName: "HomeAddress", GoType: "Address",
		Type:        &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "Address"},
		Description: "Home address", Deprecated: true,
	}
	got := renderRefField(t, "3.1", field)

	// 3.1+: $ref carries siblings directly, no wrapper.
	if strings.Contains(got, "allOf") || strings.Contains(got, "oneOf") {
		t.Errorf("3.1 ref with siblings should not be wrapped, got:\n%s", got)
	}
	if !strings.Contains(got, "$ref: '#/components/schemas/Address'") {
		t.Errorf("missing $ref, got:\n%s", got)
	}
	if !strings.Contains(got, "deprecated: true") {
		t.Errorf("missing deprecated sibling, got:\n%s", got)
	}
	if !strings.Contains(got, "description: Home address") {
		t.Errorf("missing description sibling, got:\n%s", got)
	}
}

func TestGenerator_RefField_NullableDeprecated_31OneOf(t *testing.T) {
	field := &resolver.Field{
		Name: "work_address", GoName: "WorkAddress", GoType: "Address",
		Type:        &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "Address"},
		Description: "Work address", Deprecated: true, Nullable: true,
	}
	got := renderRefField(t, "3.1", field)

	// Nullable refs need oneOf in 3.1 ($ref siblings would intersect, not union).
	if !strings.Contains(got, "oneOf") {
		t.Errorf("nullable 3.1 ref should use oneOf, got:\n%s", got)
	}
	if !strings.Contains(got, `type: "null"`) {
		t.Errorf("nullable 3.1 ref should include null type, got:\n%s", got)
	}
	if !strings.Contains(got, "deprecated: true") {
		t.Errorf("missing deprecated sibling, got:\n%s", got)
	}
}

func TestGenerator_GenerateInlineWrappedSchema(t *testing.T) {
	gen := NewGenerator("3.1")

	wrapperSchema := &resolver.Schema{
		Name: "DataResponse",
		Fields: []*resolver.Field{
			{Name: "status", GoName: "Status", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}, Required: true},
			{Name: "data", GoName: "Data", Type: &resolver.TypeRef{Shape: resolver.ShapeObject}, Required: true},
		},
	}

	// Create an inline body with bind
	inline := &resolver.InlineBody{
		ContentType: "application/json",
		Fields: []*resolver.Field{
			{Name: "id", GoName: "ID", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}, Required: true},
			{Name: "email", GoName: "Email", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}, Required: true},
		},
		Bind: &resolver.BindTarget{
			Wrapper:       "DataResponse",
			Field:         "Data",
			WrapperSchema: wrapperSchema,
		},
	}

	schemas := map[string]*resolver.Schema{
		"DataResponse": wrapperSchema,
	}

	result := gen.generateInlineWrappedSchema(inline, schemas)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	// Should have type: object
	if len(schema.Type) == 0 || schema.Type[0] != "object" {
		t.Errorf("type = %v, want object", schema.Type)
	}

	// Should have properties
	if schema.Properties == nil {
		t.Fatal("properties not found")
	}

	// Should have status property
	if schema.Properties.GetOrZero("status") == nil {
		t.Error("status property not found")
	}

	// Should have data property with nested inline schema
	if schema.Properties.GetOrZero("data") == nil {
		t.Error("data property not found")
	}
}

func TestGenerator_GenerateInlineWrappedSchema_NoBind(t *testing.T) {
	gen := NewGenerator("3.1")

	// Create an inline body without bind - should fall back to inline schema
	inline := &resolver.InlineBody{
		ContentType: "application/json",
		Fields: []*resolver.Field{
			{Name: "id", GoName: "ID", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}, Required: true},
		},
		Bind: nil,
	}

	result := gen.generateInlineWrappedSchema(inline, nil)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	// Should be inline schema (type: object with id)
	if len(schema.Type) == 0 || schema.Type[0] != "object" {
		t.Errorf("type = %v, want object", schema.Type)
	}

	if schema.Properties == nil {
		t.Fatal("properties not found")
	}

	if schema.Properties.GetOrZero("id") == nil {
		t.Error("id property not found")
	}
}

func TestGenerator_GenerateFieldSchema_ArrayEnum(t *testing.T) {
	gen := NewGenerator("3.1")

	field := &resolver.Field{
		Name:   "tags",
		GoName: "Tags",
		Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
		Enum:   []string{"red", "green", "blue"},
	}

	result := gen.generateFieldSchema(field)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	// Verify it's an array
	if len(schema.Type) == 0 || schema.Type[0] != "array" {
		t.Errorf("type = %v, want array", schema.Type)
	}

	// Verify items exists
	if schema.Items == nil {
		t.Fatal("items not found")
	}

	// For arrays, enum goes inside items - check that items schema has enum
	itemsSchema, err := schema.Items.A.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() for items error = %v", err)
	}

	if len(itemsSchema.Enum) != 3 {
		t.Errorf("enum has %d values, want 3", len(itemsSchema.Enum))
	}

	// Enum should NOT be at top level
	if len(schema.Enum) > 0 {
		t.Error("enum should not be at top level for array fields")
	}
}

func TestGenerator_GenerateFieldSchema_IntegerEnum(t *testing.T) {
	gen := NewGenerator("3.1")

	field := &resolver.Field{
		Name:   "priority",
		GoName: "Priority",
		Type:   &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "integer"},
		Enum:   []string{"1", "2", "3"},
	}

	result := gen.generateFieldSchema(field)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	// Verify enum exists
	if len(schema.Enum) != 3 {
		t.Fatalf("enum has %d values, want 3", len(schema.Enum))
	}

	// Verify enum values are integers (yaml nodes with !!int tag)
	for i, v := range schema.Enum {
		if v.Tag != "!!int" {
			t.Errorf("enum[%d] tag = %v, want !!int", i, v.Tag)
		}
	}
}

func TestGenerator_GenerateParameterFieldSchema_ArrayEnum(t *testing.T) {
	gen := NewGenerator("3.1")

	field := &resolver.Field{
		Name:   "status",
		GoName: "Status",
		Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
		Enum:   []string{"active", "pending", "done"},
	}

	result := gen.generateParameterFieldSchema(field)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	// Verify items has enum
	if schema.Items == nil {
		t.Fatal("items not found")
	}

	itemsSchema, err := schema.Items.A.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() for items error = %v", err)
	}

	if len(itemsSchema.Enum) != 3 {
		t.Errorf("enum has %d values, want 3", len(itemsSchema.Enum))
	}
}

func TestGenerator_Version31Nullable(t *testing.T) {
	gen := NewGenerator("3.1")

	field := &resolver.Field{
		Name:     "nickname",
		GoName:   "Nickname",
		Type:     &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
		Nullable: true,
	}

	result := gen.generateFieldSchema(field)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	// In 3.1, nullable should NOT be set
	if schema.Nullable != nil {
		t.Error("nullable should not be set for 3.1")
	}

	// Type should be ["string", "null"]
	if len(schema.Type) != 2 {
		t.Errorf("type = %v, want [string, null]", schema.Type)
	}

	hasNull := false
	for _, typ := range schema.Type {
		if typ == "null" {
			hasNull = true
			break
		}
	}
	if !hasNull {
		t.Error("type should include 'null' for 3.1")
	}
}

func TestGenerator_Version32Nullable(t *testing.T) {
	gen := NewGenerator("3.2")

	field := &resolver.Field{
		Name:     "nickname",
		GoName:   "Nickname",
		Type:     &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
		Nullable: true,
	}

	result := gen.generateFieldSchema(field)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	// In 3.2 (like 3.1), nullable should NOT be set
	if schema.Nullable != nil {
		t.Error("nullable should not be set for 3.2")
	}

	// Type should be ["string", "null"]
	if len(schema.Type) != 2 {
		t.Errorf("type = %v, want [string, null]", schema.Type)
	}
}

func TestGenerator_ExclusiveMinMax_31(t *testing.T) {
	gen := NewGenerator("3.1")

	exMin := 0.0
	exMax := 100.0
	field := &resolver.Field{
		Name:             "score",
		GoName:           "Score",
		Type:             &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "number"},
		ExclusiveMinimum: &exMin,
		ExclusiveMaximum: &exMax,
	}

	result := gen.generateFieldSchema(field)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	// In 3.1, exclusiveMinimum is the numeric value itself
	if schema.ExclusiveMinimum == nil || schema.ExclusiveMinimum.B != 0 {
		t.Errorf("exclusiveMinimum = %v, want 0", schema.ExclusiveMinimum)
	}
	if schema.ExclusiveMaximum == nil || schema.ExclusiveMaximum.B != 100 {
		t.Errorf("exclusiveMaximum = %v, want 100", schema.ExclusiveMaximum)
	}

	// minimum/maximum should NOT be set (those are for non-exclusive bounds)
	if schema.Minimum != nil {
		t.Error("minimum should not be set for 3.1 exclusive bounds")
	}
	if schema.Maximum != nil {
		t.Error("maximum should not be set for 3.1 exclusive bounds")
	}
}

func TestGenerator_ReadOnly(t *testing.T) {
	gen := NewGenerator("3.1")

	field := &resolver.Field{
		Name:     "id",
		GoName:   "ID",
		Type:     &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
		ReadOnly: true,
	}

	result := gen.generateFieldSchema(field)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	if schema.ReadOnly == nil || *schema.ReadOnly != true {
		t.Error("readOnly should be true")
	}
}

func TestGenerator_WriteOnly(t *testing.T) {
	gen := NewGenerator("3.1")

	field := &resolver.Field{
		Name:      "password",
		GoName:    "Password",
		Type:      &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
		WriteOnly: true,
	}

	result := gen.generateFieldSchema(field)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	if schema.WriteOnly == nil || *schema.WriteOnly != true {
		t.Error("writeOnly should be true")
	}
}

func TestGenerator_NullableSchemaRef_31(t *testing.T) {
	gen := NewGenerator("3.1")
	field := &resolver.Field{
		Name:     "address",
		GoName:   "Address",
		GoType:   "Address",
		Type:     &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "Address"},
		Nullable: true,
	}

	result := gen.generateFieldSchema(field)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	// Should have oneOf with $ref and null type
	if len(schema.OneOf) != 2 {
		t.Fatalf("oneOf should have 2 entries, got %d", len(schema.OneOf))
	}

	// First entry should be the $ref
	ref := schema.OneOf[0].GetReference()
	if ref != "#/components/schemas/Address" {
		t.Errorf("oneOf[0] should be $ref to Address, got %q", ref)
	}

	// Second entry should be null type
	nullSchema, err := schema.OneOf[1].BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() for null entry error = %v", err)
	}
	if len(nullSchema.Type) != 1 || nullSchema.Type[0] != "null" {
		t.Errorf("oneOf[1] should be type null, got %v", nullSchema.Type)
	}

	// Should NOT have nullable keyword (3.1 uses oneOf)
	if schema.Nullable != nil {
		t.Error("nullable should not be set for 3.1")
	}
}

func TestGenerator_NonNullableSchemaRef(t *testing.T) {
	gen := NewGenerator("3.1")
	field := &resolver.Field{
		Name:     "address",
		GoName:   "Address",
		GoType:   "Address",
		Type:     &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "Address"},
		Nullable: false,
	}

	result := gen.generateFieldSchema(field)

	// Non-nullable schema ref should be a bare $ref
	// A bare $ref proxy returns the reference string, not a built schema
	ref := result.GetReference()
	if ref == "" {
		t.Error("non-nullable ref should be a bare $ref proxy")
	}
	if ref != "#/components/schemas/Address" {
		t.Errorf("ref = %q, want %q", ref, "#/components/schemas/Address")
	}
}

func TestGenerator_OpenAPIVersions(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"3.1", "3.1.0"},
		{"3.2", "3.2.0"},
		{"unknown", "3.1.0"}, // the CLI rejects these; 3.1 is the fallback
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			gen := NewGenerator(tt.version)
			got := gen.getOpenAPIVersion()
			if got != tt.want {
				t.Errorf("getOpenAPIVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
