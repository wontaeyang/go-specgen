package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"path/filepath"
	"strings"

	"github.com/wontaeyang/go-specgen/pkg/annotation"
	"golang.org/x/tools/go/packages"
)

// CommentBlock represents a block of comments extracted from the AST
type CommentBlock struct {
	// Lines are the comment lines with // prefix removed and trimmed
	Lines []string

	// Position is the file position for error reporting
	Position token.Position
}

// FieldComments is the doc comment above one struct field, together with the
// comments on the fields of its type when that type is an anonymous struct.
//
// It is a tree because Go types are: an @field written inside an inline struct
// describes a field of that struct, not of the one containing it, and flattening
// the two would make them indistinguishable.
type FieldComments struct {
	// Comment is the doc comment above the field itself, if it has one.
	Comment *CommentBlock

	// Fields are the comments on the fields of this field's type, when that
	// type is an anonymous struct (or a slice or map of one).
	Fields map[string]*FieldComments
}

// harvestFieldComments collects the doc comments on a struct's fields, and
// recursively on the fields of any anonymous struct nested inside them.
//
// The previous version walked one level and stopped, so an @field written
// inside an inline struct was parsed and then dropped -- 16 of them in
// examples/inline alone.
func harvestFieldComments(fset *token.FileSet, structType *ast.StructType) map[string]*FieldComments {
	if structType == nil {
		return nil
	}

	comments := make(map[string]*FieldComments)

	for _, field := range structType.Fields.List {
		nested := harvestFieldComments(fset, anonymousStructOf(field.Type))
		if field.Doc == nil && nested == nil {
			continue
		}
		if len(field.Names) == 0 {
			continue
		}

		harvested := &FieldComments{Fields: nested}
		if field.Doc != nil {
			harvested.Comment = extractCommentBlock(fset, field.Doc)
		}

		// One *ast.Field with several Names is several fields -- "X, Y float64"
		// is two *types.Var, and the comment above them describes both.
		for _, name := range field.Names {
			comments[name.Name] = harvested
		}
	}

	if len(comments) == 0 {
		return nil
	}
	return comments
}

// anonymousStructOf returns the anonymous struct a type expression describes,
// looking through pointers, slices, arrays and maps so that []struct{...} and
// map[string]struct{...} are reached as readily as a bare struct{...}.
//
// A named type returns nil: its fields carry their own comments where the type
// is declared, and harvesting them again here would duplicate them.
func anonymousStructOf(expr ast.Expr) *ast.StructType {
	switch t := expr.(type) {
	case *ast.StructType:
		return t
	case *ast.StarExpr:
		return anonymousStructOf(t.X)
	case *ast.ArrayType:
		return anonymousStructOf(t.Elt)
	case *ast.MapType:
		return anonymousStructOf(t.Value)
	}
	return nil
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
	FieldComments map[string]map[string]*FieldComments // Key: struct name -> field name

	// Function-level comments (for @endpoint)
	FunctionComments map[string]*CommentBlock // Key: function name

	// TypeInfo contains metadata about type declarations
	TypeInfo map[string]*TypeDeclInfo // Key: type name

	// FuncInlines contains inline struct declarations within function bodies
	FuncInlines map[string]*FuncInlineInfo // Key: function name
}

// FuncInlineInfo contains inline declarations extracted from a function body.
// Cardinality matches the annotation schema:
//   - @query/@path/@header/@cookie are Repeatable: true → slices, in declaration order.
//   - @request is single-slot (not repeatable); a duplicate is an error.
//   - @response is keyed by status code; a duplicate status code is an error.
type FuncInlineInfo struct {
	// Query are the inline query parameter structs, in declaration order
	Query []*InlineStructInfo

	// Path are the inline path parameter structs, in declaration order
	Path []*InlineStructInfo

	// Header are the inline header parameter structs, in declaration order
	Header []*InlineStructInfo

	// Cookie are the inline cookie parameter structs, in declaration order
	Cookie []*InlineStructInfo

	// Request is the inline request body struct (at most one per handler)
	Request *InlineStructInfo

	// Responses are the inline response body structs keyed by status code.
	// A duplicate status code within one handler is an error.
	Responses map[string]*InlineStructInfo
}

