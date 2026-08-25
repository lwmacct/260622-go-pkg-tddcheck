package filelayout

import (
	"go/ast"
	"go/token"
	"path"
	"sort"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

var subjectDeclarationKinds = map[string]struct{}{
	"dto":     {},
	"support": {},
	"types":   {},
}

func subjectDeclarationOwnershipViolations(snapshot *rulekit.Snapshot) []Violation {
	subjectsByLayer := make(map[string][]string)
	seenByLayer := make(map[string]map[string]struct{})
	for _, file := range snapshot.Files {
		if file.IsTest || file.Layer == "" || !file.IdentityOK {
			continue
		}
		name := file.Identity
		if name.Namespace != "" || name.Subject == "" || name.Kind == "free" {
			continue
		}
		if seenByLayer[file.Layer] == nil {
			seenByLayer[file.Layer] = make(map[string]struct{})
		}
		if _, exists := seenByLayer[file.Layer][name.Subject]; exists {
			continue
		}
		seenByLayer[file.Layer][name.Subject] = struct{}{}
		subjectsByLayer[file.Layer] = append(subjectsByLayer[file.Layer], name.Subject)
	}
	for layer := range subjectsByLayer {
		sort.Slice(subjectsByLayer[layer], func(left, right int) bool {
			leftTokens := strings.Count(subjectsByLayer[layer][left], "_")
			rightTokens := strings.Count(subjectsByLayer[layer][right], "_")
			if leftTokens != rightTokens {
				return leftTokens > rightTokens
			}
			return subjectsByLayer[layer][left] < subjectsByLayer[layer][right]
		})
	}

	var violations []Violation
	for _, file := range snapshot.Files {
		if file.IsTest || file.Layer == "" || !file.IdentityOK {
			continue
		}
		name := file.Identity
		if name.Namespace != "" {
			continue
		}
		if _, ok := subjectDeclarationKinds[name.Kind]; !ok {
			continue
		}
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
				foreignSubject := foreignSubjectPrefix(typeSpec.Name.Name, name.Subject, subjectsByLayer[file.Layer])
				if foreignSubject == "" {
					continue
				}
				oldFile := rulekit.DisplayFilename(file.AbsPath)
				newFile := path.Join(path.Dir(oldFile), foreignSubject+"."+name.Kind+".go")
				violations = append(violations, Violation{
					File:    oldFile,
					Line:    1,
					Code:    RuleID + "/cross-subject-declaration",
					Message: "type " + typeSpec.Name.Name + " belongs to subject " + quoteSubject(foreignSubject) + ", but is declared in subject " + quoteSubject(name.Subject) + "; move it to " + newFile,
				})
			}
		}
	}
	return violations
}

func foreignSubjectPrefix(typeName string, currentSubject string, subjects []string) string {
	tokens := camelTokens(typeName)
	currentToken := strings.Split(currentSubject, "_")[0]
	for index, token := range tokens {
		if index > 0 && token == currentToken {
			return ""
		}
	}
	best := ""
	for _, subject := range subjects {
		subjectTokens := strings.Split(subject, "_")
		if !hasTokenPrefix(tokens, subjectTokens) {
			continue
		}
		if best != "" && len(subjectTokens) <= len(strings.Split(best, "_")) {
			continue
		}
		best = subject
	}
	if best == currentSubject {
		return ""
	}
	return best
}

func hasTokenPrefix(tokens []string, prefix []string) bool {
	if len(prefix) > len(tokens) {
		return false
	}
	for index, token := range prefix {
		if tokens[index] != token {
			return false
		}
	}
	return true
}

func quoteSubject(subject string) string {
	return `"` + subject + `"`
}
