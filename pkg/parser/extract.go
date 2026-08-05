package parser

import (
	"cmp"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// This file is the source-harvesting half of the parser: load the Go package,
// collect the comment blocks that can carry annotations, and find the annotated
// declarations inside handler bodies. Nothing here interprets annotations —
// that is parse.go's job.

// CommentBlock is a comment group with the lines that can carry annotations.
// Comment markers are stripped, lines are trimmed, and empty lines are dropped.
type CommentBlock struct {
	Lines []Line

	// Pos is the position of the block's first line.
	Pos token.Position
}

// HasAnnotation reports whether any line of the block starts with the given
// annotation name.
func (cb *CommentBlock) HasAnnotation(name string) bool {
	if cb == nil {
		return false
	}
	for _, line := range cb.Lines {
		if strings.HasPrefix(line.Text, name) {
			return true
		}
	}
	return false
}

// AnnotationLines returns the block's lines from the first annotation line
// onward. Prose above the first annotation is dropped; everything below it is
// kept, because it may be a multi-line value or a block body.
func (cb *CommentBlock) AnnotationLines() []Line {
	if cb == nil {
		return nil
	}
	for i, line := range cb.Lines {
		if strings.HasPrefix(line.Text, "@") {
			return cb.Lines[i:]
		}
	}
	return nil
}

// source is everything harvested from a package's AST, in source order.
type source struct {
	pkg   *packages.Package
	name  string
	api   *CommentBlock // the comment block carrying @api, if any
	types []*typeDecl
	funcs []*funcDecl
}

// typeDecl is one type declaration and the comments attached to it.
type typeDecl struct {
	Name string
	Pos  token.Position
	Doc  *CommentBlock

	// IsStruct reports that the declared type is a struct literal. Fields is
	// populated only for those.
	IsStruct bool
	Fields   []*fieldDecl

	IsGeneric   bool
	IsTypeAlias bool
	AliasOf     string
}

// fieldDecl is one named struct field and the comment attached to it.
type fieldDecl struct {
	GoName string
	Pos    token.Position
	Doc    *CommentBlock

	// Fields are the fields of the anonymous struct this field's type
	// contains, if any. An anonymous struct has no declaration of its own to
	// hang annotations on, so its fields are recorded here, under the field
	// that spells the struct out.
	Fields []*fieldDecl
}

// funcDecl is one function declaration, its comments, and the annotated
// declarations found in its body.
type funcDecl struct {
	Name    string
	Pos     token.Position
	Doc     *CommentBlock
	Inlines []*inlineDecl
}

// inlineDecl is an annotated var or type declaration inside a function body.
type inlineDecl struct {
	// Marker is the annotation that introduced it, including the @:
	// @path, @query, @header, @cookie, @request or @response.
	Marker string

	// Status is the literal status text of an @response marker, defaulting to
	// "200" when the marker carries none.
	Status string

	VarName string
	Pos     token.Position

	// Lines are the comment lines from the marker onward, ready to parse
	// against InlineGrammar.
	Lines []Line

	// Struct is the declared Go type. It is nil for a standalone declaration,
	// which has none.
	Struct *types.Struct

	Fields []*fieldDecl

	// Standalone marks an annotation that is attached to no declaration and
	// stands on its own. Only @response may.
	Standalone bool
}

// load loads the Go package at dir with the type information the parser and
// the resolver both need.
func load(dir string) (*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
	}

	// A bare relative path like "examples/petstore" would be treated as an
	// import path by go/packages; prefix it so it resolves as a directory.
	if !filepath.IsAbs(dir) && !strings.HasPrefix(dir, ".") {
		dir = "./" + dir
	}

	pkgs, err := packages.Load(cfg, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to load package: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found at path: %s", dir)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package has errors: %v", pkg.Errors)
	}
	return pkg, nil
}

