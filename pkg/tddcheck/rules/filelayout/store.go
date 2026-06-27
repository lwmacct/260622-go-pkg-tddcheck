package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func storeViolations(fileSet *token.FileSet, filename string, name fileName, parsedFile *ast.File) []Violation {
	var violations []Violation
	expectedSubjectPrefix := upperCamelName(name.scope)
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "store files must only declare Store receiver methods"))
		case *ast.FuncDecl:
			if receiverTypeName(typed.Recv) != "Store" {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "store files must only declare Store receiver methods"))
				continue
			}
			if message := storeMethodNameViolation(typed, expectedSubjectPrefix); message != "" {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), message))
			}
			if !firstParamIsContext(typed) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "store methods must accept context.Context as the first parameter"))
			}
			if !lastResultIsError(typed) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "store methods must return error as the last result"))
			}
			if typed.Name.IsExported() {
				if message := storeMethodResultViolation(typed); message != "" {
					violations = append(violations, violationAt(fileSet, filename, typed.Pos(), message))
				}
			}
		}
	}
	return violations
}

func storeMethodNameViolation(funcDecl *ast.FuncDecl, expectedSubjectPrefix string) string {
	name := funcDecl.Name.Name
	if !funcDecl.Name.IsExported() {
		if !lowerCamelIdentifier(name) {
			return "private store helper methods must use lowerCamel names"
		}
		if storeMethodNameExposesQuery(name) {
			return "store method names must not expose query implementation details"
		}
		return ""
	}
	action, subject, ok := splitStoreMethodName(name)
	if !ok {
		return "store method names must start with List, Fetch, Count, Exists, Create, Update, Delete, Upsert, Add, Remove, or Replace"
	}
	if subject == "" {
		return "store method names must include a subject after the action"
	}
	if !startsWithUpper(subject) {
		return "store method names must use Action+UpperCamelSubject"
	}
	if message := storeMethodSubjectViolation(action, subject, expectedSubjectPrefix); message != "" {
		return message
	}
	if storeMethodNameExposesQuery(subject) {
		return "store method names must not expose query implementation details"
	}
	return ""
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

func splitStoreMethodName(name string) (string, string, bool) {
	for _, action := range []string{"Replace", "Remove", "Update", "Upsert", "Create", "Delete", "Exists", "Fetch", "Count", "List", "Add"} {
		if strings.HasPrefix(name, action) {
			return action, strings.TrimPrefix(name, action), true
		}
	}
	return "", "", false
}

func storeMethodResultViolation(funcDecl *ast.FuncDecl) string {
	action, _, ok := splitStoreMethodName(funcDecl.Name.Name)
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
		if len(values) != 1 || !exprIsPointer(values[0]) {
			return "Fetch store methods must return a pointer and error"
		}
	case "Count":
		if len(values) != 1 || !exprIsIdent(values[0], "int") {
			return "Count store methods must return int and error"
		}
	case "Exists":
		if len(values) != 1 || !exprIsIdent(values[0], "bool") {
			return "Exists store methods must return bool and error"
		}
	case "Create", "Update", "Upsert":
		if len(values) != 1 && len(values) != 2 {
			return action + " store methods must return a pointer, optional id string, and error"
		}
		if !exprIsPointer(values[0]) {
			return action + " store methods must return a pointer, optional id string, and error"
		}
		if len(values) == 2 && !exprIsIdent(values[1], "string") {
			return action + " store methods must return a pointer, optional id string, and error"
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
