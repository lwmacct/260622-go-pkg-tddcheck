package filelayout

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func appcmdTransportViolations(root string, files []string, config rulekit.Config) ([]Violation, error) {
	if !slices.Contains(config.DependencyLayerDirs, "appcmd") {
		return nil, nil
	}
	var violations []Violation
	for _, file := range files {
		if !isAppcmdFile(root, file) {
			continue
		}
		fileViolations, err := appcmdTransportViolationsInFile(file)
		if err != nil {
			return nil, err
		}
		violations = append(violations, fileViolations...)
	}
	return violations, nil
}

func appcmdTransportViolationsInFile(filename string) ([]Violation, error) {
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, filename, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var violations []Violation
	humaNames := map[string]bool{}
	for _, decl := range parsedFile.Decls {
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
			violations = append(violations, violationAt(fileSet, filename, importSpec.Pos(), "appcmd must not import huma; register transport routes in handler"))
			humaNames[importAlias(importSpec, "huma")] = true
		}
	}
	for _, decl := range parsedFile.Decls {
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
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "appcmd must not declare DTO/TDO types; place transport contracts in handler"))
			}
		}
	}
	ast.Inspect(parsedFile, func(node ast.Node) bool {
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
			violations = append(violations, violationAt(fileSet, filename, selector.Pos(), "appcmd must not register huma routes; place transport routes in handler"))
		}
		return true
	})
	return violations, nil
}

func isAppcmdFile(root string, file string) bool {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
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