// harvest collects the annotatable declarations of a loaded package, in source
// order.
func harvest(pkg *packages.Package) (*source, error) {
	src := &source{pkg: pkg, name: pkg.Name}

	for _, file := range pkg.Syntax {
		if src.api == nil {
			src.api = findAPIComment(pkg.Fset, file)
		}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						src.types = append(src.types, newTypeDecl(pkg.Fset, d, ts))
					}
				}

			case *ast.FuncDecl:
				fn := &funcDecl{
					Name: d.Name.Name,
					Pos:  pkg.Fset.Position(d.Name.Pos()),
					Doc:  commentBlock(pkg.Fset, d.Doc),
				}
				inlines, err := extractInlines(pkg, file, d)
				if err != nil {
					return nil, wrapf(fn.Pos, err, "in function %s", fn.Name)
				}
				fn.Inlines = inlines
				src.funcs = append(src.funcs, fn)
			}
		}
	}

	return src, nil
}

// findAPIComment returns the comment block carrying @api: the file's doc
// comment when it has one, otherwise the first free-floating block that does.
func findAPIComment(fset *token.FileSet, file *ast.File) *CommentBlock {
	if hasAPIAnnotation(file.Doc) {
		return commentBlock(fset, file.Doc)
	}
	for _, cg := range file.Comments {
		if hasAPIAnnotation(cg) {
			return commentBlock(fset, cg)
		}
	}
	return nil
}

func hasAPIAnnotation(cg *ast.CommentGroup) bool {
	if cg == nil {
		return false
	}
	for _, c := range cg.List {
		if strings.HasPrefix(commentText(c.Text), "@api") {
			return true
		}
	}
	return false
}

// newTypeDecl records one type declaration. The doc comment of the enclosing
// declaration wins over the spec's own, so both `// @schema` above a `type (`
// group and above a plain `type` are found.
func newTypeDecl(fset *token.FileSet, decl *ast.GenDecl, ts *ast.TypeSpec) *typeDecl {
	td := &typeDecl{
		Name:        ts.Name.Name,
		Pos:         fset.Position(ts.Name.Pos()),
		IsTypeAlias: ts.Assign.IsValid(),
		IsGeneric:   ts.TypeParams != nil && ts.TypeParams.NumFields() > 0,
	}

	if decl.Doc != nil {
		td.Doc = commentBlock(fset, decl.Doc)
	} else {
		td.Doc = commentBlock(fset, ts.Doc)
	}

	if td.IsTypeAlias {
		td.AliasOf = formatTypeExpr(ts.Type)
	}

	if st, ok := ts.Type.(*ast.StructType); ok {
		td.IsStruct = true
		td.Fields = structFields(fset, st)
	}

	return td
}

// structFields records the named fields of a struct literal in declaration
// order, descending into the anonymous structs they spell out. Embedded fields
// have no name to bind a @field annotation to, so they are skipped; the
// resolver flattens them from the Go type instead.
func structFields(fset *token.FileSet, st *ast.StructType) []*fieldDecl {
	if st.Fields == nil {
		return nil
	}

	var fields []*fieldDecl
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		decl := &fieldDecl{
			GoName: field.Names[0].Name,
			Pos:    fset.Position(field.Names[0].Pos()),
			Doc:    commentBlock(fset, field.Doc),
		}
		if nested := anonymousStructType(field.Type); nested != nil {
			decl.Fields = structFields(fset, nested)
		}
		fields = append(fields, decl)
	}
	return fields
}

// anonymousStructType returns the anonymous struct a field's type spells out,
// looking through the wrappers the resolver also looks through: a pointer, a
// slice or array element, a map value. Named types are not followed — those
// have their own declaration to carry annotations.
func anonymousStructType(expr ast.Expr) *ast.StructType {
	switch e := expr.(type) {
	case *ast.StructType:
		return e
	case *ast.StarExpr:
		return anonymousStructType(e.X)
	case *ast.ArrayType:
		return anonymousStructType(e.Elt)
	case *ast.MapType:
		return anonymousStructType(e.Value)
	}
	return nil
}

