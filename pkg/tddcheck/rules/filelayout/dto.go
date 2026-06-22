package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func dtoViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "dto files must not declare functions"))
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			if typed.Tok != token.TYPE {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "dto files must only declare DTO types"))
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && !strings.HasSuffix(typeSpec.Name.Name, "DTO") && !strings.HasSuffix(typeSpec.Name.Name, "DTOs") {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "DTO type "+typeSpec.Name.Name+" must end with DTO or DTOs"))
				}
			}
		}
	}
	return violations
}
