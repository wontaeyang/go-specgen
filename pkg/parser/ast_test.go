package parser

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strings"
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

	// InternalStruct carries a plain doc comment, so none of the annotations
	// that would classify it as a declaration specgen owns should match.
	if cb := comments.GetStructComment("InternalStruct"); cb != nil {
		for _, name := range []string{"@schema", "@path", "@query", "@header", "@cookie"} {
			if cb.HasAnnotation(name) {
				t.Errorf("InternalStruct should not have %s", name)
			}
		}
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

	// HelperFunction carries a plain doc comment and is not an endpoint.
	if cb := comments.GetFunctionComment("HelperFunction"); cb != nil && cb.HasAnnotation("@endpoint") {
		t.Error("HelperFunction should not have @endpoint")
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
		{
			// A prefix match read this as @header and registered the struct as
			// a header-parameter struct the user never wrote.
			name:       "longer name is not a match",
			lines:      []string{"@headers"},
			annotation: "@header",
			expected:   false,
		},
		{
			name:       "apidoc's @apiVersion is not @api",
			lines:      []string{"@apiVersion 1.0.0"},
			annotation: "@api",
			expected:   false,
		},
		{
			name:       "a block opener still matches",
			lines:      []string{"@api {", "  @title Test", "}"},
			annotation: "@api",
			expected:   true,
		},
		{
			name:       "metadata after the name still matches",
			lines:      []string{"@endpoint GET /users"},
			annotation: "@endpoint",
			expected:   true,
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
		FieldComments: map[string]map[string]*FieldComments{
			"User": {
				"ID": {Comment: &CommentBlock{Lines: []string{"@field"}}},
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
	if len(getUserInlines.Path) == 0 {
		t.Error("GetUser should have @path inline")
	} else {
		path := getUserInlines.Path[0]
		if path.Annotation != "path" {
			t.Errorf("Path.Annotation = %q, want %q", path.Annotation, "path")
		}
		if path.VarName != "path" {
			t.Errorf("Path.VarName = %q, want %q", path.VarName, "path")
		}
		// Check field comments
		if path.FieldComments["ID"] == nil {
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
	if len(listUsersInlines.Query) == 0 {
		t.Error("ListUsers should have @query inline")
	} else {
		query := listUsersInlines.Query[0]
		if query.Annotation != "query" {
			t.Errorf("Query.Annotation = %q, want %q", query.Annotation, "query")
		}
		// Check field comments
		if query.FieldComments["Limit"] == nil {
			t.Error("Query should have Limit field comment")
		}
		if query.FieldComments["Status"] == nil {
			t.Error("Query should have Status field comment")
		}
	}
}

func TestExtractFuncInlines_Closure(t *testing.T) {
	comments, err := ExtractComments("../../examples/closure")
	if err != nil {
		t.Fatalf("ExtractComments() error = %v", err)
	}

	inlines := comments.FuncInlines["HandleGreet"]
	if inlines == nil {
		t.Fatal("HandleGreet should have inline declarations")
	}

	// @request is in the outer function body
	if inlines.Request == nil {
		t.Fatal("HandleGreet should have @request inline")
	}
	if inlines.Request.Annotation != "request" {
		t.Errorf("Request.Annotation = %q, want %q", inlines.Request.Annotation, "request")
	}
	if inlines.Request.VarName != "request" {
		t.Errorf("Request.VarName = %q, want %q", inlines.Request.VarName, "request")
	}
	if inlines.Request.FieldComments["Name"] == nil {
		t.Error("Request should have Name field comment")
	}

	// @response 200 is inside the returned closure
	resp := inlines.Responses["200"]
	if resp == nil {
		t.Fatal("HandleGreet should have @response 200 inline from returned closure")
	}
	if resp.StatusCode != "200" {
		t.Errorf("Response.StatusCode = %q, want %q", resp.StatusCode, "200")
	}
	if resp.VarName != "response" {
		t.Errorf("Response.VarName = %q, want %q", resp.VarName, "response")
	}
	if resp.FieldComments["Greeting"] == nil {
		t.Error("Response should have Greeting field comment")
	}
}

func TestDetectInlineAnnotation(t *testing.T) {
	tests := []struct {
		name           string
		lines          []string
		wantAnnotation string
		wantStatusCode string
	}{
		{
			name:           "query annotation",
			lines:          []string{"@query"},
			wantAnnotation: "query",
			wantStatusCode: "",
		},
		{
			name:           "path annotation",
			lines:          []string{"@path"},
			wantAnnotation: "path",
			wantStatusCode: "",
		},
		{
			name:           "header annotation",
			lines:          []string{"@header"},
			wantAnnotation: "header",
			wantStatusCode: "",
		},
		{
			name:           "cookie annotation",
			lines:          []string{"@cookie"},
			wantAnnotation: "cookie",
			wantStatusCode: "",
		},
		{
			name:           "request annotation",
			lines:          []string{"@request"},
			wantAnnotation: "request",
			wantStatusCode: "",
		},
		{
			name:           "response without status code",
			lines:          []string{"@response"},
			wantAnnotation: "response",
			wantStatusCode: "200",
		},
		{
			name:           "response with status code 200",
			lines:          []string{"@response 200"},
			wantAnnotation: "response",
			wantStatusCode: "200",
		},
		{
			name:           "response with status code 201",
			lines:          []string{"@response 201"},
			wantAnnotation: "response",
			wantStatusCode: "201",
		},
		{
			name:           "response with status code 404",
			lines:          []string{"@response 404"},
			wantAnnotation: "response",
			wantStatusCode: "404",
		},
		{
			name:           "response with status code and block",
			lines:          []string{"@response 201 {"},
			wantAnnotation: "response",
			wantStatusCode: "201",
		},
		{
			name:           "no annotation",
			lines:          []string{"some comment", "more text"},
			wantAnnotation: "",
			wantStatusCode: "",
		},
		{
			name:           "annotation with preceding text",
			lines:          []string{"some description", "@query"},
			wantAnnotation: "query",
			wantStatusCode: "",
		},
		{
			name:           "whitespace before annotation",
			lines:          []string{"  @path  "},
			wantAnnotation: "path",
			wantStatusCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAnnotation, gotStatusCode := detectInlineAnnotation(tt.lines)
			if gotAnnotation != tt.wantAnnotation {
				t.Errorf("detectInlineAnnotation() annotation = %q, want %q", gotAnnotation, tt.wantAnnotation)
			}
			if gotStatusCode != tt.wantStatusCode {
				t.Errorf("detectInlineAnnotation() statusCode = %q, want %q", gotStatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestFuncInlineInfo_Fields(t *testing.T) {
	// Test that the FuncInlineInfo struct is correctly initialized.
	// @query/@path/@header/@cookie are repeatable (slices); @request is single-slot;
	// @response is keyed by status code.
	info := &FuncInlineInfo{
		Query:     []*InlineStructInfo{{VarName: "query"}},
		Path:      []*InlineStructInfo{{VarName: "path"}},
		Header:    []*InlineStructInfo{{VarName: "header"}},
		Cookie:    []*InlineStructInfo{{VarName: "cookie"}},
		Request:   &InlineStructInfo{VarName: "request"},
		Responses: make(map[string]*InlineStructInfo),
	}
	info.Responses["200"] = &InlineStructInfo{VarName: "resp200", StatusCode: "200"}
	info.Responses["404"] = &InlineStructInfo{VarName: "resp404", StatusCode: "404"}

	if len(info.Query) != 1 || info.Query[0].VarName != "query" {
		t.Errorf("Query[0].VarName = %q, want %q", info.Query[0].VarName, "query")
	}
	if len(info.Path) != 1 || info.Path[0].VarName != "path" {
		t.Errorf("Path[0].VarName = %q, want %q", info.Path[0].VarName, "path")
	}
	if len(info.Header) != 1 || info.Header[0].VarName != "header" {
		t.Errorf("Header[0].VarName = %q, want %q", info.Header[0].VarName, "header")
	}
	if len(info.Cookie) != 1 || info.Cookie[0].VarName != "cookie" {
		t.Errorf("Cookie[0].VarName = %q, want %q", info.Cookie[0].VarName, "cookie")
	}
	if info.Request.VarName != "request" {
		t.Errorf("Request.VarName = %q, want %q", info.Request.VarName, "request")
	}
	if len(info.Responses) != 2 {
		t.Errorf("len(Responses) = %d, want %d", len(info.Responses), 2)
	}
	if info.Responses["200"].StatusCode != "200" {
		t.Errorf("Responses[200].StatusCode = %q, want %q", info.Responses["200"].StatusCode, "200")
	}
}

func TestInlineStructInfo_Fields(t *testing.T) {
	// Test that the InlineStructInfo struct is correctly initialized
	info := &InlineStructInfo{
		VarName:       "testVar",
		Annotation:    "query",
		StatusCode:    "",
		FieldComments: make(map[string]*FieldComments),
	}
	info.FieldComments["ID"] = &FieldComments{Comment: &CommentBlock{Lines: []string{"@field { @description User ID }"}}}

	if info.VarName != "testVar" {
		t.Errorf("VarName = %q, want %q", info.VarName, "testVar")
	}
	if info.Annotation != "query" {
		t.Errorf("Annotation = %q, want %q", info.Annotation, "query")
	}
	if len(info.FieldComments) != 1 {
		t.Errorf("len(FieldComments) = %d, want %d", len(info.FieldComments), 1)
	}
}

// TestExtractFuncInlines_MustDefineStruct covers the declarations an
// in-function annotation may not sit on. Each of these used to be skipped
// without a word, so a handler's parameters or body simply went missing.
func TestExtractFuncInlines_MustDefineStruct(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "named type declared elsewhere",
			body: "\t// @query\n\tvar f Filters\n\t_ = f",
			want: `@query on "f" declares Filters rather than defining a struct`,
		},
		{
			name: "not a struct at all",
			body: "\t// @query\n\tvar limit int\n\t_ = limit",
			want: `@query on "limit" declares int rather than defining a struct`,
		},
		{
			name: "type form declaring a non-struct",
			body: "\t// @response 200\n\ttype payload int\n\t_ = payload(0)",
			want: `@response on "payload" declares int rather than defining a struct`,
		},
		{
			name: "no type at all",
			body: "\t// @query\n\tvar f = Filters{}\n\t_ = f",
			want: `@query on "f" declares no type`,
		},
		{
			name: "grouped declaration",
			body: "\t// @query\n\tvar (\n\t\ta struct{ A string }\n\t\tb struct{ B string }\n\t)\n\t_, _ = a, b",
			want: "@query annotates a group of 2 declarations",
		},
		{
			name: "several names on one declaration",
			body: "\t// @query\n\tvar a, b struct{ A string }\n\t_, _ = a, b",
			want: "@query annotates a declaration of 2 variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := parseHandlerBody(t, tt.body)

			_, err := extractFuncInlines(token.NewFileSet(), nil, "H", body)
			if err == nil {
				t.Fatalf("expected an error, got none")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.want)
			}
		})
	}
}

// TestExtractFuncInlines_AcceptsStructDefinitions is the other half: both
// spellings that define a struct on the spot stay accepted.
func TestExtractFuncInlines_AcceptsStructDefinitions(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"var form", "\t// @query\n\tvar f struct{ A string }\n\t_ = f"},
		{"type form", "\t// @request\n\ttype request struct{ A string }\n\t_ = request{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := parseHandlerBody(t, tt.body)

			inlines, err := extractFuncInlines(token.NewFileSet(), nil, "H", body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if inlines == nil {
				t.Fatal("expected inline declarations, got none")
			}
		})
	}
}

// parseHandlerBody wraps statements in a function and returns its body. The
// source is parsed with comments so doc comments reach the GenDecls.
func parseHandlerBody(t *testing.T, stmts string) *ast.BlockStmt {
	t.Helper()

	src := "package p\n\nfunc H() {\n" + stmts + "\n}\n"

	file, err := goparser.ParseFile(token.NewFileSet(), "h.go", src, goparser.ParseComments)
	if err != nil {
		t.Fatalf("parsing test source: %v", err)
	}

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("first declaration is not a function")
	}
	return fn.Body
}
