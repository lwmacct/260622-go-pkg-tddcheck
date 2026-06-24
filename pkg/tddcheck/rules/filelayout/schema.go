package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func schemaViolations(fileSet *token.FileSet, filename string, name fileName, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			violations = append(violations, schemaGenDeclViolations(fileSet, filename, name, typed)...)
		case *ast.FuncDecl:
			violations = append(violations, schemaFuncViolations(fileSet, filename, typed)...)
		}
	}
	return violations
}

func schemaGenDeclViolations(fileSet *token.FileSet, filename string, name fileName, decl *ast.GenDecl) []Violation {
	var violations []Violation
	switch decl.Tok {
	case token.IMPORT:
		return nil
	case token.TYPE:
		for _, spec := range decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := typeSpec.Type.(*ast.StructType); !ok {
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "schema files must only declare model structs"))
				continue
			}
			expected := upperCamelName(name.scope)
			if !camelTokenPrefix(typeSpec.Name.Name, expected) || !strings.HasSuffix(typeSpec.Name.Name, "Model") {
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "schema model type "+typeSpec.Name.Name+" must start with "+expected+" and end with Model"))
			}
		}
	default:
		violations = append(violations, violationAt(fileSet, filename, decl.Pos(), "schema files must only declare model structs and schema lifecycle functions"))
	}
	return violations
}

func schemaFuncViolations(fileSet *token.FileSet, filename string, decl *ast.FuncDecl) []Violation {
	if decl.Recv != nil {
		receiver := receiverTypeName(decl.Recv)
		if !strings.HasSuffix(receiver, "Model") {
			return []Violation{violationAt(fileSet, filename, decl.Pos(), "schema receiver methods must use *Model receivers")}
		}
		return nil
	}
	if !schemaFunctionName(decl.Name.Name) {
		return []Violation{violationAt(fileSet, filename, decl.Pos(), "schema package-level functions must be schema lifecycle functions")}
	}
	return nil
}

func schemaFunctionName(name string) bool {
	if oneOf(name, "Schema", "CreateIndexes", "CreateSchema", "EnsureSchema") {
		return true
	}
	return strings.HasSuffix(name, "Schema")
}

func forbiddenRepositorySupportModelType(typeSpec *ast.TypeSpec) bool {
	return strings.HasSuffix(typeSpec.Name.Name, "Model") ||
		hasStructTag(typeSpec, "bun") ||
		hasStructTag(typeSpec, "gorm")
}