// commentBlock strips the comment markers from a comment group and keeps the
// non-empty lines with their positions.
func commentBlock(fset *token.FileSet, cg *ast.CommentGroup) *CommentBlock {
	if cg == nil {
		return nil
	}

	var lines []Line
	for _, c := range cg.List {
		text := commentText(c.Text)
		if text == "" {
			continue
		}
		lines = append(lines, Line{Text: text, Pos: fset.Position(c.Pos())})
	}

	if len(lines) == 0 {
		return nil
	}
	return &CommentBlock{Lines: lines, Pos: lines[0].Pos}
}

func commentText(text string) string {
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimPrefix(text, "/*")
	text = strings.TrimSuffix(text, "*/")
	return strings.TrimSpace(text)
}

// inlineMarkers are the in-function declaration markers, in the order they are
// looked for. The first line of a comment block that starts with one of them
// decides what the declaration is.
var inlineMarkers = []string{"@query", "@path", "@header", "@cookie", "@request", "@response"}

// extractInlines finds the annotations in a function body, in declaration
// order: those attached to a var or type declaration, and those standing on
// their own.
func extractInlines(pkg *packages.Package, file *ast.File, fn *ast.FuncDecl) ([]*inlineDecl, error) {
	if fn.Body == nil {
		return nil, nil
	}

	attached, err := attachedInlines(pkg, fn.Body)
	if err != nil {
		return nil, err
	}

	standalone, err := standaloneInlines(pkg, file, fn)
	if err != nil {
		return nil, err
	}

	inlines := append(attached, standalone...)
	slices.SortFunc(inlines, func(a, b *inlineDecl) int {
		return cmp.Compare(a.Pos.Offset, b.Pos.Offset)
	})
	return inlines, nil
}

// attachedInlines finds the annotated var and type declarations of a function
// body, in declaration order.
func attachedInlines(pkg *packages.Package, body *ast.BlockStmt) ([]*inlineDecl, error) {
	var inlines []*inlineDecl
	for _, genDecl := range collectGenDecls(body) {
		doc := commentBlock(pkg.Fset, genDecl.Doc)
		if doc == nil {
			continue
		}

		marker, at := findInlineMarker(doc.Lines)
		if marker == "" {
			continue
		}

		ident, structType := declaredStruct(genDecl)
		if ident == nil || structType == nil {
			continue
		}

		pos := pkg.Fset.Position(ident.Pos())
		st, err := structTypeOf(pkg, ident, pos)
		if err != nil {
			return nil, err
		}

		inline := &inlineDecl{
			Marker:  marker,
			VarName: ident.Name,
			Pos:     pos,
			Lines:   doc.Lines[at:],
			Struct:  st,
			Fields:  structFields(pkg.Fset, structType),
		}
		if marker == "@response" {
			inline.Status = responseStatus(doc.Lines[at].Text)
		}
		inlines = append(inlines, inline)
	}

	return inlines, nil
}

// standaloneInlines finds the annotations in a function body that are attached
// to no declaration at all.
//
// Only @response may stand on its own — it names its body rather than having a
// struct spell it out. Every other annotation needs the declaration below it to
// mean anything, so writing one here is a mistake rather than a no-op.
func standaloneInlines(pkg *packages.Package, file *ast.File, fn *ast.FuncDecl) ([]*inlineDecl, error) {
	var inlines []*inlineDecl

	for _, cg := range floatingComments(file, fn) {
		lines := commentBlock(pkg.Fset, cg).AnnotationLines()
		if len(lines) == 0 {
			continue
		}

		name := extractAnnotationName(lines[0].Text)
		if Grammar[name] == nil && InlineGrammar[name] == nil {
			// Not an annotation at all: ordinary prose between statements.
			continue
		}
		if name != "@response" {
			return nil, errorf(lines[0].Pos,
				"%s is attached to no declaration; annotations in a function body describe the var or type declared under them", name)
		}

		inlines = append(inlines, &inlineDecl{
			Marker:     "@response",
			Status:     responseStatus(lines[0].Text),
			Pos:        lines[0].Pos,
			Lines:      lines,
			Standalone: true,
		})
	}

	return inlines, nil
}

