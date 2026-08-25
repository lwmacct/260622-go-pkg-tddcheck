package filelayout

import (
	"go/ast"
	"go/types"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func typesViolations(file rulekit.GoFile, name fileName, initialisms map[string]string) []Violation {
	var violations []Violation
	for _, decl := range file.AST.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			violations = append(violations, declarationGenDeclViolations(file, name, "types", typed, initialisms)...)
		case *ast.FuncDecl:
			violations = append(violations, typesFuncViolations(file, typed)...)
		}
	}
	return violations
}

func typesFuncViolations(file rulekit.GoFile, decl *ast.FuncDecl) []Violation {
	if decl.Recv == nil || (decl.Name.Name != "Error" && decl.Name.Name != "Unwrap") {
		return []Violation{violationAt(file.Fset, file.AbsPath, decl.Pos(), "types files must only declare types, consts, Err* vars, and Error/Unwrap methods")}
	}
	if receiverImplementsError(file, decl) {
		return nil
	}
	return []Violation{violationAt(file.Fset, file.AbsPath, decl.Pos(), "types Error/Unwrap methods must use receivers that implement error")}
}

func receiverImplementsError(file rulekit.GoFile, decl *ast.FuncDecl) bool {
	if file.TypesInfo == nil {
		return true
	}
	object, ok := file.TypesInfo.Defs[decl.Name].(*types.Func)
	if !ok {
		return true
	}
	signature, ok := object.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	errorType, ok := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	return ok && types.Implements(signature.Recv().Type(), errorType)
}
