package filelayout

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func storeViolations(file rulekit.GoFile, name fileName, actions []string, initialisms map[string]string) []Violation {
	var violations []Violation
	expectedSubjectPrefix := upperCamelNameWithInitialisms(name.Qualifier(), initialisms)
	for _, decl := range file.AST.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			violations = append(violations, violationAt(file.Fset, file.AbsPath, typed.Pos(), "store files must only declare Store receiver methods"))
		case *ast.FuncDecl:
			if receiverTypeName(typed.Recv) != "Store" {
				violations = append(violations, violationAt(file.Fset, file.AbsPath, typed.Pos(), "store files must only declare Store receiver methods"))
				continue
			}
			if message, style := storeMethodNameViolation(typed, expectedSubjectPrefix, actions); message != "" {
				violation := violationAt(file.Fset, file.AbsPath, typed.Pos(), message)
				if style {
					violation.Code = RuleID + "/store-style"
					violation.Severity = rulekit.SeverityWarning
				}
				violations = append(violations, violation)
			}
			if !firstParamIsContext(file, typed) {
				violations = append(violations, violationAt(file.Fset, file.AbsPath, typed.Pos(), "store methods must accept context.Context as the first parameter"))
			}
			if !lastResultIsError(file, typed) {
				violations = append(violations, violationAt(file.Fset, file.AbsPath, typed.Pos(), "store methods must return error as the last result"))
			}
			if typed.Name.IsExported() {
				if message := storeMethodResultViolation(typed, actions); message != "" {
					violation := violationAt(file.Fset, file.AbsPath, typed.Pos(), message)
					violation.Code = RuleID + "/store-style"
					violation.Severity = rulekit.SeverityWarning
					violations = append(violations, violation)
				}
			}
		}
	}
	return violations
}

func storeMethodNameViolation(funcDecl *ast.FuncDecl, expectedSubjectPrefix string, actions []string) (string, bool) {
	name := funcDecl.Name.Name
	if !funcDecl.Name.IsExported() {
		if !lowerCamelIdentifier(name) {
			return "private store helper methods must use lowerCamel names", false
		}
		if storeMethodNameExposesQuery(name) {
			return "store method names must not expose query implementation details", false
		}
		return "", false
	}
	if len(actions) == 0 {
		return "", false
	}
	action, subject, ok := splitStoreMethodName(name, actions)
	if !ok {
		if storeMethodNameExposesQuery(name) {
			return "store method names must not expose query implementation details", false
		}
		return "store method action is not configured; add the domain action to StoreMethodActions when intentional", true
	}
	if subject == "" {
		return "store method names must include a subject after the action", false
	}
	if !startsWithUpper(subject) {
		return "store method names must use Action+UpperCamelSubject", false
	}
	if message := storeMethodSubjectViolation(action, subject, expectedSubjectPrefix); message != "" {
		return message, false
	}
	if storeMethodNameExposesQuery(subject) {
		return "store method names must not expose query implementation details", false
	}
	return "", false
}

func storeMethodSubjectViolation(action string, subject string, expected string) string {
	if strings.HasPrefix(subject, expected) {
		rest := strings.TrimPrefix(subject, expected)
		if rest == "" || startsWithUpper(rest) {
			return ""
		}
	}
	plural := pluralSubject(expected)
	if (action == "List" || action == "Count") && strings.HasPrefix(subject, plural) {
		rest := strings.TrimPrefix(subject, plural)
		if rest == "" || explicitListQualifier(rest) {
			return ""
		}
		return action + " store method qualifiers after plural subjects must start with By, For, With, or Without"
	}
	return "exported store method subjects must start with " + expected + " as an exact resource segment"
}

func explicitListQualifier(rest string) bool {
	return strings.HasPrefix(rest, "By") ||
		strings.HasPrefix(rest, "For") ||
		strings.HasPrefix(rest, "With") ||
		strings.HasPrefix(rest, "Without")
}

func pluralSubject(subject string) string {
	if strings.HasSuffix(subject, "y") && consonantBeforeTrailingY(subject) {
		return strings.TrimSuffix(subject, "y") + "ies"
	}
	for _, suffix := range []string{"s", "x", "z", "ch", "sh"} {
		if strings.HasSuffix(subject, suffix) {
			return subject + "es"
		}
	}
	return subject + "s"
}

func consonantBeforeTrailingY(subject string) bool {
	if len(subject) < 2 {
		return false
	}
	switch subject[len(subject)-2] {
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return false
	default:
		return true
	}
}

func storeMethodNameExposesQuery(name string) bool {
	for _, forbidden := range []string{"Where", "where", "Query", "query", "SQL", "sql"} {
		if strings.Contains(name, forbidden) {
			return true
		}
	}
	return false
}

func splitStoreMethodName(name string, actions []string) (string, string, bool) {
	ordered := append([]string(nil), actions...)
	slices.SortFunc(ordered, func(a, b string) int {
		return len(b) - len(a)
	})
	for _, action := range ordered {
		if strings.HasPrefix(name, action) {
			return action, strings.TrimPrefix(name, action), true
		}
	}
	return "", "", false
}

func storeMethodResultViolation(funcDecl *ast.FuncDecl, actions []string) string {
	if len(actions) == 0 {
		return ""
	}
	action, _, ok := splitStoreMethodName(funcDecl.Name.Name, actions)
	if !ok {
		return ""
	}
	results := resultExprs(funcDecl)
	if len(results) == 0 || !exprIsIdent(results[len(results)-1], "error") {
		return ""
	}
	values := results[:len(results)-1]
	switch action {
	case "List":
		if len(values) != 1 || !exprIsSlice(values[0]) {
			return "List store methods must return a slice and error"
		}
	case "Fetch":
		if len(values) != 1 {
			return "Fetch store methods must return one value and error"
		}
	case "Count":
		if len(values) != 1 || !integerResult(values[0]) {
			return "Count store methods must return an integer and error"
		}
	case "Exists":
		if len(values) != 1 || !exprIsIdent(values[0], "bool") {
			return "Exists store methods must return bool and error"
		}
	case "Create", "Update", "Upsert":
		if len(values) > 2 {
			return action + " store methods must return at most two values and error"
		}
	case "Delete", "Remove":
		if len(values) != 0 && (len(values) != 1 || !exprIsIdent(values[0], "bool")) {
			return action + " store methods must return optional bool and error"
		}
	case "Add", "Replace":
		if len(values) != 0 {
			return action + " store methods must only return error"
		}
	}
	return ""
}

func integerResult(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	switch ident.Name {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return true
	default:
		return false
	}
}