// floatingComments returns the comment groups inside a function body that no
// node claims as its own. Walking the function visits every comment attached
// to a declaration, a spec or a struct field, so whatever is left over is
// floating between statements.
func floatingComments(file *ast.File, fn *ast.FuncDecl) []*ast.CommentGroup {
	attached := make(map[*ast.CommentGroup]bool)
	ast.Inspect(fn, func(node ast.Node) bool {
		if cg, ok := node.(*ast.CommentGroup); ok {
			attached[cg] = true
		}
		return true
	})

	var floating []*ast.CommentGroup
	for _, cg := range file.Comments {
		if cg.Pos() < fn.Body.Pos() || cg.End() > fn.Body.End() {
			continue
		}
		if !attached[cg] {
			floating = append(floating, cg)
		}
	}
	return floating
}

// findInlineMarker returns the marker a comment block declares and the index of
// the line declaring it.
func findInlineMarker(lines []Line) (string, int) {
	for i, line := range lines {
		text := strings.TrimSpace(line.Text)
		for _, marker := range inlineMarkers {
			if strings.HasPrefix(text, marker) {
				return marker, i
			}
		}
	}
	return "", 0
}

// responseStatus returns the status an @response marker line declares, keeping
// the literal text so "default" and "4XX" survive. An @response with no status
// means 200.
func responseStatus(line string) string {
	if fields := strings.Fields(ExtractMetadata(line, "@response")); len(fields) > 0 {
		return fields[0]
	}
	return "200"
}

// declaredStruct returns the identifier and struct literal a var or type
// declaration declares. Anything else (a non-struct type, a grouped
// declaration) yields nils and is skipped.
func declaredStruct(genDecl *ast.GenDecl) (*ast.Ident, *ast.StructType) {
	var ident *ast.Ident
	var structType *ast.StructType

	for _, spec := range genDecl.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec: // var x struct { ... }
			if len(s.Names) == 0 {
				continue
			}
			ident = s.Names[0]
			if st, ok := s.Type.(*ast.StructType); ok {
				structType = st
			}
		case *ast.TypeSpec: // type X struct { ... }
			ident = s.Name
			if st, ok := s.Type.(*ast.StructType); ok {
				structType = st
			}
		}
	}

	return ident, structType
}

// structTypeOf resolves the Go type of an inline declaration. It works for both
// `var x struct{...}` and `type X struct{...}`: taking the underlying type of
// either object yields the *types.Struct.
func structTypeOf(pkg *packages.Package, ident *ast.Ident, pos token.Position) (*types.Struct, error) {
	if pkg.TypesInfo == nil {
		return nil, errorf(pos, "type info unavailable for inline struct %q", ident.Name)
	}
	obj := pkg.TypesInfo.Defs[ident]
	if obj == nil {
		return nil, errorf(pos, "could not resolve type for inline struct %q", ident.Name)
	}
	st, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		return nil, errorf(pos, "inline declaration %q is not a struct", ident.Name)
	}
	return st, nil
}

// collectGenDecls collects the declarations of a function body: the top-level
// ones, plus those in a function literal the body returns. That second case is
// the handler-factory pattern, where the response struct lives in the returned
// closure.
//
// Deliberately shallow: declarations nested any deeper (inside an if, a loop,
// or a closure that is assigned rather than returned) are not found.
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
				fl, ok := result.(*ast.FuncLit)
				if !ok || fl.Body == nil {
					continue
				}
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

	return decls
}

// formatTypeExpr renders a type expression back to source-like text, e.g. an
// *ast.IndexExpr for Foo[Bar] becomes "Foo[Bar]".
func formatTypeExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
		return e.Sel.Name
	case *ast.StarExpr:
		return "*" + formatTypeExpr(e.X)
	case *ast.ArrayType:
		return "[]" + formatTypeExpr(e.Elt)
	case *ast.MapType:
		return "map[" + formatTypeExpr(e.Key) + "]" + formatTypeExpr(e.Value)
	case *ast.IndexExpr:
		return formatTypeExpr(e.X) + "[" + formatTypeExpr(e.Index) + "]"
	case *ast.IndexListExpr:
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
