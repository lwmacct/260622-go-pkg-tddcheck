package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func architectureContextViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			if typed.Tok != token.TYPE {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture context files must only declare context key types and package-level functions"))
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && !architectureContextType(typeSpec.Name.Name) {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "architecture context types must be private and end with Key"))
				}
			}
		case *ast.FuncDecl:
			if typed.Recv != nil {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture context functions must not use receivers"))
				continue
			}
			if !architectureContextFunction(typed.Name.Name) && !lowerCamelIdentifier(typed.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture context functions must be Context* or *FromContext helpers"))
			}
		}
	}
	return violations
}

func architectureContextType(name string) bool {
	return lowerCamelIdentifier(name) && strings.HasSuffix(name, "Key")
}

func architectureContextFunction(name string) bool {
	return strings.HasPrefix(name, "Context") || strings.HasSuffix(name, "FromContext")
}
