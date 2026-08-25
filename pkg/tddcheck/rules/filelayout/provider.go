package filelayout

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func providerViolations(file rulekit.GoFile, name fileName, initialisms map[string]string) []Violation {
	var violations []Violation
	expectedProviderType := upperCamelNameWithInitialisms(name.Qualifier(), initialisms) + "Provider"
	providerTypeFound := false
	for _, decl := range file.AST.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			fileViolations, found := providerGenDeclViolations(file, typed, expectedProviderType)
			violations = append(violations, fileViolations...)
			providerTypeFound = providerTypeFound || found
		case *ast.FuncDecl:
			violations = append(violations, providerFuncViolations(file.Fset, file.AbsPath, typed, expectedProviderType)...)
		}
	}
	if !providerTypeFound {
		violations = append(violations, Violation{
			File:    displayFilename(file.AbsPath),
			Line:    1,
			Message: "provider files must declare " + expectedProviderType,
		})
	}
	return violations
}

func providerGenDeclViolations(file rulekit.GoFile, decl *ast.GenDecl, expectedProviderType string) ([]Violation, bool) {
	var violations []Violation
	providerTypeFound := false
	switch decl.Tok {
	case token.IMPORT:
		for _, importSpec := range decl.Specs {
			spec, ok := importSpec.(*ast.ImportSpec)
			if ok && forbiddenProviderImport(file, importPath(spec)) {
				violations = append(violations, violationAt(file.Fset, file.AbsPath, spec.Pos(), "provider files must not import "+importPath(spec)))
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
					violations = append(violations, violationAt(file.Fset, file.AbsPath, typeSpec.Pos(), "provider files must declare "+expectedProviderType+" as their provider type"))
				} else {
					providerTypeFound = true
				}
			}
			if hasStructTag(typeSpec, "bun") {
				violations = append(violations, violationAt(file.Fset, file.AbsPath, typeSpec.Pos(), "provider types must not declare persistence tags"))
			}
		}
	case token.CONST, token.VAR:
		violations = append(violations, violationAt(file.Fset, file.AbsPath, decl.Pos(), "provider files must only declare provider types and functions"))
	default:
		violations = append(violations, violationAt(file.Fset, file.AbsPath, decl.Pos(), "provider files must only declare provider types and functions"))
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

func forbiddenProviderImport(file rulekit.GoFile, importPath string) bool {
	for _, imported := range file.Imports {
		if imported.Path != importPath && imported.PackagePath != importPath {
			continue
		}
		return imported.ModuleLocal && imported.TargetLayer != "service"
	}
	return false
}

func typeNameContains(name string, part string) bool {
	return strings.Contains(name, part)
}
