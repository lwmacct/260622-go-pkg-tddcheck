package filelayout

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strings"
	"unicode"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func inferredSubjectViolations(name fileName, file rulekit.GoFile, _ map[string]string) []Violation {
	if name.Namespace != "" {
		return nil
	}
	for _, identifier := range fileIdentifiers(file.AST) {
		if expectedSubject, ok := inferredSnakeSubject(name.Subject, identifier.Name); ok && expectedSubject != name.Subject {
			fileName := displayFilename(file.AbsPath)
			position := file.Fset.PositionFor(identifier.Pos, true)
			return []Violation{{
				File:    fileName,
				Line:    position.Line,
				Column:  position.Column,
				Code:    RuleID + "/subject-inference",
				Message: fmt.Sprintf("subject %q must use snake_case name %q inferred from %s", name.Subject, expectedSubject, identifier.Name),
				Fix:     renameSubjectFix(fileName, name, expectedSubject),
			}}
		}
	}
	return nil
}

var subjectPrefixKinds = map[string]bool{
	"commands": true,
	"dto":      true,
	"mapper":   true,
	"provider": true,
	"support":  true,
	"types":    true,
}

func subjectPrefixViolations(profile rulekit.Profile, name fileName, file rulekit.GoFile) []Violation {
	if !subjectPrefixKinds[name.Kind] {
		return nil
	}
	mode := profile.SubjectOwnershipMode(file.Layer, name.Kind)
	if mode == "off" {
		return nil
	}
	expected := upperCamelNameWithInitialisms(name.Subject, profile.Initialisms)
	var violations []Violation
	for _, declaration := range file.AST.Decls {
		switch typed := declaration.(type) {
		case *ast.GenDecl:
			// Repository support/types policies already enforce exported type
			// ownership and provide more specific model diagnostics.
			if typed.Tok == token.TYPE && file.Layer != "repository" {
				for _, spec := range typed.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !ast.IsExported(typeSpec.Name.Name) || camelTokenPrefix(typeSpec.Name.Name, expected) {
						continue
					}
					violations = append(violations, subjectPrefixViolation(profile, name, file, typeSpec.Name.Pos(), typeSpec.Name.End(), typeSpec.Name.Name, expected, mode))
				}
			}
			if (typed.Tok == token.CONST || typed.Tok == token.VAR) && (name.Kind == "support" || name.Kind == "types") {
				for _, spec := range typed.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, identifier := range valueSpec.Names {
						if ast.IsExported(identifier.Name) && !subjectValuePrefix(identifier.Name, expected, typed.Tok) {
							want := expected
							if typed.Tok == token.VAR && strings.HasPrefix(identifier.Name, "Err") {
								want = "Err" + expected
							}
							violations = append(violations, subjectPrefixViolation(profile, name, file, identifier.Pos(), identifier.End(), identifier.Name, want, mode))
						}
					}
				}
			}
		case *ast.FuncDecl:
			if name.Kind == "support" && typed.Recv == nil && ast.IsExported(typed.Name.Name) {
				if action, ok := supportAction(typed.Name.Name); ok && !camelTokenPrefix(strings.TrimPrefix(typed.Name.Name, action), expected) {
					violations = append(violations, subjectPrefixViolation(profile, name, file, typed.Name.Pos(), typed.Name.End(), typed.Name.Name, action+expected, mode))
				}
			}
			if name.Kind != "mapper" || typed.Recv != nil || !ast.IsExported(typed.Name.Name) || !strings.HasPrefix(typed.Name.Name, "To") {
				continue
			}
			if !camelTokenPrefix(strings.TrimPrefix(typed.Name.Name, "To"), expected) {
				violations = append(violations, subjectPrefixViolation(profile, name, file, typed.Name.Pos(), typed.Name.End(), typed.Name.Name, "To"+expected, mode))
			}
		}
	}
	return violations
}

func supportAction(identifier string) (string, bool) {
	for _, action := range []string{"Wrap", "Is", "As"} {
		if strings.HasPrefix(identifier, action) {
			return action, true
		}
	}
	return "", false
}

func subjectValuePrefix(identifier string, expected string, declaration token.Token) bool {
	if camelTokenPrefix(identifier, expected) {
		return true
	}
	return declaration == token.VAR && strings.HasPrefix(identifier, "Err") && camelTokenPrefix(strings.TrimPrefix(identifier, "Err"), expected)
}

func subjectPrefixViolation(profile rulekit.Profile, name fileName, file rulekit.GoFile, position token.Pos, end token.Pos, identifier string, expected string, mode string) Violation {
	violation := violationAt(
		file.Fset,
		file.AbsPath,
		position,
		fmt.Sprintf("subject-specific declaration %s must start with %s; put shared declarations in x.shared.*", identifier, expected),
	)
	violation.Code = RuleID + "/subject-ownership"
	if mode == "warning" {
		violation.Severity = rulekit.SeverityWarning
	}
	if replacement, ok := replaceSubjectPrefix(name.Subject, identifier, expected, profile.Initialisms); ok {
		start := file.Fset.PositionFor(position, true)
		finish := file.Fset.PositionFor(end, true)
		violation.Fix = &rulekit.SuggestedFix{
			Message: "rename declaration to match its owning subject",
			Edits: []rulekit.TextEdit{{
				File: violation.File,
				Range: rulekit.Range{
					Start: rulekit.Position{File: violation.File, Line: start.Line, Column: start.Column},
					End:   rulekit.Position{File: violation.File, Line: finish.Line, Column: finish.Column},
				},
				NewText: replacement,
			}},
		}
	}
	return violation
}

func replaceSubjectPrefix(subject string, identifier string, expected string, initialisms map[string]string) (string, bool) {
	if camelTokenPrefix(identifier, expected) {
		return "", false
	}
	parts := strings.Split(subject, "_")
	if len(parts) == 0 || parts[0] == "" {
		return "", false
	}
	old := upperCamelNameWithInitialisms(parts[0], initialisms)
	if !camelTokenPrefix(identifier, old) {
		return "", false
	}
	return expected + identifier[len(old):], true
}

func renameSubjectFix(file string, name fileName, subject string) *rulekit.SuggestedFix {
	return &rulekit.SuggestedFix{
		Message: "rename the file to match the inferred subject",
		Rename:  &rulekit.RenameFix{From: file, To: path.Join(path.Dir(file), subject+"."+name.Kind+".go")},
	}
}

type fileIdentifier struct {
	Name string
	Pos  token.Pos
	End  token.Pos
}

func fileIdentifiers(parsedFile *ast.File) []fileIdentifier {
	var identifiers []fileIdentifier
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range typed.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					identifiers = append(identifiers, fileIdentifier{Name: typeSpec.Name.Name, Pos: typeSpec.Name.Pos(), End: typeSpec.Name.End()})
				}
			}
		case *ast.FuncDecl:
			identifiers = append(identifiers, fileIdentifier{Name: typed.Name.Name, Pos: typed.Name.Pos(), End: typed.Name.End()})
			receiverName := receiverTypeName(typed.Recv)
			if receiverName != "" {
				identifiers = append(identifiers, fileIdentifier{Name: receiverName})
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
