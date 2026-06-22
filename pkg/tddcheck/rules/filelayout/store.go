package filelayout

import (
	"go/ast"
	"go/token"
)

func storeViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "store files must only declare Store receiver methods"))
		case *ast.FuncDecl:
			if receiverTypeName(typed.Recv) != "Store" {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "store files must only declare Store receiver methods"))
			}
		}
	}
	return violations
}
