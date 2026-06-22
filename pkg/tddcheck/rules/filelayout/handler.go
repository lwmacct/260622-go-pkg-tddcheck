package filelayout

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func handlerViolations(fileSet *token.FileSet, filename string, name fileName, parsedFile *ast.File) []Violation {
	var violations []Violation
	expectedHandlerName := lowerCamelName(name.scope) + "Handler"
	hasHandlerType := false
	hasRegister := false

	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			if typed.Tok != token.TYPE {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "handler files must only declare the handler struct"))
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if typeSpec.Name.Name != expectedHandlerName {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), fmt.Sprintf("handler files must only declare %s", expectedHandlerName)))
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); !ok {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), expectedHandlerName+" must be a struct"))
					continue
				}
				hasHandlerType = true
			}
		case *ast.FuncDecl:
			if typed.Recv == nil {
				if !strings.HasPrefix(typed.Name.Name, "Register") {
					violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "handler files must only declare Register* package-level functions"))
					continue
				}
				hasRegister = true
				continue
			}
			if receiverTypeName(typed.Recv) != expectedHandlerName {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "handler receiver methods must use "+expectedHandlerName))
			}
		}
	}
	if !hasHandlerType {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: "handler files must declare " + expectedHandlerName,
		})
	}
	if !hasRegister {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: "handler files must declare Register* package-level functions",
		})
	}
	return violations
}
