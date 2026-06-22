package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func errorsViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			name := typed.Name.Name
			if typed.Recv != nil {
				if !oneOf(name, "Error", "Is", "As", "Unwrap") {
					violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "error receiver methods must be Error, Is, As, or Unwrap"))
				}
				continue
			}
			if !(strings.HasPrefix(name, "Wrap") || strings.HasPrefix(name, "Is") || strings.HasPrefix(name, "As")) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "error helper functions must start with Wrap, Is, or As"))
			}
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT || typed.Tok == token.VAR || typed.Tok == token.TYPE {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "errors files must only declare error vars, error types, and error helpers"))
		}
	}
	return violations
}
