package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func validationViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			if typed.Recv != nil || !(strings.HasPrefix(typed.Name.Name, "validate") || strings.HasPrefix(typed.Name.Name, "normalize")) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "validation functions must be package-level validate* or normalize*"))
			}
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT || typed.Tok == token.CONST || typed.Tok == token.VAR {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "validation files must not declare types"))
		}
	}
	return violations
}
