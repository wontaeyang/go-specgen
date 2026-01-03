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

	if v.errors == nil {
		t.Error("Validator.errors is nil")
	}
}

func TestValidator_Validate_ValidPackage(t *testing.T) {
	pkg := &resolver.ResolvedPackage{
		PackageName: "test",
		API: &resolver.ResolvedAPI{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Schemas: map[string]*resolver.ResolvedSchema{
			"User": {
				Name:       "User",
				GoTypeName: "User",
				Fields: []*resolver.ResolvedField{
					{
						Name:        "id",
						GoName:      "ID",
						OpenAPIType: "string",
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
					"200": {
						StatusCode:  "200",
						Description: "Success",
					},
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

func TestValidator_Validate_MissingAPI(t *testing.T) {
	pkg := &resolver.ResolvedPackage{
		PackageName: "test",
		API:         nil,
		Schemas:     map[string]*resolver.ResolvedSchema{},
		Parameters:  map[string]*resolver.ResolvedParameter{},
		Endpoints:   []*resolver.ResolvedEndpoint{},
	}

	v := NewValidator()
	err := v.Validate(pkg)
	if err == nil {
		t.Error("Validate() should error when API is missing")
	}

	if !strings.Contains(err.Error(), "@api") {
		t.Errorf("Error should mention @api, got: %v", err)
	}
}

func TestValidator_ValidateAPI_MissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		api     *resolver.ResolvedAPI
		wantErr string
	}{
		{
			name: "missing title",
			api: &resolver.ResolvedAPI{
				Version: "1.0.0",
			},
			wantErr: "@title",
		},
		{
			name: "missing version",
			api: &resolver.ResolvedAPI{
				Title: "Test API",
			},
			wantErr: "@version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.ResolvedPackage{
				API:        tt.api,
				Schemas:    map[string]*resolver.ResolvedSchema{},
				Parameters: map[string]*resolver.ResolvedParameter{},
				Endpoints:  []*resolver.ResolvedEndpoint{},
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
			pkg := &resolver.ResolvedPackage{
				API: &resolver.ResolvedAPI{
					Title:   "Test",
					Version: "1.0.0",
					SecuritySchemes: map[string]*resolver.SecurityScheme{
						"test": tt.scheme,
					},
				},
				Schemas:    map[string]*resolver.ResolvedSchema{},
				Parameters: map[string]*resolver.ResolvedParameter{},
				Endpoints:  []*resolver.ResolvedEndpoint{},
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
		schema  *resolver.ResolvedSchema
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid schema",
			schema: &resolver.ResolvedSchema{
				Name: "User",
				Fields: []*resolver.ResolvedField{
					{Name: "id", GoName: "ID", OpenAPIType: "string"},
				},
			},
			wantErr: false,
		},
		{
			name: "schema with no fields",
			schema: &resolver.ResolvedSchema{
				Name:   "Empty",
				Fields: []*resolver.ResolvedField{},
			},
			wantErr: true,
			errMsg:  "no fields",
		},
		{
			name: "duplicate field names",
			schema: &resolver.ResolvedSchema{
				Name: "User",
				Fields: []*resolver.ResolvedField{
					{Name: "id", GoName: "ID", OpenAPIType: "string"},
					{Name: "id", GoName: "Id", OpenAPIType: "string"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.ResolvedPackage{
				API: &resolver.ResolvedAPI{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas: map[string]*resolver.ResolvedSchema{
					tt.schema.Name: tt.schema,
				},
				Parameters: map[string]*resolver.ResolvedParameter{},
				Endpoints:  []*resolver.ResolvedEndpoint{},
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
		field   *resolver.ResolvedField
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid field",
			field: &resolver.ResolvedField{
				Name:        "age",
				GoName:      "Age",
				OpenAPIType: "integer",
			},
			wantErr: false,
		},
		{
			name: "enum on integer",
			field: &resolver.ResolvedField{
				Name:        "age",
				GoName:      "Age",
				OpenAPIType: "integer",
				Enum:        []string{"1", "2", "3"},
			},
			wantErr: false,
		},
		{
			name: "enum on array of strings",
			field: &resolver.ResolvedField{
				Name:        "tags",
				GoName:      "Tags",
				OpenAPIType: "array",
				IsArray:     true,
				ItemsType:   "string",
				Enum:        []string{"red", "green", "blue"},
			},
			wantErr: false,
		},
		{
			name: "enum on array of integers",
			field: &resolver.ResolvedField{
				Name:        "levels",
				GoName:      "Levels",
				OpenAPIType: "array",
				IsArray:     true,
				ItemsType:   "integer",
				Enum:        []string{"1", "2", "3"},
			},
			wantErr: false,
		},
		{
			name: "enum on boolean",
			field: &resolver.ResolvedField{
				Name:        "active",
				GoName:      "Active",
				OpenAPIType: "boolean",
				Enum:        []string{"true", "false"},
			},
			wantErr: true,
			errMsg:  "enum only supported for string, integer, or array types",
		},
		{
			name: "enum on array of objects",
			field: &resolver.ResolvedField{
				Name:        "items",
				GoName:      "Items",
				OpenAPIType: "array",
				IsArray:     true,
				ItemsType:   "object",
				Enum:        []string{"a", "b"},
			},
			wantErr: true,
			errMsg:  "enum for arrays only supported with string or integer items",
		},
		{
			name: "min > max",
			field: &resolver.ResolvedField{
				Name:        "value",
				GoName:      "Value",
				OpenAPIType: "number",
				Minimum:     &maxVal,
				Maximum:     &minVal,
			},
			wantErr: true,
			errMsg:  "minimum cannot be greater than maximum",
		},
		{
			name: "minLength > maxLength",
			field: &resolver.ResolvedField{
				Name:        "text",
				GoName:      "Text",
				OpenAPIType: "string",
				MinLength:   &maxLen,
				MaxLength:   &minLen,
			},
			wantErr: true,
			errMsg:  "minLength cannot be greater than maxLength",
		},
		{
			name: "length constraints on non-string",
			field: &resolver.ResolvedField{
				Name:        "count",
				GoName:      "Count",
				OpenAPIType: "integer",
				MinLength:   &minLen,
			},
			wantErr: true,
			errMsg:  "only valid for string",
		},
		{
			name: "pattern on non-string",
			field: &resolver.ResolvedField{
				Name:        "count",
				GoName:      "Count",
				OpenAPIType: "integer",
				Pattern:     "^[0-9]+$",
			},
			wantErr: true,
			errMsg:  "only valid for string",
		},
		{
			name: "invalid pattern regex",
			field: &resolver.ResolvedField{
				Name:        "text",
				GoName:      "Text",
				OpenAPIType: "string",
				Pattern:     "[invalid",
			},
			wantErr: true,
			errMsg:  "invalid pattern",
		},
		{
			name: "minItems > maxItems",
			field: &resolver.ResolvedField{
				Name:        "tags",
				GoName:      "Tags",
				OpenAPIType: "array",
				IsArray:     true,
				MinItems:    &maxLen,
				MaxItems:    &minLen,
			},
			wantErr: true,
			errMsg:  "minItems cannot be greater than maxItems",
		},
		{
			name: "items constraints on non-array",
			field: &resolver.ResolvedField{
				Name:        "name",
				GoName:      "Name",
				OpenAPIType: "string",
				MinItems:    &minLen,
			},
			wantErr: true,
			errMsg:  "only valid for array",
		},
		{
			name: "valid array with minItems and maxItems",
			field: &resolver.ResolvedField{
				Name:        "tags",
				GoName:      "Tags",
				OpenAPIType: "array",
				IsArray:     true,
				MinItems:    &minLen,
				MaxItems:    &maxLen,
			},
			wantErr: false,
		},
		{
			name: "uniqueItems on non-array",
			field: &resolver.ResolvedField{
				Name:        "name",
				GoName:      "Name",
				OpenAPIType: "string",
				UniqueItems: true,
			},
			wantErr: true,
			errMsg:  "only valid for array",
		},
		{
			name: "valid array with uniqueItems",
			field: &resolver.ResolvedField{
				Name:        "tags",
				GoName:      "Tags",
				OpenAPIType: "array",
				IsArray:     true,
				UniqueItems: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.ResolvedPackage{
				API: &resolver.ResolvedAPI{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas: map[string]*resolver.ResolvedSchema{
					"Test": {
						Name:   "Test",
						Fields: []*resolver.ResolvedField{tt.field},
					},
				},
				Parameters: map[string]*resolver.ResolvedParameter{},
				Endpoints:  []*resolver.ResolvedEndpoint{},
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
		field     *resolver.ResolvedField
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "path param cannot be nullable",
			paramType: "path",
			field: &resolver.ResolvedField{
				Name:        "id",
				GoName:      "ID",
				OpenAPIType: "string",
				Nullable:    true,
			},
			wantErr: true,
			errMsg:  "cannot be nullable",
		},
		{
			name:      "path param cannot be array",
			paramType: "path",
			field: &resolver.ResolvedField{
				Name:        "id",
				GoName:      "ID",
				OpenAPIType: "array",
				IsArray:     true,
			},
			wantErr: true,
			errMsg:  "cannot be arrays",
		},
		{
			name:      "header param cannot be array",
			paramType: "header",
			field: &resolver.ResolvedField{
				Name:        "token",
				GoName:      "Token",
				OpenAPIType: "array",
				IsArray:     true,
			},
			wantErr: true,
			errMsg:  "cannot be arrays",
		},
		{
			name:      "cookie param cannot be array",
			paramType: "cookie",
			field: &resolver.ResolvedField{
				Name:        "session",
				GoName:      "Session",
				OpenAPIType: "array",
				IsArray:     true,
			},
			wantErr: true,
			errMsg:  "cannot be arrays",
		},
		{
			name:      "query param can be array",
			paramType: "query",
			field: &resolver.ResolvedField{
				Name:        "tags",
				GoName:      "Tags",
				OpenAPIType: "array",
				IsArray:     true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.ResolvedPackage{
				API: &resolver.ResolvedAPI{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas: map[string]*resolver.ResolvedSchema{},
				Parameters: map[string]*resolver.ResolvedParameter{
					"TestParam": {
						Name:   "TestParam",
						Type:   tt.paramType,
						Fields: []*resolver.ResolvedField{tt.field},
					},
				},
				Endpoints: []*resolver.ResolvedEndpoint{},
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
		endpoint *resolver.ResolvedEndpoint
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid endpoint",
			endpoint: &resolver.ResolvedEndpoint{
				Method: "GET",
				Path:   "/users",
				Responses: map[string]*resolver.ResolvedResponse{
					"200": {StatusCode: "200"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid method",
			endpoint: &resolver.ResolvedEndpoint{
				Method: "INVALID",
				Path:   "/users",
				Responses: map[string]*resolver.ResolvedResponse{
					"200": {StatusCode: "200"},
				},
			},
			wantErr: true,
			errMsg:  "invalid HTTP method",
		},
		{
			name: "missing path",
			endpoint: &resolver.ResolvedEndpoint{
				Method: "GET",
				Path:   "",
				Responses: map[string]*resolver.ResolvedResponse{
					"200": {StatusCode: "200"},
				},
			},
			wantErr: true,
			errMsg:  "missing path",
		},
		{
			name: "path not starting with /",
			endpoint: &resolver.ResolvedEndpoint{
				Method: "GET",
				Path:   "users",
				Responses: map[string]*resolver.ResolvedResponse{
					"200": {StatusCode: "200"},
				},
			},
			wantErr: true,
			errMsg:  "must start with /",
		},
		{
			name: "no responses",
			endpoint: &resolver.ResolvedEndpoint{
				Method:    "GET",
				Path:      "/users",
				Responses: map[string]*resolver.ResolvedResponse{},
			},
			wantErr: true,
			errMsg:  "at least one response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.ResolvedPackage{
				API: &resolver.ResolvedAPI{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas:    map[string]*resolver.ResolvedSchema{},
				Parameters: map[string]*resolver.ResolvedParameter{},
				Endpoints:  []*resolver.ResolvedEndpoint{tt.endpoint},
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
		params  []*resolver.ResolvedParameter
		wantErr bool
		errMsg  string
	}{
		{
			name: "matching path variable and parameter",
			path: "/users/{id}",
			params: []*resolver.ResolvedParameter{
				{
					Fields: []*resolver.ResolvedField{
						{Name: "id", GoName: "ID"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "path variable without parameter",
			path:    "/users/{id}",
			params:  []*resolver.ResolvedParameter{},
			wantErr: true,
			errMsg:  "has no corresponding @path parameter",
		},
		{
			name: "parameter not used in path",
			path: "/users",
			params: []*resolver.ResolvedParameter{
				{
					Fields: []*resolver.ResolvedField{
						{Name: "id", GoName: "ID"},
					},
				},
			},
			wantErr: true,
			errMsg:  "not used in path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.ResolvedPackage{
				API: &resolver.ResolvedAPI{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas:    map[string]*resolver.ResolvedSchema{},
				Parameters: map[string]*resolver.ResolvedParameter{},
				Endpoints: []*resolver.ResolvedEndpoint{
					{
						Method:     "GET",
						Path:       tt.path,
						PathParams: tt.params,
						Responses: map[string]*resolver.ResolvedResponse{
							"200": {StatusCode: "200"},
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
		request *resolver.ResolvedRequestBody
		schemas map[string]*resolver.ResolvedSchema
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: &resolver.ResolvedRequestBody{
				ContentType: "application/json",
				Body:        &resolver.ResolvedBody{Schema: "User"},
			},
			schemas: map[string]*resolver.ResolvedSchema{
				"User": {
					Name: "User",
					Fields: []*resolver.ResolvedField{
						{Name: "id", GoName: "ID", OpenAPIType: "string"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing content type",
			request: &resolver.ResolvedRequestBody{
				Body: &resolver.ResolvedBody{Schema: "User"},
			},
			schemas: map[string]*resolver.ResolvedSchema{
				"User": {Name: "User"},
			},
			wantErr: true,
			errMsg:  "@contentType",
		},
		{
			name: "unknown schema",
			request: &resolver.ResolvedRequestBody{
				ContentType: "application/json",
				Body:        &resolver.ResolvedBody{Schema: "Unknown"},
			},
			schemas: map[string]*resolver.ResolvedSchema{},
			wantErr: true,
			errMsg:  "unknown schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &resolver.ResolvedPackage{
				API: &resolver.ResolvedAPI{
					Title:   "Test",
					Version: "1.0.0",
				},
				Schemas:    tt.schemas,
				Parameters: map[string]*resolver.ResolvedParameter{},
				Endpoints: []*resolver.ResolvedEndpoint{
					{
						Method:  "POST",
						Path:    "/users",
						Request: tt.request,
						Responses: map[string]*resolver.ResolvedResponse{
							"200": {StatusCode: "200"},
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

func TestMultiError_Error(t *testing.T) {
	tests := []struct {
		name   string
		errors []error
		want   string
	}{
		{
			name: "single error",
			errors: []error{
				&ValidationError{Message: "test error"},
			},
			want: "test error",
		},
		{
			name: "multiple errors",
			errors: []error{
				&ValidationError{Message: "error 1"},
				&ValidationError{Message: "error 2"},
			},
			want: "2 validation errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			me := &MultiError{Errors: tt.errors}
			got := me.Error()
			if !strings.Contains(got, tt.want) {
				t.Errorf("MultiError.Error() = %q, want to contain %q", got, tt.want)
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

			if tt.wantError && len(v.errors) == 0 {
				t.Errorf("expected error but got none")
			}

			if !tt.wantError && len(v.errors) > 0 {
				t.Errorf("expected no error but got: %v", v.errors)
			}

			if tt.wantError && len(v.errors) > 0 {
				found := false
				for _, err := range v.errors {
					if strings.Contains(err.Error(), tt.errorMessage) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q but got: %v", tt.errorMessage, v.errors)
				}
			}
		})
	}
}

func TestValidator_ValidateEndpointWithTags(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  *resolver.ResolvedEndpoint
		pkg       *resolver.ResolvedPackage
		wantError bool
		errorMsg  string
	}{
		{
			name: "endpoint with valid tags",
			endpoint: &resolver.ResolvedEndpoint{
				Method: "GET",
				Path:   "/pets",
				Tags:   []string{"pets"},
				Responses: map[string]*resolver.ResolvedResponse{
					"200": {StatusCode: "200", Description: "Success"},
				},
			},
			pkg: &resolver.ResolvedPackage{
				API: &resolver.ResolvedAPI{
					Title:   "Test API",
					Version: "1.0.0",
					Tags: []*resolver.Tag{
						{Name: "pets", Description: "Pet operations"},
					},
				},
				Schemas:    map[string]*resolver.ResolvedSchema{},
				Parameters: map[string]*resolver.ResolvedParameter{},
			},
			wantError: false,
		},
		{
			name: "endpoint with invalid tag",
			endpoint: &resolver.ResolvedEndpoint{
				Method: "GET",
				Path:   "/users",
				Tags:   []string{"users"},
				Responses: map[string]*resolver.ResolvedResponse{
					"200": {StatusCode: "200", Description: "Success"},
				},
			},
			pkg: &resolver.ResolvedPackage{
				API: &resolver.ResolvedAPI{
					Title:   "Test API",
					Version: "1.0.0",
					Tags: []*resolver.Tag{
						{Name: "pets", Description: "Pet operations"},
					},
				},
				Schemas:    map[string]*resolver.ResolvedSchema{},
				Parameters: map[string]*resolver.ResolvedParameter{},
			},
			wantError: true,
			errorMsg:  "endpoint uses undefined tag: users",
		},
		{
			name: "endpoint without tags when API has tags",
			endpoint: &resolver.ResolvedEndpoint{
				Method: "GET",
				Path:   "/health",
				Tags:   []string{},
				Responses: map[string]*resolver.ResolvedResponse{
					"200": {StatusCode: "200", Description: "Success"},
				},
			},
			pkg: &resolver.ResolvedPackage{
				API: &resolver.ResolvedAPI{
					Title:   "Test API",
					Version: "1.0.0",
					Tags: []*resolver.Tag{
						{Name: "pets", Description: "Pet operations"},
					},
				},
				Schemas:    map[string]*resolver.ResolvedSchema{},
				Parameters: map[string]*resolver.ResolvedParameter{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.validateEndpoint(tt.endpoint, tt.pkg)

			if tt.wantError && len(v.errors) == 0 {
				t.Errorf("expected error but got none")
			}

			if !tt.wantError && len(v.errors) > 0 {
				t.Errorf("expected no error but got: %v", v.errors)
			}

			if tt.wantError && tt.errorMsg != "" && len(v.errors) > 0 {
				found := false
				for _, err := range v.errors {
					if strings.Contains(err.Error(), tt.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q but got: %v", tt.errorMsg, v.errors)
				}
			}
		})
	}
}
