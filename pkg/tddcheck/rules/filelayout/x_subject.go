package filelayout

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func inferredSubjectViolations(name fileName, file rulekit.GoFile) []Violation {
	if name.Namespace != "" {
		return nil
	}
	for _, identifier := range fileIdentifiers(file.AST) {
		if expectedSubject, ok := inferredSnakeSubject(name.Subject, identifier); ok && expectedSubject != name.Subject {
			return []Violation{{
				File:    rulekit.DisplayFilename(file.AbsPath),
				Line:    1,
				Message: fmt.Sprintf("subject %q must use snake_case name %q inferred from %s", name.Subject, expectedSubject, identifier),
			}}
		}
	}
	return nil
}

func subjectOrderViolations(name fileName, file rulekit.GoFile) []Violation {
	if name.Namespace != "" || (name.Kind != "support" && name.Kind != "types" && name.Kind != "dto") {
		return nil
	}
	subjectToken := strings.Split(name.Subject, "_")[0]
	if subjectToken == "" {
		return nil
	}
	var violations []Violation
	for _, decl := range file.AST.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !ast.IsExported(typeSpec.Name.Name) {
				continue
			}
			tokens := camelTokens(typeSpec.Name.Name)
			if tokenIndex(tokens, subjectToken) <= 0 {
				continue
			}
			violations = append(violations, violationAt(
				file.Fset,
				file.AbsPath,
				typeSpec.Pos(),
				fmt.Sprintf("type %s contains file subject token %s after a qualifier; the subject token must come first", typeSpec.Name.Name, upperCamelName(subjectToken)),
			))
		}
	}
	return violations
}

func tokenIndex(tokens []string, target string) int {
	for index, token := range tokens {
		if token == target {
			return index
		}
	}
	return -1
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

func inferredSnakeSubject(subject string, identifier string) (string, bool) {
	tokens := camelTokens(identifier)
	for start := 0; start < len(tokens); start++ {
		var joined strings.Builder
		for end := start; end < len(tokens); end++ {
			joined.WriteString(tokens[end])
			if end-start < 1 {
				continue
			}
			if joined.String() == strings.ReplaceAll(subject, "_", "") {
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

func escapedKindSubject(escapedSubjectSuffixes []string, subject string) (string, bool) {
	for _, kind := range escapedSubjectSuffixes {
		if strings.HasSuffix(subject, "_"+kind) {
			return kind, true
		}
	}
	return "", false
}
