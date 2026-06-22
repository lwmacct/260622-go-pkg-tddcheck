package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func repositoryViolations(fileSet *token.FileSet, filename string, name fileName, parsedFile *ast.File) []Violation {
	var violations []Violation
	if !strings.HasPrefix(name.scope, architectureScopePrefix) {
		violations = append(violations, Violation{
			File:    displayFilename(filename),
			Line:    1,
			Message: "repository files must use an architecture scope",
		})
	}

	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT || typed.Tok == token.TYPE || typed.Tok == token.CONST || typed.Tok == token.VAR {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository files must only declare infrastructure types, values, and functions"))
		case *ast.FuncDecl:
			if typed.Recv != nil {
				if name.scope != "x_store" || receiverTypeName(typed.Recv) != "Store" {
					violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository receiver methods must belong to x_store Store"))
				}
				continue
			}
			if !repositoryFunctionName(typed.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository package-level functions must be infrastructure entry points or private helpers"))
			}
		}
	}
	return violations
}

func repositoryFunctionName(name string) bool {
	return lowerCamelIdentifier(name) ||
		strings.HasPrefix(name, "New") ||
		strings.HasPrefix(name, "Open") ||
		strings.HasPrefix(name, "Build") ||
		strings.HasPrefix(name, "Apply") ||
		strings.HasPrefix(name, "With") ||
		strings.HasPrefix(name, "Ensure") ||
		strings.HasPrefix(name, "Create") ||
		strings.HasSuffix(name, "Schema")
}
