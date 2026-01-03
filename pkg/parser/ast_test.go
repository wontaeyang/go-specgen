package parser

import (
	"testing"
)

func TestExtractComments(t *testing.T) {
	comments, err := ExtractComments("./testdata")
	if err != nil {
		t.Fatalf("ExtractComments() error = %v", err)
	}

	if comments == nil {
		t.Fatal("ExtractComments() returned nil")
	}

	// Test package-level comments
	if comments.PackageComments == nil {
		t.Error("PackageComments is nil, expected @api annotation")
	} else {
		if !comments.PackageComments.HasAnnotation("@api") {
			t.Error("PackageComments should have @api annotation")
		}
	}

	// Test struct-level comments
	expectedStructs := []string{"User", "UserIDPath", "SearchQuery"}
	for _, structName := range expectedStructs {
		if comments.GetStructComment(structName) == nil {
			t.Errorf("Missing struct comment for %s", structName)
		}
	}

	// InternalStruct should not have annotations
	if cb := comments.GetStructComment("InternalStruct"); cb != nil && cb.HasAnnotation("@") {
		t.Error("InternalStruct should not have annotations")
	}

	// Test field-level comments
	userFields := comments.FieldComments["User"]
	if userFields == nil {
		t.Fatal("User struct should have field comments")
	}

	if userFields["ID"] == nil {
		t.Error("User.ID should have @field annotation")
	}

	if userFields["Email"] == nil {
		t.Error("User.Email should have @field annotation")
	}

	// Name field has no @field annotation
	if userFields["Name"] != nil {
		t.Error("User.Name should not have field comment")
	}

	// Test function-level comments
	expectedFuncs := []string{"GetUser", "CreateUser"}
	for _, funcName := range expectedFuncs {
		if comments.GetFunctionComment(funcName) == nil {
			t.Errorf("Missing function comment for %s", funcName)
		}
	}

	// HelperFunction should not have annotations
	if cb := comments.GetFunctionComment("HelperFunction"); cb != nil && cb.HasAnnotation("@") {
		t.Error("HelperFunction should not have annotations")
	}
}

func TestCommentBlock_HasAnnotation(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		annotation string
		expected   bool
	}{
		{
			name:       "has @api",
			lines:      []string{"@api {", "  @title Test", "}"},
			annotation: "@api",
			expected:   true,
		},
		{
			name:       "has @field",
			lines:      []string{"@field {", "  @description Test", "}"},
			annotation: "@field",
			expected:   true,
		},
		{
			name:       "does not have @api",
			lines:      []string{"@schema", "Some description"},
			annotation: "@api",
			expected:   false,
		},
		{
			name:       "empty comment",
			lines:      []string{},
			annotation: "@api",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := &CommentBlock{Lines: tt.lines}
			if got := cb.HasAnnotation(tt.annotation); got != tt.expected {
				t.Errorf("HasAnnotation() = %v, want %v", got, tt.expected)
			}
		})
	}

	// Test nil comment block
	var nilCB *CommentBlock
	if nilCB.HasAnnotation("@api") {
		t.Error("nil CommentBlock should return false")
	}
}

func TestCommentBlock_GetAnnotationLines(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected int // Expected number of annotation lines
	}{
		{
			name: "simple annotation",
			lines: []string{
				"Some regular comment",
				"@api {",
				"  @title Test",
				"}",
			},
			expected: 3, // @api, @title, }
		},
		{
			name: "multiple annotations",
			lines: []string{
				"@schema",
				"@deprecated",
			},
			expected: 2,
		},
		{
			name: "no annotations",
			lines: []string{
				"Just a regular comment",
				"Another line",
			},
			expected: 0,
		},
		{
			name:     "empty",
			lines:    []string{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := &CommentBlock{Lines: tt.lines}
			got := cb.GetAnnotationLines()
			if len(got) != tt.expected {
				t.Errorf("GetAnnotationLines() returned %d lines, want %d", len(got), tt.expected)
			}
		})
	}

	// Test nil comment block
	var nilCB *CommentBlock
	if lines := nilCB.GetAnnotationLines(); lines != nil {
		t.Error("nil CommentBlock should return nil")
	}
}

func TestCommentBlock_String(t *testing.T) {
	cb := &CommentBlock{
		Lines: []string{"@api {", "  @title Test", "}"},
	}

	str := cb.String()
	expected := "@api {\n  @title Test\n}"
	if str != expected {
		t.Errorf("String() = %q, want %q", str, expected)
	}

	// Test nil comment block
	var nilCB *CommentBlock
	if nilCB.String() != "" {
		t.Error("nil CommentBlock.String() should return empty string")
	}
}