// InlineStructInfo contains the metadata captured for an inline struct declaration.
// The resolver looks up the *types.Struct via TypesInfo.Defs[Ident] — no AST node
// is stored here. FieldComments is parser-internal transport between extraction
// (extractFuncInlines) and field parsing (parseStructFields), mirroring how
// PackageComments.FieldComments carries raw comments for named @schema structs.
type InlineStructInfo struct {
	// VarName is the variable or type name
	VarName string

	// Annotation is the annotation type (query, path, header, cookie, request, response)
	Annotation string

	// Comment is the parsed comment block containing annotations
	Comment *CommentBlock

	// Ident is the AST identifier for type resolution via TypesInfo.Defs
	Ident *ast.Ident

	// StatusCode is the response status code (for response only)
	StatusCode string

	// FieldComments are the raw per-field comments collected at extraction time.
	// Consumed by parseStructFields to populate Fields. Parser-internal — the
	// resolver does not read this.
	FieldComments map[string]*FieldComments

	// Fields are the parsed @field annotations. Populated by the parser so the
	// resolver does not re-parse annotations. Shape mirrors Schema.Fields so both
	// @schema structs and inline var structs feed the resolver through the same
	// signature.
	Fields []*Field
}

// TypeDeclInfo contains metadata about a type declaration
type TypeDeclInfo struct {
	Name        string
	IsGeneric   bool   // Has type parameters (e.g., type Foo[T any] struct{})
	IsTypeAlias bool   // Is a type alias (e.g., type Bar = Foo[Baz])
	AliasOf     string // For type aliases, the aliased type (e.g., "Foo[Baz]")

	// Local marks a type declared inside a function body. Such a type is an
	// in-function declaration, so the annotations legal on it are the
	// in-function ones rather than the ones legal on a package-level type.
	Local bool
}

// ExtractComments extracts all comment blocks from a Go package
func ExtractComments(packagePath string) (*PackageComments, error) {
	// A bare relative path is read by go/packages as an import path, not a
	// directory: "examples/petstore" is looked for in std and not found. The
	// caller means a directory, so say so.
	if !filepath.IsAbs(packagePath) && !strings.HasPrefix(packagePath, ".") {
		packagePath = "./" + packagePath
	}

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
		FieldComments:    make(map[string]map[string]*FieldComments),
		FunctionComments: make(map[string]*CommentBlock),
		TypeInfo:         make(map[string]*TypeDeclInfo),
		FuncInlines:      make(map[string]*FuncInlineInfo),
	}

	// extractFuncInlines can surface duplicate-inline errors. ast.Inspect's
	// visitor signature is `func(ast.Node) bool`, so we capture the error via
	// a closure variable and halt walking as soon as one is seen.
	var extractErr error

	// Traverse all files in the package
	for _, file := range pkg.Syntax {
		fset := pkg.Fset

		// Extract @api annotation: prefer file.Doc, fall back to scanning all comments
		if file.Doc != nil && hasAPIAnnotation(file.Doc) {
			comments.PackageComments = extractCommentBlock(fset, file.Doc)
		} else {
			for _, cg := range file.Comments {
				if hasAPIAnnotation(cg) {
					comments.PackageComments = extractCommentBlock(fset, cg)
					break
				}
			}
		}

		// A type declared inside a function body is reached by ast.Inspect just
		// like a package-level one, and is indistinguishable from it by name
		// alone. It matters because such a type is an in-function declaration —
		// examples/closure writes `type request struct` under an @request — and
		// the annotations legal on it are not the ones legal on a package-level
		// type.
		topLevel := make(map[ast.Decl]bool, len(file.Decls))
		for _, decl := range file.Decls {
			topLevel[decl] = true
		}

		// Traverse AST nodes
		ast.Inspect(file, func(n ast.Node) bool {
			if extractErr != nil {
				return false
			}
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
								Local:       !topLevel[node],
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
									comments.FieldComments[typeName] = harvestFieldComments(fset, structType)
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
					inlines, err := extractFuncInlines(fset, pkg.TypesInfo, funcName, node.Body)
					if err != nil {
						extractErr = fmt.Errorf("in function %s: %w", funcName, err)
						return false
					}
					if inlines != nil {
						comments.FuncInlines[funcName] = inlines
					}
				}
			}

			return true
		})

		if extractErr != nil {
			return nil, extractErr
		}
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

