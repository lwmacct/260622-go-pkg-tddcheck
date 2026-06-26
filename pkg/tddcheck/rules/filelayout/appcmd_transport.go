package filelayout

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func appcmdTransportViolations(context *rulekit.Context) []Violation {
	if !context.Profile.DependencyLayer("appcmd") {
		return nil
	}
	var violations []Violation
	for _, file := range context.Files {
		if !isAppcmdFile(file) {
			continue
		}
		violations = append(violations, appcmdTransportViolationsInFile(file)...)
	}
	return violations
}

func appcmdTransportViolationsInFile(file rulekit.GoFile) []Violation {
	var violations []Violation
	humaNames := map[string]bool{}
	for _, decl := range file.AST.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		for _, spec := range genDecl.Specs {
			importSpec, ok := spec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			if importPath(importSpec) != "github.com/danielgtaylor/huma/v2" {
				continue
			}
			violations = append(violations, violationAt(file.Fset, file.AbsPath, importSpec.Pos(), "appcmd must not import huma; register transport routes in handler"))
			humaNames[importAlias(importSpec, "huma")] = true
		}
	}
	for _, decl := range file.AST.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if transportContractType(typeSpec.Name.Name) {
				violations = append(violations, violationAt(file.Fset, file.AbsPath, typeSpec.Pos(), "appcmd must not declare DTO/TDO types; place transport contracts in handler"))
			}
		}
	}
	ast.Inspect(file.AST, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok || !humaNames[ident.Name] {
			return true
		}
		if selector.Sel.Name == "Register" || selector.Sel.Name == "NewGroup" {
			violations = append(violations, violationAt(file.Fset, file.AbsPath, selector.Pos(), "appcmd must not register huma routes; place transport routes in handler"))
		}
		return true
	})
	return violations
}

func isAppcmdFile(file rulekit.GoFile) bool {
	for _, part := range strings.Split(file.RelPath, "/") {
		if part == "appcmd" {
			return true
		}
	}
	return false
}

func importAlias(spec *ast.ImportSpec, fallback string) string {
	if spec.Name == nil {
		return fallback
	}
	return spec.Name.Name
}

func transportContractType(name string) bool {
	upper := strings.ToUpper(name)
	return strings.HasSuffix(upper, "DTO") ||
		strings.HasSuffix(upper, "DTOS") ||
		strings.HasSuffix(upper, "TDO") ||
		strings.HasSuffix(upper, "TDOS")
}
