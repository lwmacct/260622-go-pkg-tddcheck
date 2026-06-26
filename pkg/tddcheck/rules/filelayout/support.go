package filelayout

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func supportViolations(fileSet *token.FileSet, filename string, layer string, name fileName, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			violations = append(violations, supportGenDeclViolations(fileSet, filename, layer, name, typed)...)
		case *ast.FuncDecl:
			violations = append(violations, supportFuncViolations(fileSet, filename, layer, typed)...)
		}
	}
	return violations
}

func supportGenDeclViolations(fileSet *token.FileSet, filename string, layer string, name fileName, decl *ast.GenDecl) []Violation {
	var violations []Violation
	switch decl.Tok {
	case token.IMPORT:
		for _, importSpec := range decl.Specs {
			spec, ok := importSpec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			path := importPath(spec)
			if forbiddenSupportImport(layer, path) {
				violations = append(violations, violationAt(fileSet, filename, spec.Pos(), "support files must not import "+path))
			}
		}
	case token.TYPE:
		for _, spec := range decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if layer == "service" {
				if forbiddenServiceModelType(typeSpec.Name.Name) {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "service support type "+typeSpec.Name.Name+" must not use transport or command suffixes"))
				}
				if hasStructTag(typeSpec, "bun") || hasStructTag(typeSpec, "json") || hasStructTag(typeSpec, "query") || hasStructTag(typeSpec, "path") {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "service support types must not declare transport or persistence tags"))
				}
			}
			if layer == "repository" && forbiddenRepositorySupportTypeName(name, typeSpec.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), fmt.Sprintf("repository support type %s must start with %s", typeSpec.Name.Name, upperCamelName(name.scope))))
			}
			if layer == "repository" && forbiddenRepositorySupportModelType(typeSpec) {
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "repository support files must not declare schema models; place Model structs and ORM tags in .schema.go"))
			}
		}
	case token.CONST:
		return nil
	case token.VAR:
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range valueSpec.Names {
				if !strings.HasPrefix(name.Name, "Err") {
					violations = append(violations, violationAt(fileSet, filename, name.Pos(), "support vars must start with Err"))
				}
			}
		}
	default:
		violations = append(violations, violationAt(fileSet, filename, decl.Pos(), "support files must only declare types, consts, Err* vars, and support functions"))
	}
	return violations
}

func supportFuncViolations(fileSet *token.FileSet, filename string, layer string, decl *ast.FuncDecl) []Violation {
	name := decl.Name.Name
	if decl.Recv != nil {
		if layer == "repository" {
			return nil
		}
		if name == "Error" || name == "Unwrap" {
			return nil
		}
		return []Violation{violationAt(fileSet, filename, decl.Pos(), "service support files must not declare receiver methods")}
	}
	if supportFunctionName(name) {
		return nil
	}
	return []Violation{violationAt(fileSet, filename, decl.Pos(), "support functions must start with util, validate, normalize, Wrap, Is, or As")}
}

func supportFunctionName(name string) bool {
	return strings.HasPrefix(name, "util") ||
		strings.HasPrefix(name, "validate") ||
		strings.HasPrefix(name, "normalize") ||
		strings.HasPrefix(name, "Wrap") ||
		strings.HasPrefix(name, "Is") ||
		strings.HasPrefix(name, "As")
}

func forbiddenRepositorySupportTypeName(name fileName, typeName string) bool {
	if strings.HasPrefix(name.scope, architectureScopePrefix) || !startsWithUpper(typeName) {
		return false
	}
	return !camelTokenPrefix(typeName, upperCamelName(name.scope))
}

func camelTokenPrefix(value string, prefix string) bool {
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	if len(value) == len(prefix) {
		return true
	}
	return upperRune(rune(value[len(prefix)]))
}

func forbiddenSupportImport(layer string, importPath string) bool {
	switch layer {
	case "service":
		return forbiddenServiceModelImport(importPath)
	case "repository":
		return forbiddenRepositoryModelImport(importPath)
	default:
		return false
	}
}