// hasAPIAnnotation checks if a comment group contains an @api annotation
func hasAPIAnnotation(cg *ast.CommentGroup) bool {
	for _, c := range cg.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSpace(text)
		// Whole-name match, for the reason HasAnnotation gives. Here the
		// near-miss is a real annotation family rather than a typo: apidoc's
		// @apiVersion and @apiParam both begin with @api, and either could win
		// this scan and be handed to parseAPI as the API block.
		if ExtractAnnotationName(text) == "@api" {
			return true
		}
	}
	return false
}

// GetStructComment returns the comment block for a struct
func (pc *PackageComments) GetStructComment(structName string) *CommentBlock {
	return pc.StructComments[structName]
}

// GetFieldComment returns the comment block for a field
func (pc *PackageComments) GetFieldComment(structName, fieldName string) *CommentBlock {
	if fields, ok := pc.FieldComments[structName]; ok {
		if field, ok := fields[fieldName]; ok {
			return field.Comment
		}
	}
	return nil
}

// GetFunctionComment returns the comment block for a function
func (pc *PackageComments) GetFunctionComment(funcName string) *CommentBlock {
	return pc.FunctionComments[funcName]
}

// HasAnnotation reports whether the block writes the named annotation on a line
// of its own.
//
// The name has to match whole. This is what classifies a declaration — a struct
// carrying @path becomes a path-parameter struct here, before any grammar is
// consulted — so a prefix match let @headers register as @header, and the
// declaration was read as something the user never wrote. ExtractAnnotationName
// draws the boundary the rest of the parser draws, which is why it is borrowed
// rather than restated.
func (cb *CommentBlock) HasAnnotation(annotation string) bool {
	if cb == nil {
		return false
	}

	for _, line := range cb.Lines {
		if ExtractAnnotationName(line) == annotation {
			return true
		}
	}
	return false
}

