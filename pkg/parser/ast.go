package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// CommentBlock represents a block of comments extracted from the AST
type CommentBlock struct {
	// Lines are the comment lines with // prefix removed and trimmed
	Lines []string

	// Position is the file position for error reporting
	Position token.Position
}

// PackageComments represents all comments extracted from a package
type PackageComments struct {
	// Name is the package name
	Name string

	// Pkg is the loaded package (needed for type resolution of inline declarations)
	Pkg *packages.Package

	// Package-level comments (for @api)
	PackageComments *CommentBlock

	// Struct-level comments (for @schema, @path, @query, @header, @cookie)
	StructComments map[string]*CommentBlock // Key: struct name

	// Field-level comments (for @field)
	FieldComments map[string]map[string]*CommentBlock // Key: struct name -> field name

	// Function-level comments (for @endpoint)
	FunctionComments map[string]*CommentBlock // Key: function name

	// TypeInfo contains metadata about type declarations
	TypeInfo map[string]*TypeDeclInfo // Key: type name

	// FuncInlines contains inline struct declarations within function bodies
	FuncInlines map[string]*FuncInlineInfo // Key: function name
}

// FuncInlineInfo contains inline declarations extracted from a function body
type FuncInlineInfo struct {
	// Query is the inline query parameter struct
	Query *InlineStructInfo

	// Path is the inline path parameter struct
	Path *InlineStructInfo

	// Header is the inline header parameter struct
	Header *InlineStructInfo

	// Cookie is the inline cookie parameter struct
	Cookie *InlineStructInfo

	// Request is the inline request body struct
	Request *InlineStructInfo

	// Responses are the inline response body structs keyed by status code
	Responses map[string]*InlineStructInfo
}

// InlineStructInfo contains AST information for an inline struct declaration
type InlineStructInfo struct {
	// VarName is the variable or type name
	VarName string

	// Annotation is the annotation type (query, path, header, cookie, request, response)
	Annotation string

	// Comment is the parsed comment block containing annotations
	Comment *CommentBlock

	// StructType is the AST struct type for field extraction
	StructType *ast.StructType

	// Ident is the AST identifier for type resolution via TypesInfo.Defs
	Ident *ast.Ident

	// StatusCode is the response status code (for response only)
	StatusCode string

	// FieldComments are the comments for struct fields
	FieldComments map[string]*CommentBlock
}

// TypeDeclInfo contains metadata about a type declaration
type TypeDeclInfo struct {
	Name        string
	IsGeneric   bool   // Has type parameters (e.g., type Foo[T any] struct{})
	IsTypeAlias bool   // Is a type alias (e.g., type Bar = Foo[Baz])
	AliasOf     string // For type aliases, the aliased type (e.g., "Foo[Baz]")
}

