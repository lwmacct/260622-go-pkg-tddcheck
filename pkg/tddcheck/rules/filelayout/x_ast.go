package filelayout

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func typeOnlyViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File, message string) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && (genDecl.Tok == token.IMPORT || genDecl.Tok == token.TYPE) {
			continue
		}
		violations = append(violations, violationAt(fileSet, filename, decl.Pos(), message))
	}
	return violations
}

func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch typed := recv.List[0].Type.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		if ident, ok := typed.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func upperCamelName(value string) string {
	parts := strings.Split(value, "_")
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	return builder.String()
}

func lowerCamelName(value string) string {
	upper := upperCamelName(value)
	if upper == "" {
		return ""
	}
	return strings.ToLower(upper[:1]) + upper[1:]
}

func lowerCamelIdentifier(value string) bool {
	if value == "" {
		return false
	}
	first := value[0]
	return 'a' <= first && first <= 'z'
}

func snakeName(value string) string {
	var builder strings.Builder
	for index, char := range value {
		if index > 0 && 'A' <= char && char <= 'Z' {
			builder.WriteByte('_')
		}
		builder.WriteRune(char)
	}
	return strings.ToLower(builder.String())
}

func violationAt(fileSet *token.FileSet, filename string, pos token.Pos, message string) Violation {
	return Violation{File: displayFilename(filename), Line: fileSet.Position(pos).Line, Message: message}
}

func displayFilename(filename string) string {
	return rulekit.DisplayFilename(filename)
}

func oneOf(value string, values ...string) bool {
	return slices.Contains(values, value)
}