// firstAnnotationName is the annotation a comment block opens with, or "" when
// the block is ordinary documentation.
//
// Only the first one matters. It is the line that decides what the declaration
// is; everything after it is inside a block, where parseChildren checks each
// name against the grammar already. A line beginning \@ is prose, since that is
// how a literal @ is written everywhere else in the syntax.
func firstAnnotationName(cb *CommentBlock) string {
	if cb == nil {
		return ""
	}

	for _, line := range cb.Lines {
		if name := ExtractAnnotationName(line); name != "" {
			return name
		}
	}
	return ""
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

// collectGenDecls collects *ast.GenDecl from the top-level statements of a
// block and from any FuncLit returned via a ReturnStmt. This lets us find
// inline struct annotations inside handler-factory closures.
func collectGenDecls(body *ast.BlockStmt) []*ast.GenDecl {
	var decls []*ast.GenDecl

	for _, stmt := range body.List {
		switch s := stmt.(type) {
		case *ast.DeclStmt:
			if gd, ok := s.Decl.(*ast.GenDecl); ok {
				decls = append(decls, gd)
			}
		case *ast.ReturnStmt:
			for _, result := range s.Results {
				if fl, ok := result.(*ast.FuncLit); ok && fl.Body != nil {
					for _, inner := range fl.Body.List {
						if ds, ok := inner.(*ast.DeclStmt); ok {
							if gd, ok := ds.Decl.(*ast.GenDecl); ok {
								decls = append(decls, gd)
							}
						}
					}
				}
			}
		}
	}

	return decls
}

// extractFuncInlines extracts inline struct declarations from a function body.
// Looks for var/type declarations with @query, @path, @header, @cookie, @request,
// @response annotations. Only @response is repeatable (keyed by status code);
// a duplicate in any other category — or a duplicate status code for @response —
// is an error. Users needing composition should reference named @schema types.
//
// funcName is only for messages: a declaration inside a handler has no name a
// reader could find it by on its own.
func extractFuncInlines(fset *token.FileSet, typesInfo *types.Info, funcName string, body *ast.BlockStmt) (*FuncInlineInfo, error) {
	if body == nil {
		return nil, nil
	}

	result := &FuncInlineInfo{
		Responses: make(map[string]*InlineStructInfo),
	}
	hasInlines := false

	for _, genDecl := range collectGenDecls(body) {
		if genDecl.Doc == nil {
			continue
		}

		// Check for annotation in comment
		commentBlock := extractCommentBlock(fset, genDecl.Doc)
		if commentBlock == nil {
			continue
		}

		// detectInlineAnnotation answers "" for a name no in-function grammar
		// defines, which is the whole of what it can say about @quer. What the
		// user actually wrote is the only thing that can be reported, so an
		// unclaimed comment is asked again, lexically.
		inlineName, statusCode := detectInlineAnnotation(commentBlock.Lines)
		if inlineName == "" {
			written := firstAnnotationName(commentBlock)
			if written == "" {
				// Ordinary documentation on a local variable.
				continue
			}

			// The error is wrapped with "in function <name>" by the caller, so
			// it names only the declaration. The log line has no such wrapper
			// and carries the function itself.
			if annotation.IsKnown(written) {
				return nil, fmt.Errorf("%s cannot annotate %q; a declaration in a function body takes: %s",
					written, declaredName(genDecl), strings.Join(annotation.InFunction(), ", "))
			}

			log.Printf("%s: %s is not an annotation; it was skipped", declarationSubject(funcName, genDecl), written)
			continue
		}

		ident, structType, err := inlineStructDecl(genDecl, inlineName)
		if err != nil {
			return nil, err
		}

		inline := &InlineStructInfo{
			VarName:       ident.Name,
			Annotation:    inlineName,
			Comment:       commentBlock,
			Ident:         ident,
			StatusCode:    statusCode,
			FieldComments: harvestFieldComments(fset, structType),
		}

		// Store based on annotation type. @query/@path/@header/@cookie are
		// repeatable per schema — append. @request is single-slot. @response is
		// keyed by status code; a duplicate status code is an error.
		switch inlineName {
		case "query":
			result.Query = append(result.Query, inline)
		case "path":
			result.Path = append(result.Path, inline)
		case "header":
			result.Header = append(result.Header, inline)
		case "cookie":
			result.Cookie = append(result.Cookie, inline)
		case "request":
			if result.Request != nil {
				return nil, fmt.Errorf("duplicate inline @request on %q (previous: %q); only one inline @request per handler is allowed", ident.Name, result.Request.VarName)
			}
			result.Request = inline
		case "response":
			if statusCode == "" {
				statusCode = "200" // default status code
			}
			if existing, ok := result.Responses[statusCode]; ok {
				return nil, fmt.Errorf("duplicate inline @response %s on %q (previous: %q); each status code can have only one inline response per handler", statusCode, ident.Name, existing.VarName)
			}
			result.Responses[statusCode] = inline
		}
		hasInlines = true
	}

	if !hasInlines {
		return nil, nil
	}
	return result, nil
}

// declarationSubject names a declaration inside a handler, for a message. The
// function is half of it: a var called "filters" is not findable on its own.
func declarationSubject(funcName string, genDecl *ast.GenDecl) string {
	subject := "declaration in " + funcName
	if name := declaredName(genDecl); name != "" {
		subject = fmt.Sprintf("declaration %s.%s", funcName, name)
	}
	return subject
}

// declaredName is the first name a declaration binds, or "" when it binds none.
func declaredName(genDecl *ast.GenDecl) string {
	for _, spec := range genDecl.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			if len(s.Names) > 0 {
				return s.Names[0].Name
			}
		case *ast.TypeSpec:
			return s.Name.Name
		}
	}
	return ""
}

