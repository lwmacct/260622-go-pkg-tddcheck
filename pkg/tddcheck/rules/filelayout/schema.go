package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func schemaViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Recv == nil {
			if !oneOf(funcDecl.Name.Name, "Schema", "CreateIndexes", "CreateSchema", "EnsureSchema") &&
				!strings.HasSuffix(funcDecl.Name.Name, "Schema") {
				violations = append(violations, violationAt(fileSet, filename, funcDecl.Pos(), "schema package-level functions must be schema lifecycle functions"))
			}
		}
	}
	return violations
}
