package filelayout

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func serviceViolations(fileSet *token.FileSet, filename string, name fileName, parsedFile *ast.File) []Violation {
	var violations []Violation
	serviceName, serviceNameOK := serviceStructName(parsedFile)
	expectedServiceName := serviceName
	if expectedServiceName == "" {
		expectedServiceName = upperCamelName(name.scope) + "Service"
	}
	if serviceNameOK {
		expectedScope := snakeName(strings.TrimSuffix(expectedServiceName, "Service"))
		if name.scope != expectedScope {
			violations = append(violations, Violation{
				File:    rulekit.DisplayFilename(filename),
				Line:    1,
				Message: fmt.Sprintf("service file for %s must use scope %q", expectedServiceName, expectedScope),
			})
		}
	}

	hasServiceType := false
	hasConstructor := false
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			if typed.Tok != token.TYPE {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "service files must only declare the service struct"))
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if typeSpec.Name.Name != expectedServiceName {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), fmt.Sprintf("service files must only declare %s", expectedServiceName)))
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); !ok {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), expectedServiceName+" must be a struct"))
					continue
				}
				hasServiceType = true
			}
		case *ast.FuncDecl:
			if typed.Recv == nil {
				if typed.Name.Name != "New"+expectedServiceName {
					violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "service files must only declare New"+expectedServiceName+" as a package-level function"))
					continue
				}
				hasConstructor = true
				continue
			}
			if receiverTypeName(typed.Recv) != expectedServiceName {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "service receiver methods must use "+expectedServiceName))
			}
		}
	}
	if !hasServiceType {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: "service files must declare " + expectedServiceName,
		})
	}
	if !hasConstructor {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: "service files must declare New" + expectedServiceName,
		})
	}
	return violations
}

func serviceStructName(parsedFile *ast.File) (string, bool) {
	var serviceName string
	for _, decl := range parsedFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !strings.HasSuffix(typeSpec.Name.Name, "Service") {
				continue
			}
			if _, ok := typeSpec.Type.(*ast.StructType); !ok {
				continue
			}
			if serviceName != "" {
				return serviceName, false
			}
			serviceName = typeSpec.Name.Name
		}
	}
	return serviceName, serviceName != ""
}
