package filelayout

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func supportViolations(fileSet *token.FileSet, filename string, layer string, name fileName, parsedFile *ast.File, maxDeclarationLines int) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			violations = append(violations, declarationGenDeclViolations(fileSet, filename, layer, name, "support", typed)...)
		case *ast.FuncDecl:
			violations = append(violations, supportFuncViolations(fileSet, filename, layer, typed)...)
		}
	}
	violations = append(violations, supportDeclarationLineViolations(fileSet, filename, name, parsedFile, maxDeclarationLines)...)
	return violations
}

func supportDeclarationLineViolations(fileSet *token.FileSet, filename string, name fileName, parsedFile *ast.File, maxDeclarationLines int) []Violation {
	lines, first := declarationLines(fileSet, parsedFile)
	if maxDeclarationLines <= 0 || lines <= maxDeclarationLines {
		return nil
	}
	return []Violation{violationAt(fileSet, filename, first, fmt.Sprintf("support declarations occupy %d lines (maximum %d); move types, consts, and Err* vars to %s", lines, maxDeclarationLines, typesFilename(name)))}
}

func declarationGenDeclViolations(fileSet *token.FileSet, filename string, layer string, name fileName, kind string, decl *ast.GenDecl) []Violation {
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
				violations = append(violations, violationAt(fileSet, filename, spec.Pos(), kind+" files must not import "+path))
			}
		}
	case token.TYPE:
		for _, spec := range decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if layer == "service" {
				if hasStructTag(typeSpec, "bun") {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "service "+kind+" types must not declare persistence tags"))
				}
			}
			if layer == "repository" && forbiddenRepositorySupportTypeName(name, typeSpec.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), fmt.Sprintf("repository %s type %s must start with %s", kind, typeSpec.Name.Name, upperCamelName(name.Subject))))
			}
			if layer == "repository" && forbiddenRepositorySupportModelType(typeSpec) {
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "repository "+kind+" files must not declare schema models; place Model structs and ORM tags in .schema.go"))
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
					violations = append(violations, violationAt(fileSet, filename, name.Pos(), kind+" vars must start with Err"))
				}
			}
		}
	default:
		violations = append(violations, violationAt(fileSet, filename, decl.Pos(), kind+" files must only declare types, consts, Err* vars, and support functions"))
	}
	return violations
}

func declarationLines(fileSet *token.FileSet, parsedFile *ast.File) (int, token.Pos) {
	var lines int
	var first token.Pos
	for _, decl := range parsedFile.Decls {
		typed, ok := decl.(*ast.GenDecl)
		if !ok || (typed.Tok != token.TYPE && typed.Tok != token.CONST && typed.Tok != token.VAR) {
			continue
		}
		if first == token.NoPos {
			first = typed.Pos()
		}
		start := fileSet.PositionFor(typed.Pos(), true).Line
		end := fileSet.PositionFor(typed.End(), true).Line
		if end >= start {
			lines += end - start + 1
		}
	}
	return lines, first
}

func typesFilename(name fileName) string {
	if name.Namespace != "" {
		return "x." + name.Namespace + ".types.go"
	}
	return name.Subject + ".types.go"
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
	if name.Namespace != "" || !startsWithUpper(typeName) {
		return false
	}
	return !camelTokenPrefix(typeName, upperCamelName(name.Subject))
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
