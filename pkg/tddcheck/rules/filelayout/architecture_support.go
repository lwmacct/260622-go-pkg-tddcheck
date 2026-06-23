package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func architectureSupportViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			if typed.Tok != token.TYPE && typed.Tok != token.CONST {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture support files must only declare support types, consts, and package-level functions"))
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && !architectureSupportType(typeSpec.Name.Name) {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "architecture support types must be Config, Options, Services, Dependencies, or end with Config/Routes/Auth"))
				}
			}
		case *ast.FuncDecl:
			if typed.Recv != nil {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture support functions must not use receivers"))
				continue
			}
			if !supportFunctionName(typed.Name.Name) && !lowerCamelIdentifier(typed.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture support functions must be support helpers or private helpers"))
			}
		}
	}
	return violations
}

func architectureSupportType(name string) bool {
	return oneOf(name, "Config", "Options", "Services", "Dependencies") ||
		strings.HasSuffix(name, "Config") ||
		strings.HasSuffix(name, "Routes") ||
		strings.HasSuffix(name, "Auth")
}
