package filelayout

import (
	"fmt"
	"go/ast"
	"strings"
	"unicode"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func inferredScopeViolations(name fileName, file rulekit.GoFile) []Violation {
	if strings.HasPrefix(name.scope, architectureScopePrefix) {
		return nil
	}
	for _, identifier := range fileIdentifiers(file.AST) {
		if expectedScope, ok := inferredSnakeScope(name.scope, identifier); ok && expectedScope != name.scope {
			return []Violation{{
				File:    rulekit.DisplayFilename(file.AbsPath),
				Line:    1,
				Message: fmt.Sprintf("scope %q must use snake_case subject name %q inferred from %s", name.scope, expectedScope, identifier),
			}}
		}
	}
	return nil
}

func fileIdentifiers(parsedFile *ast.File) []string {
	var identifiers []string
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range typed.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					identifiers = append(identifiers, typeSpec.Name.Name)
				}
			}
		case *ast.FuncDecl:
			identifiers = append(identifiers, typed.Name.Name)
			receiverName := receiverTypeName(typed.Recv)
			if receiverName != "" {
				identifiers = append(identifiers, receiverName)
			}
		}
	}
	return identifiers
}

func inferredSnakeScope(scope string, identifier string) (string, bool) {
	tokens := camelTokens(identifier)
	for start := 0; start < len(tokens); start++ {
		var joined strings.Builder
		for end := start; end < len(tokens); end++ {
			joined.WriteString(tokens[end])
			if end-start < 1 {
				continue
			}
			if joined.String() == strings.ReplaceAll(scope, "_", "") {
				return strings.Join(tokens[start:end+1], "_"), true
			}
		}
	}
	return "", false
}

func camelTokens(value string) []string {
	var tokens []string
	var current []rune
	runes := []rune(value)
	for index, char := range runes {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			if len(current) > 0 {
				tokens = append(tokens, strings.ToLower(string(current)))
				current = nil
			}
			continue
		}
		if index > 0 && len(current) > 0 && unicode.IsUpper(char) {
			prev := runes[index-1]
			var next rune
			if index+1 < len(runes) {
				next = runes[index+1]
			}
			if unicode.IsLower(prev) || (next != 0 && unicode.IsLower(next)) {
				tokens = append(tokens, strings.ToLower(string(current)))
				current = nil
			}
		}
		current = append(current, char)
	}
	if len(current) > 0 {
		tokens = append(tokens, strings.ToLower(string(current)))
	}
	return tokens
}

func escapedKindScope(escapedScopeSuffixes []string, scope string) (string, bool) {
	for _, kind := range escapedScopeSuffixes {
		if strings.HasSuffix(scope, "_"+kind) {
			return kind, true
		}
	}
	return "", false
}
