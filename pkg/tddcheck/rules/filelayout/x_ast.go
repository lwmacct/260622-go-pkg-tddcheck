package filelayout

import (
	"go/ast"
	"go/token"
	"go/types"
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

func firstParamIsContext(file rulekit.GoFile, funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) == 0 {
		return false
	}
	expr := funcDecl.Type.Params.List[0].Type
	if semanticType := file.TypeOf(expr); semanticType != nil {
		path, name, ok := namedType(semanticType)
		return ok && path == "context" && name == "Context"
	}
	return selectorPackage(expr) == "context"
}

func lastResultIsError(file rulekit.GoFile, funcDecl *ast.FuncDecl) bool {
	results := resultExprs(funcDecl)
	if len(results) == 0 {
		return false
	}
	last := results[len(results)-1]
	if semanticType := file.TypeOf(last); semanticType != nil {
		return types.Identical(types.Unalias(semanticType), types.Universe.Lookup("error").Type())
	}
	if ident, ok := last.(*ast.Ident); ok {
		return ident.Name == "error"
	}
	return false
}

func namedType(value types.Type) (string, string, bool) {
	value = types.Unalias(value)
	if pointer, ok := value.(*types.Pointer); ok {
		value = types.Unalias(pointer.Elem())
	}
	named, ok := value.(*types.Named)
	if !ok || named.Obj() == nil {
		return "", "", false
	}
	path := ""
	if named.Obj().Pkg() != nil {
		path = named.Obj().Pkg().Path()
	}
	return path, named.Obj().Name(), true
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

func exprIsIdent(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func startsWithUpper(value string) bool {
	first, _ := utf8.DecodeRuneInString(value)
	return first != utf8.RuneError && unicode.IsUpper(first)
}

func upperCamelName(value string) string {
	return upperCamelNameWithInitialisms(value, nil)
}

func upperCamelNameWithInitialisms(value string, initialisms map[string]string) string {
	parts := strings.Split(value, "_")
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if initialism, ok := configuredInitialism(part, initialisms); ok {
			builder.WriteString(initialism)
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	return builder.String()
}

func configuredInitialism(value string, initialisms map[string]string) (string, bool) {
	if initialism, ok := initialisms[strings.ToLower(value)]; ok {
		return initialism, true
	}
	return commonInitialism(value)
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
	case "json":
		return "JSON", true
	case "llm":
		return "LLM", true
	case "oauth":
		return "OAuth", true
	case "rbac":
		return "RBAC", true
	case "sql":
		return "SQL", true
	case "ssh":
		return "SSH", true
	case "tls":
		return "TLS", true
	case "url":
		return "URL", true
	case "uuid":
		return "UUID", true
	case "ws":
		return "WS", true
	default:
		return "", false
	}
}

func lowerCamelName(value string) string {
	return lowerCamelNameWithInitialisms(value, nil)
}

func lowerCamelNameWithInitialisms(value string, initialisms map[string]string) string {
	upper := upperCamelNameWithInitialisms(value, initialisms)
	if upper == "" {
		return ""
	}
	parts := strings.Split(value, "_")
	if len(parts) > 0 {
		if initialism, ok := configuredInitialism(parts[0], initialisms); ok {
			return strings.ToLower(initialism) + upper[len(initialism):]
		}
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
	return snakeNameWithInitialisms(value, nil)
}

func snakeNameWithInitialisms(value string, initialisms map[string]string) string {
	if tokens := camelTokensWithInitialisms(value, initialisms); len(tokens) > 0 {
		return strings.Join(tokens, "_")
	}
	return ""
}

func camelTokensWithInitialisms(value string, initialisms map[string]string) []string {
	for _, spelling := range initialismSpellings(initialisms) {
		value = strings.ReplaceAll(value, spelling, "_"+strings.ToLower(spelling)+"_")
	}
	return camelTokens(value)
}

func initialismSpellings(initialisms map[string]string) []string {
	values := make(map[string]bool)
	for _, value := range initialisms {
		values[value] = true
	}
	for _, value := range []string{"API", "HTTP", "ID", "IP", "JSON", "LLM", "OAuth", "RBAC", "SQL", "SSH", "TLS", "URL", "UUID", "WS"} {
		values[value] = true
	}
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	slices.SortFunc(result, func(a, b string) int {
		if len(a) != len(b) {
			return len(b) - len(a)
		}
		return strings.Compare(a, b)
	})
	return result
}

func legacySnakeName(value string) string {
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
	position := fileSet.PositionFor(pos, true)
	return Violation{File: displayFilename(filename), Line: position.Line, Column: position.Column, Message: message}
}

func displayFilename(filename string) string {
	return rulekit.DisplayFilename(filename)
}

func oneOf(value string, values ...string) bool {
	return slices.Contains(values, value)
}
