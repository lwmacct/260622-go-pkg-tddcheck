package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func commandsViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			violations = append(violations, violationAt(fileSet, filename, decl.Pos(), "commands files must only declare types"))
			continue
		}
		if genDecl.Tok == token.IMPORT {
			continue
		}
		if genDecl.Tok != token.TYPE {
			violations = append(violations, violationAt(fileSet, filename, genDecl.Pos(), "commands files must only declare types"))
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && !commandTypeName(typeSpec.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "command type "+typeSpec.Name.Name+" must end with Request, Response, Result, or Item"))
			}
		}
	}
	return violations
}

func commandTypeName(name string) bool {
	return strings.HasSuffix(name, "Request") ||
		strings.HasSuffix(name, "Response") ||
		strings.HasSuffix(name, "Result") ||
		strings.HasSuffix(name, "Item")
}
