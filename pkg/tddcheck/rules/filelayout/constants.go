package filelayout

import (
	"go/ast"
	"go/token"
)

func constantsViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && (genDecl.Tok == token.IMPORT || genDecl.Tok == token.CONST) {
			continue
		}
		violations = append(violations, violationAt(fileSet, filename, decl.Pos(), "constants files must only declare const"))
	}
	return violations
}
