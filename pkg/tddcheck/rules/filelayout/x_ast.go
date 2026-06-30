package filelayout

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

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

func importPath(spec *ast.ImportSpec) string {
	return strings.Trim(spec.Path.Value, `"`)
}

func selectorPackage(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.SelectorExpr:
		if ident, ok := typed.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.StarExpr:
		return selectorPackage(typed.X)
	case *ast.ArrayType:
		return selectorPackage(typed.Elt)
	case *ast.MapType:
		if value := selectorPackage(typed.Key); value != "" {
			return value
		}
		return selectorPackage(typed.Value)
	}
	return ""
}

func hasStructTag(typeSpec *ast.TypeSpec, tagName string) bool {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return false
	}
	for _, field := range structType.Fields.List {
		if field.Tag != nil && strings.Contains(field.Tag.Value, tagName+":") {
			return true
		}
		if embeddedType, ok := field.Type.(*ast.StructType); ok {
			nested := &ast.TypeSpec{Type: embeddedType}
			if hasStructTag(nested, tagName) {
				return true
			}
		}
	}
	return false
}

func firstParamIsContext(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) == 0 {
		return false
	}
	return selectorPackage(funcDecl.Type.Params.List[0].Type) == "context"
}

func lastResultIsError(funcDecl *ast.FuncDecl) bool {
	results := resultExprs(funcDecl)
	if len(results) == 0 {
		return false
	}
	last := results[len(results)-1]
	if ident, ok := last.(*ast.Ident); ok {
		return ident.Name == "error"
	}
	return false
}

func resultExprs(funcDecl *ast.FuncDecl) []ast.Expr {
	if funcDecl.Type.Results == nil {
		return nil
	}
	var results []ast.Expr
	for _, field := range funcDecl.Type.Results.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			results = append(results, field.Type)
		}
	}
	return results
}

func exprIsSlice(expr ast.Expr) bool {
	_, ok := expr.(*ast.ArrayType)
	return ok
}

func exprIsPointer(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

func exprIsIdent(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func startsWithUpper(value string) bool {
	first, _ := utf8.DecodeRuneInString(value)
	return first != utf8.RuneError && unicode.IsUpper(first)
}

func upperCamelName(value string) string {
	parts := strings.Split(value, "_")
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if initialism, ok := commonInitialism(part); ok {
			builder.WriteString(initialism)
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	return builder.String()
}

func commonInitialism(value string) (string, bool) {
	switch strings.ToLower(value) {
	case "api":
		return "API", true
	case "http":
		return "HTTP", true
	case "id":
		return "ID", true
	case "ip":
		return "IP", true
	case "llm":
		return "LLM", true
	case "ssh":
		return "SSH", true
	case "tls":
		return "TLS", true
	case "url":
		return "URL", true
	case "ws":
		return "WS", true
	default:
		return "", false
	}
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
	runes := []rune(value)
	for index, char := range runes {
		if index > 0 && upperRune(char) {
			prev := runes[index-1]
			var next rune
			if index+1 < len(runes) {
				next = runes[index+1]
			}
			if !upperRune(prev) || (next != 0 && !upperRune(next)) {
				builder.WriteByte('_')
			}
		}
		builder.WriteRune(char)
	}
	return strings.ToLower(builder.String())
}

func upperRune(value rune) bool {
	return 'A' <= value && value <= 'Z'
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
