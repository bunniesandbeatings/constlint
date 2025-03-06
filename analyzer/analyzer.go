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
	Doc:      "checks for writes to struct fields marked with // +const", // TODO: improve doc field, include new markers
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

	// First pass: find all struct fields and function parameters marked with // +const
	constFields := make(map[constField]token.Pos)
	constParams := make(map[constParam]token.Pos)
	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.FuncDecl)(nil),
	}
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.TypeSpec:
			structType, ok := node.Type.(*ast.StructType)
			if !ok {
				return
			}

			// Get the type object for this struct
			obj := pass.TypesInfo.Defs[node.Name]
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

		case *ast.FuncDecl:
			if node.Doc == nil {
				return
			}

			// Look for +const comment
			var constParamList string
			var allParamsConst bool
			
			for _, comment := range node.Doc.List {
				text := comment.Text
				
				// Check for +const:[param1,param2] format
				constIndex := strings.Index(text, "// +const:[")
				if constIndex != -1 {
					startIdx := constIndex + len("// +const:[")
					endIdx := strings.Index(text[startIdx:], "]")
					if endIdx != -1 {
						constParamList = text[startIdx : startIdx+endIdx]
						break
					}
				}
				
				// Check for standalone +const marker (all params are const)
				if strings.TrimSpace(text) == "// +const" {
					allParamsConst = true
					break
				}
			}

			// If neither format was found, return
			if constParamList == "" && !allParamsConst {
				return
			}

			// Get all parameter names if allParamsConst is true
			var paramNames []string
			if allParamsConst {
				// Get all parameter names from the function
				if node.Type.Params != nil {
					for _, field := range node.Type.Params.List {
						for _, name := range field.Names {
							paramNames = append(paramNames, name.Name)
						}
					}
				}
			} else {
				// Parse the parameter list from the comment
				paramNames = strings.Split(constParamList, ",")
				for i := range paramNames {
					paramNames[i] = strings.TrimSpace(paramNames[i])
				}
			}

			// Get function name and package path
			funcName := node.Name.Name
			packagePath := pass.Pkg.Path()

			// Mark each parameter as const
			for _, paramName := range paramNames {
				constParams[constParam{
					funcName:    funcName,
					paramName:   paramName,
					packagePath: packagePath,
				}] = node.Pos()
			}
		}
	})

	// Second pass: locate mutations of constant fields or params
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
		if !isInstanciator(pass, selExpr, namedType) {
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

func isInstanciator(pass *analysis.Pass, expr ast.Expr, namedType *types.Named) bool {
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

	// Check if the function contains a composite literal of the struct type
	foundInstantiation := false
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		if foundInstantiation {
			return false
		}

		// Look for composite literals
		if compLit, ok := n.(*ast.CompositeLit); ok {
			// Get the type of the composite literal
			litType := pass.TypesInfo.TypeOf(compLit.Type)
			if litType == nil {
				return true
			}

			// Handle pointer types
			if ptr, ok := litType.(*types.Pointer); ok {
				litType = ptr.Elem()
			}

			// Check if it's our struct type
			if types.Identical(litType, namedType) {
				foundInstantiation = true
				return false
			}
		}
		return true
	})

	return foundInstantiation
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
