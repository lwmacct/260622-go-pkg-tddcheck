package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func utilsViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "utils files must only declare util* functions"))
		case *ast.FuncDecl:
			if typed.Recv != nil {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "utils functions must not use receivers"))
			}
			if !strings.HasPrefix(typed.Name.Name, "util") {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "utils function "+typed.Name.Name+" must start with util"))
			}
		}
	}
	return violations
}
