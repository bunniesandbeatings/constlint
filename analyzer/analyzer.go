// Package analyzer provides a linter that checks for writes to struct fields
// marked with "// +const" comments.
package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	astinspector "golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the main entry point for the linter.
var Analyzer = &analysis.Analyzer{
	Name:     "const",
	Doc:      "checks for writes to struct fields marked with // +const",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

// constField represents a field that should be treated as constant.
type constField struct {
	structType *types.TypeName
	fieldName  string
}

// constParam represents a parameter that should be treated as constant.
type constParam struct {
	funcName    string
	paramName   string
	packagePath string
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*astinspector.Inspector)

	// Maps to store const fields and parameters
	constFields := make(map[constField]token.Pos)

	// First pass: find all struct fields marked with // +const
	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
	}

	inspector.Preorder(nodeFilter, func(n ast.Node) {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return
		}

		// Get the type object for this struct
		obj := pass.TypesInfo.Defs[typeSpec.Name]
		if obj == nil {
			return
		}

		typeName, ok := obj.(*types.TypeName)
		if !ok {
			return
		}

		// Check each field for the +const comment
		for _, field := range structType.Fields.List {
			if field.Doc == nil && field.Comment == nil {
				continue
			}

			var hasConstMarker bool
			// Check doc comments
			if field.Doc != nil {
				for _, comment := range field.Doc.List {
					if strings.Contains(comment.Text, "+const") {
						hasConstMarker = true
						break
					}
				}
			}

			// Check inline comments
			if !hasConstMarker && field.Comment != nil {
				for _, comment := range field.Comment.List {
					if strings.Contains(comment.Text, "+const") {
						hasConstMarker = true
						break
					}
				}
			}

			if hasConstMarker {
				for _, name := range field.Names {
					constFields[constField{
						structType: typeName,
						fieldName:  name.Name,
					}] = name.Pos()
				}
			}
		}
	})

	// Find all function parameters marked with // +const:[param1,param2,...]
	constParams := make(map[constParam]token.Pos)
	funcFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspector.Preorder(funcFilter, func(n ast.Node) {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Doc == nil {
			return
		}

		// Look for +const: comment
		var constParamList string
		for _, comment := range funcDecl.Doc.List {
			text := comment.Text
			constIndex := strings.Index(text, "// +const:[")
			if constIndex != -1 {
				startIdx := constIndex + len("// +const:[")
				endIdx := strings.Index(text[startIdx:], "]")
				if endIdx != -1 {
					constParamList = text[startIdx : startIdx+endIdx]
					break
				}
			}
		}

		if constParamList == "" {
			return
		}

		// Parse the parameter list
		paramNames := strings.Split(constParamList, ",")
		for i := range paramNames {
			paramNames[i] = strings.TrimSpace(paramNames[i])
		}

		// Get function name and package path
		funcName := funcDecl.Name.Name
		packagePath := pass.Pkg.Path()

		// Mark each parameter as const
		for _, paramName := range paramNames {
			constParams[constParam{
				funcName:    funcName,
				paramName:   paramName,
				packagePath: packagePath,
			}] = funcDecl.Pos()
		}
	})

	assignFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
	}

	inspector.Preorder(assignFilter, func(n ast.Node) {
		assignStmt, ok := n.(*ast.AssignStmt)
		if !ok {
			return
		}

		// Skip declarations (var x = y)
		if assignStmt.Tok == token.DEFINE {
			return
		}

		// Check each LHS of the assignment
		for _, lhs := range assignStmt.Lhs {
			checkFieldAssignment(pass, lhs, constFields)
			checkParamAssignment(pass, lhs, constParams)
		}
	})

	return nil, nil
}

func checkAssignment(pass *analysis.Pass, expr ast.Expr, constFields map[constField]token.Pos) {
	// We're looking for field selections (x.y = z)
	selExpr, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Get the type information
	selection, ok := pass.TypesInfo.Selections[selExpr]
	if !ok {
		return
	}

	// Only interested in field selections
	if selection.Kind() != types.FieldVal {
		return
	}

	// Get the receiver type
	recvType := selection.Recv()
	if recvType == nil {
		return
	}

	// Get the named type (dereference pointers if needed)
	var namedType *types.Named
	switch t := recvType.(type) {
	case *types.Named:
		namedType = t
	case *types.Pointer:
		if named, ok := t.Elem().(*types.Named); ok {
			namedType = named
		} else {
			return
		}
	default:
		return
	}

	// Get the type name
	typeName := namedType.Obj()
	fieldName := selExpr.Sel.Name

	// Check if this is a const field
	cf := constField{
		structType: typeName,
		fieldName:  fieldName,
	}

	if fieldPos, exists := constFields[cf]; exists {
		// Now we need to determine if we're in a constructor
		if !isInConstructor(pass, selExpr, namedType) {
			pass.Reportf(selExpr.Pos(), "assignment to const field %s.%s (marked with // +const at %s)",
				typeName.Name(), fieldName, pass.Fset.Position(fieldPos))
		}
	}
}

