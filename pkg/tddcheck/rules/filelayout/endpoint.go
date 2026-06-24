package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func architectureEndpointViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	hasEndpoint := false
	hasNewEndpoint := false
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			if typed.Tok != token.TYPE {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture endpoint files must only declare endpoint types and functions"))
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if typeSpec.Name.Name == "Endpoint" {
					if _, ok := typeSpec.Type.(*ast.StructType); ok {
						hasEndpoint = true
					} else {
						violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "Endpoint must be a struct"))
					}
				}
				if !architectureEndpointType(typeSpec.Name.Name) {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "architecture endpoint types must be Endpoint, Config, Options, Services, Dependencies, or end with Config/Routes/Auth"))
				}
			}
		case *ast.FuncDecl:
			if typed.Name.Name == "NewEndpoint" && typed.Recv == nil {
				hasNewEndpoint = true
			}
			if typed.Recv != nil {
				receiver := receiverTypeName(typed.Recv)
				if receiver != "Endpoint" && !lowerCamelIdentifier(receiver) {
					violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture endpoint receiver methods must use Endpoint or private adapter types"))
				}
				continue
			}
			if !strings.HasPrefix(typed.Name.Name, "New") && !lowerCamelIdentifier(typed.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "architecture endpoint functions must be New* or private helpers"))
			}
		}
	}
	if !hasEndpoint {
		violations = append(violations, Violation{File: displayFilename(filename), Line: 1, Message: "architecture endpoint files must declare Endpoint struct"})
	}
	if !hasNewEndpoint {
		violations = append(violations, Violation{File: displayFilename(filename), Line: 1, Message: "architecture endpoint files must declare NewEndpoint"})
	}
	return violations
}

func architectureEndpointType(name string) bool {
	return oneOf(name, "Endpoint", "Config", "Options", "Services", "Dependencies") ||
		lowerCamelIdentifier(name) ||
		strings.HasSuffix(name, "Config") ||
		strings.HasSuffix(name, "Routes") ||
		strings.HasSuffix(name, "Auth")
}
