package validator

import (
	"strings"
	"testing"

	"github.com/wontaeyang/go-specgen/pkg/resolver"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator() returned nil")
	}

	if v.errs.Stage != "validation" {
		t.Errorf("Validator.errs.Stage = %q, want %q", v.errs.Stage, "validation")
	}
}

func TestValidator_Validate_ValidPackage(t *testing.T) {
	pkg := &resolver.Package{
		PackageName: "test",
		API: &resolver.API{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Schemas: map[string]*resolver.Schema{
			"User": {
				Name:       "User",
				GoTypeName: "User",
				Fields: []*resolver.Field{
					{
						Name:   "id",
						GoName: "ID",
						Type:   &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
					},
				},
				// Hand-built packages skip the resolver, so the direction
				// chain the resolver would have recorded is set directly.
				ResponseChain: "@endpoint[GET /users].@response[200] → User",
			},
		},
		Parameters: map[string]*resolver.ParameterStruct{},
		Endpoints: []*resolver.Endpoint{
			{
				Method: "GET",
				Path:   "/users",
				Responses: []*resolver.Response{
					{StatusCode: "200",
						Description: "Success"},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(pkg)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

// TestValidator_Validate_MissingAPI checks that a nil API is recorded and the
// rest of the package still gets validated. The endpoint is what makes it a
// real test: validateEndpoint reads the API's tags, and reading them without a
// guard turned this case into a panic instead of the error it promises.
func TestValidator_Validate_MissingAPI(t *testing.T) {
	pkg := &resolver.Package{
		PackageName: "test",
		API:         nil,
		Schemas:     map[string]*resolver.Schema{},
		Parameters:  map[string]*resolver.ParameterStruct{},
		Endpoints: []*resolver.Endpoint{
			{
				Method:    "GET",
				Path:      "/widgets",
				Tags:      []string{"widgets"},
				Responses: []*resolver.Response{{StatusCode: "200"}},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(pkg)
	if err == nil {
		t.Fatal("Validate() should error when API is missing")
	}

	if !strings.Contains(err.Error(), "@api") {
		t.Errorf("Error should mention @api, got: %v", err)
	}
	if !strings.Contains(err.Error(), "undefined tag: widgets") {
		t.Errorf("Error should report the endpoint's tag as undefined, got: %v", err)
	}
}

func TestValidator_ValidateAPI_MissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		api     *resolver.API
		wantErr string
	}{
		{
			name: "missing title",
			api: &resolver.API{
				Version: "1.0.0",
			},
			wantErr: "@title",
		},
		{
			name: "missing version",
			api: &resolver.API{
				Title: "Test API",
			},
			wantErr: "@version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.Package{
				API:        tt.api,
				Schemas:    map[string]*resolver.Schema{},
				Parameters: map[string]*resolver.ParameterStruct{},
				Endpoints:  []*resolver.Endpoint{},
			}

			v := NewValidator()
			err := v.Validate(pkg)
			if err == nil {
				t.Error("Validate() should error")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error should mention %s, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidator_ValidateSecurityScheme(t *testing.T) {
	tests := []struct {
		name    string
		scheme  *resolver.SecurityScheme
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid http scheme",
			scheme: &resolver.SecurityScheme{
				Type:   "http",
				Scheme: "bearer",
			},
			wantErr: false,
		},
		{
			name: "http scheme missing @scheme",
			scheme: &resolver.SecurityScheme{
				Type: "http",
			},
			wantErr: true,
			errMsg:  "@scheme",
		},
		{
			name: "valid apiKey scheme",
			scheme: &resolver.SecurityScheme{
				Type:          "apiKey",
				In:            "header",
				ParameterName: "X-API-Key",
			},
			wantErr: false,
		},
		{
			name: "apiKey missing @in",
			scheme: &resolver.SecurityScheme{
				Type:          "apiKey",
				ParameterName: "X-API-Key",
			},
			wantErr: true,
			errMsg:  "@in",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.Package{
				API: &resolver.API{
					Title:   "Test",
					Version: "1.0.0",
					SecuritySchemes: map[string]*resolver.SecurityScheme{
						"test": tt.scheme,
					},
				},
				Schemas:    map[string]*resolver.Schema{},
				Parameters: map[string]*resolver.ParameterStruct{},
				Endpoints:  []*resolver.Endpoint{},
			}

			v := NewValidator()
			err := v.Validate(pkg)

			if tt.wantErr && err == nil {
				t.Error("Validate() should error")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should mention %s, got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidator_ValidateSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  *resolver.Schema
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid schema",
			schema: &resolver.Schema{
				Name: "User",
				Fields: []*resolver.Field{
					{Name: "id", GoName: "ID", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
				},
				ResponseChain: "@endpoint[GET /users].@response[200] → User",
			},
			wantErr: false,
		},
		{
			name: "schema with no fields",
			schema: &resolver.Schema{
				Name:          "Empty",
				Fields:        []*resolver.Field{},
				ResponseChain: "@endpoint[GET /empty].@response[200] → Empty",
			},
			wantErr: true,
			errMsg:  "no fields",
		},
		{
			name: "duplicate field names",
			schema: &resolver.Schema{
				Name: "User",
				Fields: []*resolver.Field{
					{Name: "id", GoName: "ID", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
					{Name: "id", GoName: "Id", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
				},
				ResponseChain: "@endpoint[GET /users].@response[200] → User",
			},
			wantErr: true,
			errMsg:  "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.Package{
				API: &resolver.API{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas: map[string]*resolver.Schema{
					tt.schema.Name: tt.schema,
				},
				Parameters: map[string]*resolver.ParameterStruct{},
				Endpoints:  []*resolver.Endpoint{},
			}

			v := NewValidator()
			err := v.Validate(pkg)

			if tt.wantErr && err == nil {
				t.Error("Validate() should error")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should mention %s, got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidator_ValidateField(t *testing.T) {
	minVal := 0.0
	maxVal := 100.0
	minLen := 5
	maxLen := 50

	tests := []struct {
		name    string
		field   *resolver.Field
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid field",
			field: &resolver.Field{
				Name:   "age",
				GoName: "Age",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "integer"},
			},
			wantErr: false,
		},
		{
			name: "enum on integer",
			field: &resolver.Field{
				Name:   "age",
				GoName: "Age",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "integer"},
				Enum:   []string{"1", "2", "3"},
			},
			wantErr: false,
		},
		{
			name: "enum on array of strings",
			field: &resolver.Field{
				Name:   "tags",
				GoName: "Tags",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},

				Enum: []string{"red", "green", "blue"},
			},
			wantErr: false,
		},
		{
			name: "enum on array of integers",
			field: &resolver.Field{
				Name:   "levels",
				GoName: "Levels",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},

				Enum: []string{"1", "2", "3"},
			},
			wantErr: false,
		},
		{
			name: "enum on boolean",
			field: &resolver.Field{
				Name:   "active",
				GoName: "Active",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "boolean"},
				Enum:   []string{"true", "false"},
			},
			wantErr: true,
			errMsg:  "enum only supported for string, integer, or array types",
		},
		{
			name: "enum on array of objects",
			field: &resolver.Field{
				Name:   "items",
				GoName: "Items",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeObject}},
				Enum:   []string{"a", "b"},
			},
			wantErr: true,
			errMsg:  "enum for arrays only supported with string or integer items",
		},
		{
			name: "min > max",
			field: &resolver.Field{
				Name:    "value",
				GoName:  "Value",
				Type:    &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "number"},
				Minimum: &maxVal,
				Maximum: &minVal,
			},
			wantErr: true,
			errMsg:  "minimum cannot be greater than maximum",
		},
		{
			name: "minLength > maxLength",
			field: &resolver.Field{
				Name:      "text",
				GoName:    "Text",
				Type:      &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
				MinLength: &maxLen,
				MaxLength: &minLen,
			},
			wantErr: true,
			errMsg:  "minLength cannot be greater than maxLength",
		},
		{
			name: "length constraints on non-string",
			field: &resolver.Field{
				Name:      "count",
				GoName:    "Count",
				Type:      &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "integer"},
				MinLength: &minLen,
			},
			wantErr: true,
			errMsg:  "only valid for string",
		},
		{
			name: "pattern on non-string",
			field: &resolver.Field{
				Name:    "count",
				GoName:  "Count",
				Type:    &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "integer"},
				Pattern: "^[0-9]+$",
			},
			wantErr: true,
			errMsg:  "only valid for string",
		},
		{
			name: "invalid pattern regex",
			field: &resolver.Field{
				Name:    "text",
				GoName:  "Text",
				Type:    &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
				Pattern: "[invalid",
			},
			wantErr: true,
			errMsg:  "invalid pattern",
		},
		{
			name: "minItems > maxItems",
			field: &resolver.Field{
				Name:   "tags",
				GoName: "Tags",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},

				MinItems: &maxLen,
				MaxItems: &minLen,
			},
			wantErr: true,
			errMsg:  "minItems cannot be greater than maxItems",
		},
		{
			name: "items constraints on non-array",
			field: &resolver.Field{
				Name:     "name",
				GoName:   "Name",
				Type:     &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
				MinItems: &minLen,
			},
			wantErr: true,
			errMsg:  "only valid for array",
		},
		{
			name: "valid array with minItems and maxItems",
			field: &resolver.Field{
				Name:   "tags",
				GoName: "Tags",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},

				MinItems: &minLen,
				MaxItems: &maxLen,
			},
			wantErr: false,
		},
		{
			name: "uniqueItems on non-array",
			field: &resolver.Field{
				Name:        "name",
				GoName:      "Name",
				Type:        &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
				UniqueItems: true,
			},
			wantErr: true,
			errMsg:  "only valid for array",
		},
		{
			name: "valid array with uniqueItems",
			field: &resolver.Field{
				Name:   "tags",
				GoName: "Tags",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},

				UniqueItems: true,
			},
			wantErr: false,
		},
		{
			name: "readOnly and writeOnly both true",
			field: &resolver.Field{
				Name:      "field",
				GoName:    "Field",
				Type:      &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
				ReadOnly:  true,
				WriteOnly: true,
			},
			wantErr: true,
			errMsg:  "readOnly and writeOnly cannot both be true",
		},
		{
			name: "readOnly only",
			field: &resolver.Field{
				Name:     "id",
				GoName:   "ID",
				Type:     &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
				ReadOnly: true,
			},
			wantErr: false,
		},
		{
			name: "writeOnly only",
			field: &resolver.Field{
				Name:      "password",
				GoName:    "Password",
				Type:      &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
				WriteOnly: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.Package{
				API: &resolver.API{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas: map[string]*resolver.Schema{
					"Test": {
						Name:          "Test",
						Fields:        []*resolver.Field{tt.field},
						ResponseChain: "@endpoint[GET /tests].@response[200] → Test",
					},
				},
				Parameters: map[string]*resolver.ParameterStruct{},
				Endpoints:  []*resolver.Endpoint{},
			}

			v := NewValidator()
			err := v.Validate(pkg)

			if tt.wantErr && err == nil {
				t.Error("Validate() should error")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should mention %s, got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidator_ValidateParameterField(t *testing.T) {
	tests := []struct {
		name      string
		paramType string
		field     *resolver.Field
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "path param cannot be nullable",
			paramType: "path",
			field: &resolver.Field{
				Name:     "id",
				GoName:   "ID",
				Type:     &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"},
				Nullable: true,
			},
			wantErr: true,
			errMsg:  "cannot be nullable",
		},
		{
			name:      "path param cannot be array",
			paramType: "path",
			field: &resolver.Field{
				Name:   "id",
				GoName: "ID",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
			},
			wantErr: true,
			errMsg:  "cannot be arrays",
		},
		{
			name:      "header param cannot be array",
			paramType: "header",
			field: &resolver.Field{
				Name:   "token",
				GoName: "Token",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
			},
			wantErr: true,
			errMsg:  "cannot be arrays",
		},
		{
			name:      "cookie param cannot be array",
			paramType: "cookie",
			field: &resolver.Field{
				Name:   "session",
				GoName: "Session",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
			},
			wantErr: true,
			errMsg:  "cannot be arrays",
		},
		{
			name:      "query param can be array",
			paramType: "query",
			field: &resolver.Field{
				Name:   "tags",
				GoName: "Tags",
				Type:   &resolver.TypeRef{Shape: resolver.ShapeArray, Elem: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.Package{
				API: &resolver.API{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas: map[string]*resolver.Schema{},
				Parameters: map[string]*resolver.ParameterStruct{
					"TestParam": {
						Name:   "TestParam",
						Type:   tt.paramType,
						Fields: []*resolver.Field{tt.field},
					},
				},
				Endpoints: []*resolver.Endpoint{},
			}

			v := NewValidator()
			err := v.Validate(pkg)

			if tt.wantErr && err == nil {
				t.Error("Validate() should error")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should mention %s, got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidator_ValidateEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *resolver.Endpoint
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid endpoint",
			endpoint: &resolver.Endpoint{
				Method: "GET",
				Path:   "/users",
				Responses: []*resolver.Response{
					{StatusCode: "200"},
				},
			},
			wantErr: false,
		},
		{
			// The eighth Path Item operation, and the one the generator could
			// emit but the validator used to reject.
			name: "TRACE is a valid method",
			endpoint: &resolver.Endpoint{
				Method: "TRACE",
				Path:   "/users",
				Responses: []*resolver.Response{
					{StatusCode: "200"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid method",
			endpoint: &resolver.Endpoint{
				Method: "INVALID",
				Path:   "/users",
				Responses: []*resolver.Response{
					{StatusCode: "200"},
				},
			},
			wantErr: true,
			errMsg:  "invalid HTTP method",
		},
		{
			name: "missing path",
			endpoint: &resolver.Endpoint{
				Method: "GET",
				Path:   "",
				Responses: []*resolver.Response{
					{StatusCode: "200"},
				},
			},
			wantErr: true,
			errMsg:  "missing path",
		},
		{
			name: "path not starting with /",
			endpoint: &resolver.Endpoint{
				Method: "GET",
				Path:   "users",
				Responses: []*resolver.Response{
					{StatusCode: "200"},
				},
			},
			wantErr: true,
			errMsg:  "must start with /",
		},
		{
			name: "no responses",
			endpoint: &resolver.Endpoint{
				Method:    "GET",
				Path:      "/users",
				Responses: []*resolver.Response{},
			},
			wantErr: true,
			errMsg:  "at least one response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.Package{
				API: &resolver.API{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas:    map[string]*resolver.Schema{},
				Parameters: map[string]*resolver.ParameterStruct{},
				Endpoints:  []*resolver.Endpoint{tt.endpoint},
			}

			v := NewValidator()
			err := v.Validate(pkg)

			if tt.wantErr && err == nil {
				t.Error("Validate() should error")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should mention %s, got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestExtractPathVariables(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "no variables",
			path:     "/users",
			expected: []string{},
		},
		{
			name:     "single variable",
			path:     "/users/{id}",
			expected: []string{"id"},
		},
		{
			name:     "multiple variables",
			path:     "/users/{userId}/posts/{postId}",
			expected: []string{"userId", "postId"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathVariables(tt.path)
			if len(got) != len(tt.expected) {
				t.Errorf("extractPathVariables() returned %d vars, want %d", len(got), len(tt.expected))
				return
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("extractPathVariables()[%d] = %s, want %s", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestValidator_ValidatePathParameters(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		params  []*resolver.Parameter
		wantErr bool
		errMsg  string
	}{
		{
			name: "matching path variable and parameter",
			path: "/users/{id}",
			params: []*resolver.Parameter{
				{In: "path", Field: &resolver.Field{Name: "id", GoName: "ID"}},
			},
			wantErr: false,
		},
		{
			name:    "path variable without parameter",
			path:    "/users/{id}",
			params:  []*resolver.Parameter{},
			wantErr: true,
			errMsg:  "has no corresponding @path parameter",
		},
		{
			name: "parameter not used in path",
			path: "/users",
			params: []*resolver.Parameter{
				{In: "path", Field: &resolver.Field{Name: "id", GoName: "ID"}},
			},
			wantErr: true,
			errMsg:  "not used in path",
		},
		{
			name: "query parameter is not a path parameter",
			path: "/users",
			params: []*resolver.Parameter{
				{In: "query", Field: &resolver.Field{Name: "id", GoName: "ID"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.Package{
				API: &resolver.API{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas:    map[string]*resolver.Schema{},
				Parameters: map[string]*resolver.ParameterStruct{},
				Endpoints: []*resolver.Endpoint{
					{
						Method:     "GET",
						Path:       tt.path,
						Parameters: tt.params,
						Responses: []*resolver.Response{
							{StatusCode: "200"},
						},
					},
				},
			}

			v := NewValidator()
			err := v.Validate(pkg)

			if tt.wantErr && err == nil {
				t.Error("Validate() should error")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should mention %s, got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidator_ValidateRequestBody(t *testing.T) {
	tests := []struct {
		name    string
		request *resolver.RequestBody
		schemas map[string]*resolver.Schema
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: &resolver.RequestBody{
				ContentType: "application/json",
				Body:        &resolver.Body{Schema: "User"},
			},
			schemas: map[string]*resolver.Schema{
				"User": {
					Name: "User",
					Fields: []*resolver.Field{
						{Name: "id", GoName: "ID", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
					},
					RequestChain: "@endpoint[POST /users].@request → User",
				},
			},
			wantErr: false,
		},
		{
			name: "missing content type",
			request: &resolver.RequestBody{
				Body: &resolver.Body{Schema: "User"},
			},
			schemas: map[string]*resolver.Schema{
				"User": {Name: "User", RequestChain: "@endpoint[POST /users].@request → User"},
			},
			wantErr: true,
			errMsg:  "@contentType",
		},
		{
			name: "unknown schema",
			request: &resolver.RequestBody{
				ContentType: "application/json",
				Body:        &resolver.Body{Schema: "Unknown", Type: &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "Unknown"}},
			},
			schemas: map[string]*resolver.Schema{},
			wantErr: true,
			errMsg:  "unknown schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.Package{
				API: &resolver.API{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas:    tt.schemas,
				Parameters: map[string]*resolver.ParameterStruct{},
				Endpoints: []*resolver.Endpoint{
					{
						Method:  "POST",
						Path:    "/users",
						Request: tt.request,
						Responses: []*resolver.Response{
							{StatusCode: "200"},
						},
					},
				},
			}

			v := NewValidator()
			err := v.Validate(pkg)

			if tt.wantErr && err == nil {
				t.Error("Validate() should error")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should mention %s, got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidator_ValidateResponseStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode string
		wantErr    bool
	}{
		{"exact 200", "200", false},
		{"exact 404", "404", false},
		{"exact 500", "500", false},
		{"range 2XX", "2XX", false},
		{"range 4XX", "4XX", false},
		{"range 5XX", "5XX", false},
		{"default", "default", false},
		{"invalid letters", "abc", true},
		{"invalid 6XX", "6XX", true},
		{"invalid 0XX", "0XX", true},
		{"too short", "20", true},
		{"too long", "2000", true},
		{"lowercase xx", "2xx", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.Package{
				API: &resolver.API{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas:    map[string]*resolver.Schema{},
				Parameters: map[string]*resolver.ParameterStruct{},
				Endpoints: []*resolver.Endpoint{
					{
						Method: "GET",
						Path:   "/test",
						Responses: []*resolver.Response{
							{StatusCode: tt.statusCode},
						},
					},
				},
			}

			v := NewValidator()
			err := v.Validate(pkg)

			if tt.wantErr && (err == nil || !strings.Contains(err.Error(), "invalid status code")) {
				t.Errorf("expected 'invalid status code' error for %q, got: %v", tt.statusCode, err)
			}
			if !tt.wantErr && err != nil && strings.Contains(err.Error(), "invalid status code") {
				t.Errorf("unexpected 'invalid status code' error for %q: %v", tt.statusCode, err)
			}
		})
	}
}

func TestValidator_ValidateParameterRefs(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *resolver.Endpoint
		want     string
	}{
		{
			name: "known reference",
			endpoint: &resolver.Endpoint{
				Method:    "GET",
				Path:      "/test",
				ParamRefs: []resolver.ParamRef{{Name: "ListQuery", In: "query"}},
				Responses: []*resolver.Response{{StatusCode: "200"}},
			},
		},
		{
			name: "unknown endpoint reference",
			endpoint: &resolver.Endpoint{
				Method:    "GET",
				Path:      "/test",
				ParamRefs: []resolver.ParamRef{{Name: "Missing", In: "query"}},
				Responses: []*resolver.Response{{StatusCode: "200"}},
			},
			want: "@query references unknown parameter struct: Missing",
		},
		{
			name: "unknown named response header reference",
			endpoint: &resolver.Endpoint{
				Method: "GET",
				Path:   "/test",
				Responses: []*resolver.Response{
					{StatusCode: "200", HeaderRefs: []string{"Missing"}},
				},
			},
			want: "@header references unknown parameter struct: Missing",
		},
		{
			name: "unknown inline response header reference",
			endpoint: &resolver.Endpoint{
				Method: "GET",
				Path:   "/test",
				Responses: []*resolver.Response{
					{
						StatusCode: "200",
						HeaderRefs: []string{"Missing"},
						Inline:     &resolver.InlineBody{HeaderRefs: []string{"Missing"}},
					},
				},
			},
			want: "@header references unknown parameter struct: Missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.Package{
				API:     &resolver.API{Title: "Test", Version: "1.0.0"},
				Schemas: map[string]*resolver.Schema{},
				Parameters: map[string]*resolver.ParameterStruct{
					"ListQuery": {Name: "ListQuery", Type: "query", Fields: []*resolver.Field{
						{Name: "limit", GoName: "Limit", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "integer"}},
					}},
				},
				Endpoints: []*resolver.Endpoint{tt.endpoint},
			}

			err := NewValidator().Validate(pkg)

			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected %q, got: %v", tt.want, err)
			}
		})
	}
}

func TestValidator_ValidateEndpointTags(t *testing.T) {
	tests := []struct {
		name         string
		endpointTags []string
		apiTags      []*resolver.Tag
		wantError    bool
		errorMessage string
	}{
		{
			name:         "valid tags",
			endpointTags: []string{"pets", "users"},
			apiTags: []*resolver.Tag{
				{Name: "pets", Description: "Pet operations"},
				{Name: "users", Description: "User operations"},
			},
			wantError: false,
		},
		{
			name:         "undefined tag",
			endpointTags: []string{"pets", "unknown"},
			apiTags: []*resolver.Tag{
				{Name: "pets", Description: "Pet operations"},
			},
			wantError:    true,
			errorMessage: "endpoint uses undefined tag: unknown",
		},
		{
			name:         "no API tags defined",
			endpointTags: []string{},
			apiTags:      []*resolver.Tag{},
			wantError:    false,
		},
		{
			name:         "endpoint uses no tags",
			endpointTags: []string{},
			apiTags: []*resolver.Tag{
				{Name: "pets", Description: "Pet operations"},
			},
			wantError: false,
		},
		{
			name:         "all endpoint tags undefined",
			endpointTags: []string{"unknown1", "unknown2"},
			apiTags: []*resolver.Tag{
				{Name: "pets", Description: "Pet operations"},
			},
			wantError:    true,
			errorMessage: "endpoint uses undefined tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.validateEndpointTags("@endpoint[GET /test]", tt.endpointTags, tt.apiTags)

			if tt.wantError && len(v.errs.Errors) == 0 {
				t.Errorf("expected error but got none")
			}

			if !tt.wantError && len(v.errs.Errors) > 0 {
				t.Errorf("expected no error but got: %v", v.errs.Errors)
			}

			if tt.wantError && len(v.errs.Errors) > 0 {
				found := false
				for _, err := range v.errs.Errors {
					if strings.Contains(err.Error(), tt.errorMessage) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q but got: %v", tt.errorMessage, v.errs.Errors)
				}
			}
		})
	}
}

func TestValidator_ValidateEndpointWithTags(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  *resolver.Endpoint
		pkg       *resolver.Package
		wantError bool
		errorMsg  string
	}{
		{
			name: "endpoint with valid tags",
			endpoint: &resolver.Endpoint{
				Method: "GET",
				Path:   "/pets",
				Tags:   []string{"pets"},
				Responses: []*resolver.Response{
					{StatusCode: "200", Description: "Success"},
				},
			},
			pkg: &resolver.Package{
				API: &resolver.API{
					Title:   "Test API",
					Version: "1.0.0",
					Tags: []*resolver.Tag{
						{Name: "pets", Description: "Pet operations"},
					},
				},
				Schemas:    map[string]*resolver.Schema{},
				Parameters: map[string]*resolver.ParameterStruct{},
			},
			wantError: false,
		},
		{
			name: "endpoint with invalid tag",
			endpoint: &resolver.Endpoint{
				Method: "GET",
				Path:   "/users",
				Tags:   []string{"users"},
				Responses: []*resolver.Response{
					{StatusCode: "200", Description: "Success"},
				},
			},
			pkg: &resolver.Package{
				API: &resolver.API{
					Title:   "Test API",
					Version: "1.0.0",
					Tags: []*resolver.Tag{
						{Name: "pets", Description: "Pet operations"},
					},
				},
				Schemas:    map[string]*resolver.Schema{},
				Parameters: map[string]*resolver.ParameterStruct{},
			},
			wantError: true,
			errorMsg:  "endpoint uses undefined tag: users",
		},
		{
			name: "endpoint without tags when API has tags",
			endpoint: &resolver.Endpoint{
				Method: "GET",
				Path:   "/health",
				Tags:   []string{},
				Responses: []*resolver.Response{
					{StatusCode: "200", Description: "Success"},
				},
			},
			pkg: &resolver.Package{
				API: &resolver.API{
					Title:   "Test API",
					Version: "1.0.0",
					Tags: []*resolver.Tag{
						{Name: "pets", Description: "Pet operations"},
					},
				},
				Schemas:    map[string]*resolver.Schema{},
				Parameters: map[string]*resolver.ParameterStruct{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.validateEndpoint(tt.endpoint, tt.pkg)

			if tt.wantError && len(v.errs.Errors) == 0 {
				t.Errorf("expected error but got none")
			}

			if !tt.wantError && len(v.errs.Errors) > 0 {
				t.Errorf("expected no error but got: %v", v.errs.Errors)
			}

			if tt.wantError && tt.errorMsg != "" && len(v.errs.Errors) > 0 {
				found := false
				for _, err := range v.errs.Errors {
					if strings.Contains(err.Error(), tt.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q but got: %v", tt.errorMsg, v.errs.Errors)
				}
			}
		})
	}
}

func TestValidator_ValidateBindTarget(t *testing.T) {
	wrapperSchema := &resolver.Schema{
		Name:       "DataResponse",
		GoTypeName: "DataResponse",
		Fields: []*resolver.Field{
			{Name: "data", GoName: "Data", Type: &resolver.TypeRef{Shape: resolver.ShapeObject}},
			{Name: "message", GoName: "Message", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}},
		},
	}

	tests := []struct {
		name    string
		bind    *resolver.BindTarget
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid bind target",
			bind: &resolver.BindTarget{
				Wrapper:       "DataResponse",
				Field:         "Data",
				WrapperSchema: wrapperSchema,
			},
			wantErr: false,
		},
		{
			name: "unknown wrapper schema",
			bind: &resolver.BindTarget{
				Wrapper:       "NonExistent",
				Field:         "Data",
				WrapperSchema: nil,
			},
			wantErr: true,
			errMsg:  "references unknown wrapper schema: NonExistent",
		},
		{
			name: "unknown field in wrapper",
			bind: &resolver.BindTarget{
				Wrapper:       "DataResponse",
				Field:         "Missing",
				WrapperSchema: wrapperSchema,
			},
			wantErr: true,
			errMsg:  `wrapper schema "DataResponse" has no field "Missing"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			schemas := map[string]*resolver.Schema{
				"DataResponse": wrapperSchema,
			}
			v.validateBindTarget("@endpoint[GET /users].@request", tt.bind, schemas)

			if tt.wantErr && len(v.errs.Errors) == 0 {
				t.Error("expected error but got none")
			}

			if !tt.wantErr && len(v.errs.Errors) > 0 {
				t.Errorf("expected no error but got: %v", v.errs.Errors)
			}

			if tt.wantErr && len(v.errs.Errors) > 0 {
				if !strings.Contains(v.errs.Errors[0].Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errMsg, v.errs.Errors[0])
				}
			}
		})
	}
}

func TestValidator_ValidateBindTarget_RequestBody(t *testing.T) {
	wrapperSchema := &resolver.Schema{
		Name:       "DataResponse",
		GoTypeName: "DataResponse",
		Fields: []*resolver.Field{
			{Name: "data", GoName: "Data", Type: &resolver.TypeRef{Shape: resolver.ShapeObject}},
		},
	}

	pkg := &resolver.Package{
		API: &resolver.API{
			Title:   "Test",
			Version: "1.0.0",
		},
		Schemas: map[string]*resolver.Schema{
			"User":         {Name: "User", Fields: []*resolver.Field{{Name: "id", GoName: "ID", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}}}},
			"DataResponse": wrapperSchema,
		},
		Parameters: map[string]*resolver.ParameterStruct{},
		Endpoints: []*resolver.Endpoint{
			{
				Method: "POST",
				Path:   "/users",
				Request: &resolver.RequestBody{
					ContentType: "application/json",
					Body: &resolver.Body{
						Schema: "User",
						Type:   &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "User"},
						Bind: &resolver.BindTarget{
							Wrapper:       "NonExistent",
							Field:         "Data",
							WrapperSchema: nil,
						},
					},
				},
				Responses: []*resolver.Response{
					{StatusCode: "200"},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(pkg)
	if err == nil {
		t.Fatal("expected validation error for unknown bind wrapper")
	}
	if !strings.Contains(err.Error(), "references unknown wrapper schema: NonExistent") {
		t.Errorf("expected error about unknown wrapper schema, got: %v", err)
	}
}

func TestValidator_ValidateBindTarget_Response(t *testing.T) {
	wrapperSchema := &resolver.Schema{
		Name:       "DataResponse",
		GoTypeName: "DataResponse",
		Fields: []*resolver.Field{
			{Name: "data", GoName: "Data", Type: &resolver.TypeRef{Shape: resolver.ShapeObject}},
		},
	}

	pkg := &resolver.Package{
		API: &resolver.API{
			Title:   "Test",
			Version: "1.0.0",
		},
		Schemas: map[string]*resolver.Schema{
			"User":         {Name: "User", Fields: []*resolver.Field{{Name: "id", GoName: "ID", Type: &resolver.TypeRef{Shape: resolver.ShapeScalar, Type: "string"}}}},
			"DataResponse": wrapperSchema,
		},
		Parameters: map[string]*resolver.ParameterStruct{},
		Endpoints: []*resolver.Endpoint{
			{
				Method: "GET",
				Path:   "/users",
				Responses: []*resolver.Response{
					{StatusCode: "200",
						ContentType: "application/json",
						Body: &resolver.Body{
							Schema: "User",
							Type:   &resolver.TypeRef{Shape: resolver.ShapeRef, Ref: "User"},
							Bind: &resolver.BindTarget{
								Wrapper:       "DataResponse",
								Field:         "NonExistent",
								WrapperSchema: wrapperSchema,
							},
						}},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(pkg)
	if err == nil {
		t.Fatal("expected validation error for unknown bind field")
	}
	if !strings.Contains(err.Error(), `wrapper schema "DataResponse" has no field "NonExistent"`) {
		t.Errorf("expected error about unknown field, got: %v", err)
	}
}

func TestValidator_ValidateBindTarget_InlineResponse(t *testing.T) {
	pkg := &resolver.Package{
		API: &resolver.API{
			Title:   "Test",
			Version: "1.0.0",
		},
		Schemas:    map[string]*resolver.Schema{},
		Parameters: map[string]*resolver.ParameterStruct{},
		Endpoints: []*resolver.Endpoint{
			{
				Method: "GET",
				Path:   "/users",
				Responses: []*resolver.Response{
					{StatusCode: "200"},
					{
						StatusCode:  "201",
						ContentType: "application/json",
						Inline: &resolver.InlineBody{
							ContentType: "application/json",
							Bind: &resolver.BindTarget{
								Wrapper:       "UnknownWrapper",
								Field:         "Data",
								WrapperSchema: nil,
							},
						},
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(pkg)
	if err == nil {
		t.Fatal("expected validation error for unknown inline bind wrapper")
	}
	if !strings.Contains(err.Error(), "references unknown wrapper schema: UnknownWrapper") {
		t.Errorf("expected error about unknown wrapper schema, got: %v", err)
	}
}
