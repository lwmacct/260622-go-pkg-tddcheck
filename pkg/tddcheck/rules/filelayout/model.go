package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func modelViolations(fileSet *token.FileSet, filename string, layer string, parsedFile *ast.File) []Violation {
	if layer == "repository" {
		return repositoryModelViolations(fileSet, filename, parsedFile)
	}
	return serviceModelViolations(fileSet, filename, parsedFile)
}

func serviceModelViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				for _, importSpec := range typed.Specs {
					spec, ok := importSpec.(*ast.ImportSpec)
					if ok && forbiddenServiceModelImport(importPath(spec)) {
						violations = append(violations, violationAt(fileSet, filename, spec.Pos(), "service model files must not import "+importPath(spec)))
					}
				}
				continue
			}
			if typed.Tok != token.TYPE {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "service model files must only declare types"))
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if forbiddenServiceModelType(typeSpec.Name.Name) {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "service model type "+typeSpec.Name.Name+" must not use transport or command suffixes"))
				}
				if hasStructTag(typeSpec, "bun") || hasStructTag(typeSpec, "json") || hasStructTag(typeSpec, "query") || hasStructTag(typeSpec, "path") {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "service model types must not declare transport or persistence tags"))
				}
			}
		case *ast.FuncDecl:
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "service model files must not declare functions"))
		}
	}
	return violations
}

func repositoryModelViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				for _, importSpec := range typed.Specs {
					spec, ok := importSpec.(*ast.ImportSpec)
					if ok && forbiddenRepositoryModelImport(importPath(spec)) {
						violations = append(violations, violationAt(fileSet, filename, spec.Pos(), "repository model files must not import "+importPath(spec)))
					}
				}
				continue
			}
			if typed.Tok == token.TYPE {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository model files must only declare types and model receiver methods"))
		case *ast.FuncDecl:
			if typed.Recv != nil {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository model files must not declare package-level functions"))
		}
	}
	return violations
}

func forbiddenServiceModelImport(importPath string) bool {
	return strings.Contains(importPath, "/internal/handler") ||
		oneOf(
			importPath,
			"net/http",
			"github.com/danielgtaylor/huma/v2",
			"github.com/uptrace/bun",
			"gorm.io/gorm",
		)
}

func forbiddenServiceModelType(name string) bool {
	return strings.HasSuffix(name, "DTO") ||
		strings.HasSuffix(name, "DTOs") ||
		strings.HasSuffix(name, "Request") ||
		strings.HasSuffix(name, "Response") ||
		strings.HasSuffix(name, "Result") ||
		strings.HasSuffix(name, "Item")
}

func forbiddenRepositoryModelImport(importPath string) bool {
	return strings.Contains(importPath, "/internal/handler") ||
		strings.Contains(importPath, "/internal/service") ||
		oneOf(
			importPath,
			"net/http",
			"github.com/danielgtaylor/huma/v2",
		)
}