// inlineStructDecl reads the one struct an in-function annotation is attached
// to.
//
// The annotation goes on the declaration that *defines* the struct, never on
// one that names a struct defined elsewhere: `var f Filters` puts its @query on
// Filters itself, and @endpoint references it by name. So the only thing
// accepted here is a struct literal, in either the var or the type spelling —
// `type request struct { ... }` inside a handler defines its struct just as
// much as `var request struct { ... }` does.
//
// One annotation, one struct. A grouped declaration under a single annotation
// has no single subject; the loop that used to walk one kept whichever spec
// came last and dropped the rest without a word.
func inlineStructDecl(genDecl *ast.GenDecl, annotation string) (*ast.Ident, *ast.StructType, error) {
	keyword := strings.ToLower(genDecl.Tok.String())

	if len(genDecl.Specs) != 1 {
		return nil, nil, fmt.Errorf("@%s annotates a group of %d declarations; an in-function @%s takes one struct, so give each its own declaration and its own @%s",
			annotation, len(genDecl.Specs), annotation, annotation)
	}

	switch spec := genDecl.Specs[0].(type) {
	case *ast.ValueSpec:
		if len(spec.Names) != 1 {
			return nil, nil, fmt.Errorf("@%s annotates a declaration of %d variables; an in-function @%s takes one struct, so give each its own declaration and its own @%s",
				annotation, len(spec.Names), annotation, annotation)
		}
		st, ok := spec.Type.(*ast.StructType)
		if !ok {
			return nil, nil, notAStructDefinition(annotation, keyword, spec.Names[0].Name, spec.Type)
		}
		return spec.Names[0], st, nil

	case *ast.TypeSpec:
		st, ok := spec.Type.(*ast.StructType)
		if !ok {
			return nil, nil, notAStructDefinition(annotation, keyword, spec.Name.Name, spec.Type)
		}
		return spec.Name, st, nil
	}

	return nil, nil, fmt.Errorf("@%s must annotate a var or type declaration", annotation)
}

// notAStructDefinition reports a declaration that carries an in-function
// annotation without defining the struct it describes.
func notAStructDefinition(annotation, keyword, name string, declared ast.Expr) error {
	advice := fmt.Sprintf("write \"%s %s struct { ... }\" here, or annotate the struct where it is defined and reference it from @endpoint",
		keyword, name)

	if declared == nil {
		return fmt.Errorf("@%s on %q declares no type; an in-function @%s must define its struct: %s", annotation, name, annotation, advice)
	}
	return fmt.Errorf("@%s on %q declares %s rather than defining a struct; %s", annotation, name, formatTypeExpr(declared), advice)
}

// detectInlineAnnotation reports which in-function annotation a comment block
// opens with, and the status code when that annotation is @response.
//
// The names come from annotation.Declaration rather than a list here, so adding
// an in-function annotation means editing the grammar and nothing else.
func detectInlineAnnotation(lines []string) (name string, statusCode string) {
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		def := annotation.Declaration.GetChild(fields[0])
		if def == nil {
			continue
		}

		if def.Name != "@response" {
			return strings.TrimPrefix(def.Name, "@"), ""
		}

		// @response 404 { ... } — the status code is the block's metadata.
		if len(fields) > 1 && isStatusCode(fields[1]) {
			return "response", fields[1]
		}
		return "response", "200"
	}

	return "", ""
}

// isStatusCode reports whether s looks like an HTTP status code. Wildcard forms
// like 4XX are deliberately not accepted here: they are valid in a doc-comment
// @response, but an in-function one describes a concrete struct being returned.
func isStatusCode(s string) bool {
	return len(s) == 3 && s[0] >= '1' && s[0] <= '5'
}