// ExtractComments extracts all comment blocks from a Go package
func ExtractComments(packagePath string) (*PackageComments, error) {
	// Load the package with documentation
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
	}

	pkgs, err := packages.Load(cfg, packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found at path: %s", packagePath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package has errors: %v", pkg.Errors)
	}

	comments := &PackageComments{
		Name:             pkg.Name,
		Pkg:              pkg,
		StructComments:   make(map[string]*CommentBlock),
		FieldComments:    make(map[string]map[string]*CommentBlock),
		FunctionComments: make(map[string]*CommentBlock),
		TypeInfo:         make(map[string]*TypeDeclInfo),
		FuncInlines:      make(map[string]*FuncInlineInfo),
	}

	// Traverse all files in the package
	for _, file := range pkg.Syntax {
		fset := pkg.Fset

		// Extract package-level comments
		if file.Doc != nil {
			comments.PackageComments = extractCommentBlock(fset, file.Doc)
		}

		// Traverse AST nodes
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.GenDecl:
				// Handle type declarations
				if node.Tok == token.TYPE {
					for _, spec := range node.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							typeName := typeSpec.Name.Name

							// Check if it's a type alias (type X = Y)
							isTypeAlias := typeSpec.Assign.IsValid()

							// Check if it's a generic type (has type parameters)
							isGeneric := typeSpec.TypeParams != nil && typeSpec.TypeParams.NumFields() > 0

							// Store type info for all type declarations
							typeInfo := &TypeDeclInfo{
								Name:        typeName,
								IsGeneric:   isGeneric,
								IsTypeAlias: isTypeAlias,
							}

							// For type aliases, extract the aliased type
							if isTypeAlias {
								typeInfo.AliasOf = formatTypeExpr(typeSpec.Type)
							}

							comments.TypeInfo[typeName] = typeInfo

							// Handle struct declarations (including generic ones)
							if _, ok := typeSpec.Type.(*ast.StructType); ok {
								// Extract struct-level comment
								if node.Doc != nil {
									comments.StructComments[typeName] = extractCommentBlock(fset, node.Doc)
								} else if typeSpec.Doc != nil {
									comments.StructComments[typeName] = extractCommentBlock(fset, typeSpec.Doc)
								}

								// Extract field-level comments
								if structType, ok := typeSpec.Type.(*ast.StructType); ok {
									comments.FieldComments[typeName] = make(map[string]*CommentBlock)

									for _, field := range structType.Fields.List {
										if field.Doc != nil && len(field.Names) > 0 {
											fieldName := field.Names[0].Name
											comments.FieldComments[typeName][fieldName] = extractCommentBlock(fset, field.Doc)
										}
									}
								}
							}

							// Handle type aliases that reference other types
							if isTypeAlias {
								// Also extract comment for type alias
								if node.Doc != nil {
									comments.StructComments[typeName] = extractCommentBlock(fset, node.Doc)
								} else if typeSpec.Doc != nil {
									comments.StructComments[typeName] = extractCommentBlock(fset, typeSpec.Doc)
								}
							}
						}
					}
				}

			case *ast.FuncDecl:
				funcName := node.Name.Name
				// Extract function-level comments
				if node.Doc != nil {
					comments.FunctionComments[funcName] = extractCommentBlock(fset, node.Doc)
				}
				// Extract inline declarations from function body
				if node.Body != nil {
					inlines := extractFuncInlines(fset, pkg.TypesInfo, node.Body)
					if inlines != nil {
						comments.FuncInlines[funcName] = inlines
					}
				}
			}

			return true
		})
	}

	return comments, nil
}

// extractCommentBlock extracts lines from a comment group
func extractCommentBlock(fset *token.FileSet, cg *ast.CommentGroup) *CommentBlock {
	if cg == nil {
		return nil
	}

	lines := make([]string, 0, len(cg.List))
	var position token.Position

	for i, comment := range cg.List {
		if i == 0 {
			position = fset.Position(comment.Pos())
		}

		// Remove // or /* */ and trim whitespace
		text := comment.Text
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)

		// Only add non-empty lines
		if text != "" {
			lines = append(lines, text)
		}
	}

	if len(lines) == 0 {
		return nil
	}

	return &CommentBlock{
		Lines:    lines,
		Position: position,
	}
}

// GetStructComment returns the comment block for a struct
func (pc *PackageComments) GetStructComment(structName string) *CommentBlock {
	return pc.StructComments[structName]
}

// GetFieldComment returns the comment block for a field
func (pc *PackageComments) GetFieldComment(structName, fieldName string) *CommentBlock {
	if fields, ok := pc.FieldComments[structName]; ok {
		return fields[fieldName]
	}
	return nil
}

// GetFunctionComment returns the comment block for a function
func (pc *PackageComments) GetFunctionComment(funcName string) *CommentBlock {
	return pc.FunctionComments[funcName]
}

// HasAnnotation checks if a comment block contains an annotation
func (cb *CommentBlock) HasAnnotation(annotation string) bool {
	if cb == nil {
		return false
	}

	for _, line := range cb.Lines {
		if strings.HasPrefix(line, annotation) {
			return true
		}
	}
	return false
}

// GetAnnotationLines returns all lines that are part of an annotation
// This includes the annotation line and any continuation lines
func (cb *CommentBlock) GetAnnotationLines() []string {
	if cb == nil {
		return nil
	}

	var result []string
	inAnnotation := false

	for _, line := range cb.Lines {
		// Check if line starts with @
		if strings.HasPrefix(line, "@") {
			inAnnotation = true
			result = append(result, line)
		} else if inAnnotation {
			// Continuation line (indented or part of multi-line annotation)
			result = append(result, line)
		}
	}

	return result
}

// String returns the comment block as a formatted string for debugging
func (cb *CommentBlock) String() string {
	if cb == nil {
		return ""
	}
	return strings.Join(cb.Lines, "\n")
}

