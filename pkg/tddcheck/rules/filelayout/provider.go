package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func providerViolations(fileSet *token.FileSet, filename string, name fileName, parsedFile *ast.File) []Violation {
	var violations []Violation
	expectedProviderType := upperCamelName(name.scope) + "Provider"
	providerTypeFound := false
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			fileViolations, found := providerGenDeclViolations(fileSet, filename, typed, expectedProviderType)
			violations = append(violations, fileViolations...)
			providerTypeFound = providerTypeFound || found
		case *ast.FuncDecl:
			violations = append(violations, providerFuncViolations(fileSet, filename, typed, expectedProviderType)...)
		}
	}
	if !providerTypeFound {
		violations = append(violations, Violation{
			File:    displayFilename(filename),
			Line:    1,
			Message: "provider files must declare " + expectedProviderType,
		})
	}
	return violations
}

func providerGenDeclViolations(fileSet *token.FileSet, filename string, decl *ast.GenDecl, expectedProviderType string) ([]Violation, bool) {
	var violations []Violation
	providerTypeFound := false
	switch decl.Tok {
	case token.IMPORT:
		for _, importSpec := range decl.Specs {
			spec, ok := importSpec.(*ast.ImportSpec)
			if ok && forbiddenProviderImport(importPath(spec)) {
				violations = append(violations, violationAt(fileSet, filename, spec.Pos(), "provider files must not import "+importPath(spec)))
			}
		}
	case token.TYPE:
		for _, spec := range decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if typeNameContains(typeSpec.Name.Name, "Provider") {
				if typeSpec.Name.Name != expectedProviderType {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "provider files must declare "+expectedProviderType+" as their provider type"))
				} else {
					providerTypeFound = true
				}
			}
			if forbiddenServiceModelType(typeSpec.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "provider type "+typeSpec.Name.Name+" must not use transport or command suffixes"))
			}
			if hasStructTag(typeSpec, "bun") || hasStructTag(typeSpec, "json") || hasStructTag(typeSpec, "query") || hasStructTag(typeSpec, "path") {
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "provider types must not declare transport or persistence tags"))
			}
		}
	case token.CONST, token.VAR:
		violations = append(violations, violationAt(fileSet, filename, decl.Pos(), "provider files must only declare provider types and functions"))
	default:
		violations = append(violations, violationAt(fileSet, filename, decl.Pos(), "provider files must only declare provider types and functions"))
	}
	return violations, providerTypeFound
}

func providerFuncViolations(fileSet *token.FileSet, filename string, decl *ast.FuncDecl, expectedProviderType string) []Violation {
	if decl.Recv != nil {
		if receiverTypeName(decl.Recv) != expectedProviderType {
			return []Violation{violationAt(fileSet, filename, decl.Pos(), "provider receiver methods must use "+expectedProviderType)}
		}
		return nil
	}
	if !strings.HasPrefix(decl.Name.Name, "New") {
		return []Violation{violationAt(fileSet, filename, decl.Pos(), "provider package-level functions must start with New")}
	}
	if !constructorReturnsProvider(decl, expectedProviderType) {
		return []Violation{violationAt(fileSet, filename, decl.Pos(), "provider constructors must return "+expectedProviderType)}
	}
	return nil
}

func constructorReturnsProvider(decl *ast.FuncDecl, expectedProviderType string) bool {
	if decl.Type.Results == nil {
		return false
	}
	for _, result := range decl.Type.Results.List {
		if resultTypeName(result.Type) == expectedProviderType {
			return true
		}
	}
	return false
}

func resultTypeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return resultTypeName(typed.X)
	default:
		return ""
	}
}

func forbiddenProviderImport(importPath string) bool {
	return forbiddenServiceModelImport(importPath) ||
		containsImport(importPath, "/internal/adapter") ||
		containsImport(importPath, "/internal/runtime") ||
		containsImport(importPath, "/internal/repository")
}

func typeNameContains(name string, part string) bool {
	return strings.Contains(name, part)
}

func containsImport(importPath string, part string) bool {
	return strings.Contains(importPath, part)
}