// Rename checkAssignment to checkFieldAssignment for clarity
func checkFieldAssignment(pass *analysis.Pass, expr ast.Expr, constFields map[constField]token.Pos) {
	checkAssignment(pass, expr, constFields)
}

// checkParamAssignment checks if a parameter marked as const is being modified
func checkParamAssignment(pass *analysis.Pass, expr ast.Expr, constParams map[constParam]token.Pos) {
	// Get the identifier being assigned to
	var ident *ast.Ident
	switch e := expr.(type) {
	case *ast.Ident:
		ident = e
	default:
		return
	}

	// Find the enclosing function
	path, found := astPath(pass.Files, expr)
	if !found {
		return
	}

	var funcDecl *ast.FuncDecl
	for i := len(path) - 1; i >= 0; i-- {
		if fd, ok := path[i].(*ast.FuncDecl); ok {
			funcDecl = fd
			break
		}
	}

	if funcDecl == nil {
		return
	}

	// Check if this identifier is a parameter in the function
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil || obj.Pos() == token.NoPos {
		return
	}

	// Check if this parameter is marked as const
	cp := constParam{funcName: funcDecl.Name.Name, paramName: ident.Name, packagePath: pass.Pkg.Path()}
	if paramPos, exists := constParams[cp]; exists {
		pass.Reportf(ident.Pos(), "assignment to const parameter %s (marked with // +const at %s)",
			ident.Name, pass.Fset.Position(paramPos))
	}
}

// isInConstructor checks if the given expression is within a constructor method/function
func isInConstructor(pass *analysis.Pass, expr ast.Expr, namedType *types.Named) bool {
	// Find the enclosing function
	path, _ := astPath(pass.Files, expr)
	var funcDecl *ast.FuncDecl
	for i := len(path) - 1; i >= 0; i-- {
		if fd, ok := path[i].(*ast.FuncDecl); ok {
			funcDecl = fd
			break
		}
	}

	if funcDecl == nil {
		return false
	}

	// Check if this is a method of the struct
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) == 1 {
		recvType := pass.TypesInfo.TypeOf(funcDecl.Recv.List[0].Type)
		if recvType == nil {
			return false
		}

		// Handle pointer receiver
		if ptr, ok := recvType.(*types.Pointer); ok {
			recvType = ptr.Elem()
		}

		// Check if the receiver is our struct type
		if recvType == namedType.Underlying() {
			// This is a method of our struct
			return isConstructorName(funcDecl.Name.Name) || returnsOwnType(pass, funcDecl, namedType)
		}
	}

	// Check if this is a function that returns the struct type
	return returnsOwnType(pass, funcDecl, namedType)
}

// isConstructorName checks if the function name follows constructor naming conventions
func isConstructorName(name string) bool {
	constructorPrefixes := []string{"New", "Create", "Init", "Make"}
	for _, prefix := range constructorPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// returnsOwnType checks if the function returns the struct type
func returnsOwnType(pass *analysis.Pass, funcDecl *ast.FuncDecl, namedType *types.Named) bool {
	if funcDecl.Type.Results == nil {
		return false
	}

	for _, result := range funcDecl.Type.Results.List {
		resultType := pass.TypesInfo.TypeOf(result.Type)
		if resultType == nil {
			continue
		}

		// Handle pointer return type
		if ptr, ok := resultType.(*types.Pointer); ok {
			resultType = ptr.Elem()
		}

		if resultType == namedType {
			return true
		}
	}

	return false
}

// astPath returns the path from the root of the AST to the given node
func astPath(files []*ast.File, target ast.Node) ([]ast.Node, bool) {
	var path []ast.Node
	found := false

	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			if found {
				return false
			}
			if n == target {
				found = true
				return false
			}
			if n != nil {
				path = append(path, n)
			}
			return true
		})
		if found {
			break
		}
		path = nil
	}

	return path, found
}
