package generator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/resolver"
	"go.yaml.in/yaml/v4"
)

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator("3.0")
	if gen == nil {
		t.Fatal("NewGenerator() returned nil")
	}

	if gen.version != "3.0" {
		t.Errorf("version = %s, want 3.0", gen.version)
	}
}

func TestGenerator_Generate(t *testing.T) {
	pkg := &resolver.ResolvedPackage{
		API: &resolver.ResolvedAPI{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Schemas: map[string]*resolver.ResolvedSchema{
			"User": {
				Name: "User",
				Fields: []*resolver.ResolvedField{
					{
						Name:        "id",
						GoName:      "ID",
						OpenAPIType: "string",
						Required:    true,
					},
				},
			},
		},
		Parameters: map[string]*resolver.ResolvedParameter{},
		Endpoints: []*resolver.ResolvedEndpoint{
			{
				Method: "GET",
				Path:   "/users",
				Responses: map[string]*resolver.ResolvedResponse{
					"200": {StatusCode: "200", Description: "Success"},
				},
			},
		},
	}

	gen := NewGenerator("3.0")
	spec, err := gen.Generate(pkg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if spec == nil {
		t.Fatal("Generate() returned nil spec")
	}

	// Verify openapi version
	if spec.Version != "3.0.3" {
		t.Errorf("openapi = %v, want 3.0.3", spec.Version)
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
	api := &resolver.ResolvedAPI{
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

	gen := NewGenerator("3.0")
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

	gen := NewGenerator("3.0")
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
	schemas := map[string]*resolver.ResolvedSchema{
		"User": {
			Name:        "User",
			Description: "User model",
			Fields: []*resolver.ResolvedField{
				{
					Name:        "id",
					GoName:      "ID",
					OpenAPIType: "string",
					Format:      "uuid",
					Required:    true,
					Description: "User ID",
				},
				{
					Name:        "email",
					GoName:      "Email",
					OpenAPIType: "string",
					Format:      "email",
					Required:    true,
				},
			},
		},
	}

	gen := NewGenerator("3.0")
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
	pkg := &resolver.ResolvedPackage{
		API: &resolver.ResolvedAPI{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Schemas:    map[string]*resolver.ResolvedSchema{},
		Parameters: map[string]*resolver.ResolvedParameter{},
		Endpoints: []*resolver.ResolvedEndpoint{
			{
				Method: "GET",
				Path:   "/test",
				Responses: map[string]*resolver.ResolvedResponse{
					"200": {StatusCode: "200", Description: "OK"},
				},
			},
		},
	}

	gen := NewGenerator("3.0")
	doc, err := gen.Generate(pkg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := gen.Render(doc, FormatJSON)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
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
	pkg := &resolver.ResolvedPackage{
		API: &resolver.ResolvedAPI{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Schemas:    map[string]*resolver.ResolvedSchema{},
		Parameters: map[string]*resolver.ResolvedParameter{},
		Endpoints: []*resolver.ResolvedEndpoint{
			{
				Method: "GET",
				Path:   "/test",
				Responses: map[string]*resolver.ResolvedResponse{
					"200": {StatusCode: "200", Description: "OK"},
				},
			},
		},
	}

	gen := NewGenerator("3.0")
	doc, err := gen.Generate(pkg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := gen.Render(doc, FormatYAML)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
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
	endpoints := []*resolver.ResolvedEndpoint{
		{
			Method:  "GET",
			Path:    "/users",
			Summary: "List users",
			Responses: map[string]*resolver.ResolvedResponse{
				"200": {StatusCode: "200", Description: "Success"},
			},
		},
		{
			Method:  "POST",
			Path:    "/users",
			Summary: "Create user",
			Responses: map[string]*resolver.ResolvedResponse{
				"201": {StatusCode: "201", Description: "Created"},
			},
		},
	}

	gen := NewGenerator("3.0")
	paths := gen.generatePaths(endpoints, map[string]*resolver.ResolvedParameter{}, map[string]*resolver.ResolvedSchema{})

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
	endpoint := &resolver.ResolvedEndpoint{
		Method:      "GET",
		Path:        "/users/{id}",
		Summary:     "Get user",
		Description: "Get user by ID",
		OperationID: "getUser",
		Tags:        []string{"users"},
		PathParams: []*resolver.ResolvedParameter{
			{
				Name: "UserIDPath",
				Fields: []*resolver.ResolvedField{
					{
						Name:        "id",
						GoName:      "ID",
						OpenAPIType: "string",
						Required:    true,
					},
				},
			},
		},
		Responses: map[string]*resolver.ResolvedResponse{
			"200": {
				StatusCode:  "200",
				Description: "Success",
				ContentType: "application/json",
				Body:        &resolver.ResolvedBody{Schema: "User", ElementType: "User"},
			},
		},
	}

	paramMap := map[string]*resolver.ResolvedParameter{
		"UserIDPath": endpoint.PathParams[0],
	}

	schemas := map[string]*resolver.ResolvedSchema{
		"User": {Name: "User"},
	}

	gen := NewGenerator("3.0")
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
	endpoints := []*resolver.ResolvedEndpoint{
		{
			Method: "GET",
			Path:   "/test",
			Responses: map[string]*resolver.ResolvedResponse{
				"200": {StatusCode: "200", Description: "OK"},
			},
		},
		{
			Method: "POST",
			Path:   "/test",
			Responses: map[string]*resolver.ResolvedResponse{
				"201": {StatusCode: "201", Description: "Created"},
			},
		},
	}

	gen := NewGenerator("3.0")
	paths := gen.generatePaths(endpoints, map[string]*resolver.ResolvedParameter{}, map[string]*resolver.ResolvedSchema{})

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
	gen := NewGenerator("3.0")

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
	gen := NewGenerator("3.0")

	pkg := &resolver.ResolvedPackage{
		API: &resolver.ResolvedAPI{
			Title:   "Test API",
			Version: "1.0.0",
			Tags: []*resolver.Tag{
				{Name: "pets", Description: "Pet operations"},
				{Name: "users", Description: "User operations"},
			},
		},
		Schemas:    map[string]*resolver.ResolvedSchema{},
		Parameters: map[string]*resolver.ResolvedParameter{},
		Endpoints: []*resolver.ResolvedEndpoint{
			{
				Method: "GET",
				Path:   "/pets",
				Tags:   []string{"pets"},
				Responses: map[string]*resolver.ResolvedResponse{
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
	gen := NewGenerator("3.0")

	body := &resolver.ResolvedBody{
		Schema:      "User",
		ElementType: "User",
	}

	schemas := map[string]*resolver.ResolvedSchema{}

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
	gen := NewGenerator("3.0")

	wrapperSchema := &resolver.ResolvedSchema{
		Name: "DataResponse",
		Fields: []*resolver.ResolvedField{
			{Name: "status", GoName: "Status", OpenAPIType: "string", Required: true},
			{Name: "data", GoName: "Data", OpenAPIType: "object", Required: true},
		},
	}

	body := &resolver.ResolvedBody{
		Schema:      "User",
		ElementType: "User",
		Bind: &resolver.ResolvedBindTarget{
			Wrapper:       "DataResponse",
			Field:         "Data",
			WrapperSchema: wrapperSchema,
		},
	}

	schemas := map[string]*resolver.ResolvedSchema{
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

func TestGenerator_GenerateSchemaRef_Simple(t *testing.T) {
	gen := NewGenerator("3.0")

	result := gen.generateSchemaRef("User", false, false, "User")

	if !result.IsReference() {
		t.Error("Expected schema reference")
	}

	if result.GetReference() != "#/components/schemas/User" {
		t.Errorf("$ref = %v, want #/components/schemas/User", result.GetReference())
	}
}

func TestGenerator_GenerateSchemaRef_Array(t *testing.T) {
	gen := NewGenerator("3.0")

	result := gen.generateSchemaRef("[]User", true, false, "User")

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

func TestGenerator_GenerateSchemaRef_Map(t *testing.T) {
	gen := NewGenerator("3.0")

	result := gen.generateSchemaRef("map[string]User", false, true, "User")

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

func TestGenerator_GenerateSchemaRef_Primitive(t *testing.T) {
	gen := NewGenerator("3.0")

	result := gen.generateSchemaRef("int", false, false, "int")

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	if len(schema.Type) == 0 || schema.Type[0] != "integer" {
		t.Errorf("type = %v, want integer", schema.Type)
	}
}

func TestIsPrimitive(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"string", true},
		{"int", true},
		{"int64", true},
		{"float64", true},
		{"bool", true},
		{"User", false},
		{"CustomType", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isPrimitive(tt.input)
			if got != tt.want {
				t.Errorf("isPrimitive(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGoTypeToPrimitive(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"string", "string"},
		{"int", "integer"},
		{"int64", "integer"},
		{"float64", "number"},
		{"bool", "boolean"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := goTypeToPrimitive(tt.input)
			if got != tt.want {
				t.Errorf("goTypeToPrimitive(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerator_GenerateInlineWrappedSchema(t *testing.T) {
	gen := NewGenerator("3.0")

	wrapperSchema := &resolver.ResolvedSchema{
		Name: "DataResponse",
		Fields: []*resolver.ResolvedField{
			{Name: "status", GoName: "Status", OpenAPIType: "string", Required: true},
			{Name: "data", GoName: "Data", OpenAPIType: "object", Required: true},
		},
	}

	// Create an inline body with bind
	inline := &resolver.ResolvedInlineBody{
		ContentType: "application/json",
		Fields: []*resolver.ResolvedField{
			{Name: "id", GoName: "ID", OpenAPIType: "string", Required: true},
			{Name: "email", GoName: "Email", OpenAPIType: "string", Required: true},
		},
		Bind: &resolver.ResolvedBindTarget{
			Wrapper:       "DataResponse",
			Field:         "Data",
			WrapperSchema: wrapperSchema,
		},
	}

	schemas := map[string]*resolver.ResolvedSchema{
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
	gen := NewGenerator("3.0")

	// Create an inline body without bind - should fall back to inline schema
	inline := &resolver.ResolvedInlineBody{
		ContentType: "application/json",
		Fields: []*resolver.ResolvedField{
			{Name: "id", GoName: "ID", OpenAPIType: "string", Required: true},
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

func TestConvertEnumValues(t *testing.T) {
	tests := []struct {
		name        string
		values      []string
		openAPIType string
		want        []any
	}{
		{
			name:        "string enum",
			values:      []string{"active", "pending", "done"},
			openAPIType: "string",
			want:        []any{"active", "pending", "done"},
		},
		{
			name:        "integer enum",
			values:      []string{"1", "2", "3"},
			openAPIType: "integer",
			want:        []any{int64(1), int64(2), int64(3)},
		},
		{
			name:        "integer enum with invalid value",
			values:      []string{"1", "invalid", "3"},
			openAPIType: "integer",
			want:        []any{int64(1), "invalid", int64(3)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertEnumValues(tt.values, tt.openAPIType)
			if len(got) != len(tt.want) {
				t.Errorf("convertEnumValues() returned %d values, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("convertEnumValues()[%d] = %v (%T), want %v (%T)", i, v, v, tt.want[i], tt.want[i])
				}
			}
		})
	}
}

func TestGenerator_GenerateFieldSchema_ArrayEnum(t *testing.T) {
	gen := NewGenerator("3.0")

	field := &resolver.ResolvedField{
		Name:        "tags",
		GoName:      "Tags",
		OpenAPIType: "array",
		IsArray:     true,
		ItemsType:   "string",
		Enum:        []string{"red", "green", "blue"},
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
	gen := NewGenerator("3.0")

	field := &resolver.ResolvedField{
		Name:        "priority",
		GoName:      "Priority",
		OpenAPIType: "integer",
		Enum:        []string{"1", "2", "3"},
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
	gen := NewGenerator("3.0")

	field := &resolver.ResolvedField{
		Name:        "status",
		GoName:      "Status",
		OpenAPIType: "array",
		IsArray:     true,
		ItemsType:   "string",
		Enum:        []string{"active", "pending", "done"},
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

func TestGenerator_Version30Nullable(t *testing.T) {
	gen := NewGenerator("3.0")

	field := &resolver.ResolvedField{
		Name:        "nickname",
		GoName:      "Nickname",
		OpenAPIType: "string",
		Nullable:    true,
	}

	result := gen.generateFieldSchema(field)

	schema, err := result.BuildSchema()
	if err != nil {
		t.Fatalf("BuildSchema() error = %v", err)
	}

	// In 3.0, nullable should be set to true
	if schema.Nullable == nil || *schema.Nullable != true {
		t.Error("nullable should be true for 3.0")
	}

	// Type should be just "string"
	if len(schema.Type) != 1 || schema.Type[0] != "string" {
		t.Errorf("type = %v, want [string]", schema.Type)
	}
}

func TestGenerator_Version31Nullable(t *testing.T) {
	gen := NewGenerator("3.1")

	field := &resolver.ResolvedField{
		Name:        "nickname",
		GoName:      "Nickname",
		OpenAPIType: "string",
		Nullable:    true,
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

	field := &resolver.ResolvedField{
		Name:        "nickname",
		GoName:      "Nickname",
		OpenAPIType: "string",
		Nullable:    true,
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

func TestGenerator_OpenAPIVersions(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"3.0", "3.0.3"},
		{"3.1", "3.1.0"},
		{"3.2", "3.2.0"},
		{"unknown", "3.0.3"}, // defaults to 3.0
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
