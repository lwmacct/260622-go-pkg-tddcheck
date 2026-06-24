package filelayout

import (
	"go/ast"
	"go/token"
)

func repositoryViolations(fileSet *token.FileSet, filename string, name fileName, parsedFile *ast.File) []Violation {
	var violations []Violation
	if name.scope != "x_store" {
		violations = append(violations, Violation{
			File:    displayFilename(filename),
			Line:    1,
			Message: "repository files must use x_store scope",
		})
	}
	hasStore := false
	hasNewStore := false

	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			if typed.Tok != token.TYPE {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository files must only declare Store and Store functions"))
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if typeSpec.Name.Name != "Store" {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "repository files must only declare Store"))
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); !ok {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "Store must be a struct"))
					continue
				}
				hasStore = true
			}
		case *ast.FuncDecl:
			if typed.Recv != nil {
				if receiverTypeName(typed.Recv) != "Store" {
					violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository receiver methods must use Store"))
				}
				continue
			}
			if typed.Name.Name == "NewStore" {
				hasNewStore = true
				continue
			}
			if !lowerCamelIdentifier(typed.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository package-level functions must be NewStore or private helpers"))
			}
		}
	}
	if !hasStore {
		violations = append(violations, Violation{File: displayFilename(filename), Line: 1, Message: "repository files must declare Store struct"})
	}
	if !hasNewStore {
		violations = append(violations, Violation{File: displayFilename(filename), Line: 1, Message: "repository files must declare NewStore"})
	}
	return violations
}
