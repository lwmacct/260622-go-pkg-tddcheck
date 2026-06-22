package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func mapperViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				for _, importSpec := range typed.Specs {
					spec, ok := importSpec.(*ast.ImportSpec)
					if !ok {
						continue
					}
					importPath := strings.Trim(spec.Path.Value, `"`)
					if forbiddenMapperImport(importPath) {
						violations = append(violations, violationAt(fileSet, filename, spec.Pos(), "mapper files must not import "+importPath))
					}
				}
			} else {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "mapper files must only declare functions"))
			}
		case *ast.FuncDecl:
			if typed.Recv != nil {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "mapper functions must not use receivers"))
			}
			if !strings.HasPrefix(typed.Name.Name, "To") {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "mapper function "+typed.Name.Name+" must start with To"))
			}
		}
	}
	return violations
}

func forbiddenMapperImport(importPath string) bool {
	return oneOf(
		importPath,
		"context",
		"database/sql",
		"net/http",
		"github.com/danielgtaylor/huma/v2",
		"github.com/uptrace/bun",
		"gorm.io/gorm",
	)
}