// formatTypeExpr converts an AST type expression to a string representation
// e.g., *ast.IndexExpr for Foo[Bar] -> "Foo[Bar]"
func formatTypeExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		// pkg.Type
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
		return e.Sel.Name
	case *ast.StarExpr:
		// *Type
		return "*" + formatTypeExpr(e.X)
	case *ast.ArrayType:
		// []Type
		return "[]" + formatTypeExpr(e.Elt)
	case *ast.MapType:
		// map[Key]Value
		return "map[" + formatTypeExpr(e.Key) + "]" + formatTypeExpr(e.Value)
	case *ast.IndexExpr:
		// Generic with single type param: Foo[T]
		return formatTypeExpr(e.X) + "[" + formatTypeExpr(e.Index) + "]"
	case *ast.IndexListExpr:
		// Generic with multiple type params: Foo[T, U]
		params := make([]string, len(e.Indices))
		for i, idx := range e.Indices {
			params[i] = formatTypeExpr(idx)
		}
		return formatTypeExpr(e.X) + "[" + strings.Join(params, ", ") + "]"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// extractFuncInlines extracts inline struct declarations from a function body
// It looks for var/type declarations with @query, @path, @header, @cookie, @request, @response annotations
func extractFuncInlines(fset *token.FileSet, typesInfo *types.Info, body *ast.BlockStmt) *FuncInlineInfo {
	if body == nil {
		return nil
	}

	result := &FuncInlineInfo{
		Responses: make(map[string]*InlineStructInfo),
	}
	hasInlines := false

	for _, stmt := range body.List {
		decl, ok := stmt.(*ast.DeclStmt)
		if !ok {
			continue
		}
		genDecl, ok := decl.Decl.(*ast.GenDecl)
		if !ok || genDecl.Doc == nil {
			continue
		}

		// Check for annotation in comment
		commentBlock := extractCommentBlock(fset, genDecl.Doc)
		if commentBlock == nil {
			continue
		}

		annotation, statusCode := detectInlineAnnotation(commentBlock.Lines)
		if annotation == "" {
			continue
		}

		// Extract struct from var or type declaration
		var ident *ast.Ident
		var structType *ast.StructType

		if genDecl.Tok == token.VAR {
			// var x struct { ... }
			for _, spec := range genDecl.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok && len(vs.Names) > 0 {
					ident = vs.Names[0]
					if st, ok := vs.Type.(*ast.StructType); ok {
						structType = st
					}
				}
			}
		} else if genDecl.Tok == token.TYPE {
			// type X struct { ... }
			for _, spec := range genDecl.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					ident = ts.Name
					if st, ok := ts.Type.(*ast.StructType); ok {
						structType = st
					}
				}
			}
		}

		if ident == nil || structType == nil {
			continue
		}

		// Extract field comments from the struct
		fieldComments := make(map[string]*CommentBlock)
		for _, field := range structType.Fields.List {
			if field.Doc != nil && len(field.Names) > 0 {
				fieldName := field.Names[0].Name
				fieldComments[fieldName] = extractCommentBlock(fset, field.Doc)
			}
		}

		inline := &InlineStructInfo{
			VarName:       ident.Name,
			Annotation:    annotation,
			Comment:       commentBlock,
			StructType:    structType,
			Ident:         ident,
			StatusCode:    statusCode,
			FieldComments: fieldComments,
		}

		// Store based on annotation type
		switch annotation {
		case "query":
			result.Query = inline
		case "path":
			result.Path = inline
		case "header":
			result.Header = inline
		case "cookie":
			result.Cookie = inline
		case "request":
			result.Request = inline
		case "response":
			if statusCode == "" {
				statusCode = "200" // default status code
			}
			result.Responses[statusCode] = inline
		}
		hasInlines = true
	}

	if !hasInlines {
		return nil
	}
	return result
}

// detectInlineAnnotation detects the annotation type from comment lines
// Returns the annotation type and optionally a status code for responses
func detectInlineAnnotation(lines []string) (annotation string, statusCode string) {
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "@query") {
			return "query", ""
		}
		if strings.HasPrefix(line, "@path") {
			return "path", ""
		}
		if strings.HasPrefix(line, "@header") {
			return "header", ""
		}
		if strings.HasPrefix(line, "@cookie") {
			return "cookie", ""
		}
		if strings.HasPrefix(line, "@request") {
			return "request", ""
		}
		if strings.HasPrefix(line, "@response") {
			// Extract status code if present: @response 200 { ... } or @response 200
			rest := strings.TrimPrefix(line, "@response")
			rest = strings.TrimSpace(rest)
			// Parse status code (first numeric part)
			parts := strings.Fields(rest)
			if len(parts) > 0 {
				// Check if first part is a status code
				code := parts[0]
				if len(code) == 3 && code[0] >= '1' && code[0] <= '5' {
					return "response", code
				}
			}
			return "response", "200" // default status code
		}
	}
	return "", ""
}
