package filelayout

import (
	"go/ast"
	"go/token"
)

func repositoryModelViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT || typed.Tok == token.TYPE {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository model files must only declare types and model receiver methods"))
		case *ast.FuncDecl:
			if typed.Recv != nil {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository model files must not declare package-level functions"))
		}
	}
	return violations
}