func TestPackageComments_Getters(t *testing.T) {
	pc := &PackageComments{
		StructComments: map[string]*CommentBlock{
			"User": {Lines: []string{"@schema"}},
		},
		FieldComments: map[string]map[string]*CommentBlock{
			"User": {
				"ID": {Lines: []string{"@field"}},
			},
		},
		FunctionComments: map[string]*CommentBlock{
			"GetUser": {Lines: []string{"@endpoint GET /users"}},
		},
	}

	// Test GetStructComment
	if cb := pc.GetStructComment("User"); cb == nil {
		t.Error("GetStructComment(User) should not be nil")
	}

	if cb := pc.GetStructComment("NonExistent"); cb != nil {
		t.Error("GetStructComment(NonExistent) should be nil")
	}

	// Test GetFieldComment
	if cb := pc.GetFieldComment("User", "ID"); cb == nil {
		t.Error("GetFieldComment(User, ID) should not be nil")
	}

	if cb := pc.GetFieldComment("User", "NonExistent"); cb != nil {
		t.Error("GetFieldComment(User, NonExistent) should be nil")
	}

	if cb := pc.GetFieldComment("NonExistent", "ID"); cb != nil {
		t.Error("GetFieldComment(NonExistent, ID) should be nil")
	}

	// Test GetFunctionComment
	if cb := pc.GetFunctionComment("GetUser"); cb == nil {
		t.Error("GetFunctionComment(GetUser) should not be nil")
	}

	if cb := pc.GetFunctionComment("NonExistent"); cb != nil {
		t.Error("GetFunctionComment(NonExistent) should be nil")
	}
}

func TestExtractFuncInlines(t *testing.T) {
	// Test extraction of inline struct declarations from the inline example
	comments, err := ExtractComments("../../examples/inline")
	if err != nil {
		t.Fatalf("ExtractComments() error = %v", err)
	}

	// Check GetUser function has inline declarations
	getUserInlines := comments.FuncInlines["GetUser"]
	if getUserInlines == nil {
		t.Fatal("GetUser should have inline declarations")
	}

	// GetUser has @path and @response 200
	if getUserInlines.Path == nil {
		t.Error("GetUser should have @path inline")
	} else {
		if getUserInlines.Path.Annotation != "path" {
			t.Errorf("Path.Annotation = %q, want %q", getUserInlines.Path.Annotation, "path")
		}
		if getUserInlines.Path.VarName != "path" {
			t.Errorf("Path.VarName = %q, want %q", getUserInlines.Path.VarName, "path")
		}
		// Check field comments
		if getUserInlines.Path.FieldComments["ID"] == nil {
			t.Error("Path should have ID field comment")
		}
	}

	if getUserInlines.Responses["200"] == nil {
		t.Error("GetUser should have @response 200 inline")
	} else {
		resp := getUserInlines.Responses["200"]
		if resp.StatusCode != "200" {
			t.Errorf("Response.StatusCode = %q, want %q", resp.StatusCode, "200")
		}
		// Check field comments
		if resp.FieldComments["ID"] == nil {
			t.Error("Response should have ID field comment")
		}
		if resp.FieldComments["Email"] == nil {
			t.Error("Response should have Email field comment")
		}
	}

	// Check CreateUser function has inline declarations
	createUserInlines := comments.FuncInlines["CreateUser"]
	if createUserInlines == nil {
		t.Fatal("CreateUser should have inline declarations")
	}

	// CreateUser has @request, @response 201, @response 400
	if createUserInlines.Request == nil {
		t.Error("CreateUser should have @request inline")
	} else {
		if createUserInlines.Request.Annotation != "request" {
			t.Errorf("Request.Annotation = %q, want %q", createUserInlines.Request.Annotation, "request")
		}
	}

	if createUserInlines.Responses["201"] == nil {
		t.Error("CreateUser should have @response 201 inline")
	}
	if createUserInlines.Responses["400"] == nil {
		t.Error("CreateUser should have @response 400 inline")
	}

	// Check ListUsers function has inline declarations
	listUsersInlines := comments.FuncInlines["ListUsers"]
	if listUsersInlines == nil {
		t.Fatal("ListUsers should have inline declarations")
	}

	// ListUsers has @query and @response 200
	if listUsersInlines.Query == nil {
		t.Error("ListUsers should have @query inline")
	} else {
		if listUsersInlines.Query.Annotation != "query" {
			t.Errorf("Query.Annotation = %q, want %q", listUsersInlines.Query.Annotation, "query")
		}
		// Check field comments
		if listUsersInlines.Query.FieldComments["Limit"] == nil {
			t.Error("Query should have Limit field comment")
		}
		if listUsersInlines.Query.FieldComments["Status"] == nil {
			t.Error("Query should have Status field comment")
		}
	}
}
