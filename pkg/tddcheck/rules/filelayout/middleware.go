package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func architectureMiddlewareViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			if typed.Tok != token.TYPE {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture middleware files must only declare middleware types and functions"))
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && !architectureMiddlewareType(typeSpec.Name.Name) {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "architecture middleware types must be Middleware or end with Middleware"))
				}
			}
		case *ast.FuncDecl:
			if typed.Recv != nil {
				if receiverTypeName(typed.Recv) != "Endpoint" {
					violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture middleware receiver methods must use Endpoint"))
				}
				continue
			}
			if !strings.Contains(strings.ToLower(typed.Name.Name), "middleware") && !lowerCamelIdentifier(typed.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture middleware functions must be middleware helpers or private helpers"))
			}
		}
	}
	return violations
}

func architectureMiddlewareType(name string) bool {
	return name == "Middleware" || strings.HasSuffix(name, "Middleware")
}
