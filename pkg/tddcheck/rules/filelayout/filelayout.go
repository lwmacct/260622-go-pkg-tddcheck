package filelayout

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

const RuleID = "filelayout"

type Rules struct {
	root   string
	config rulekit.Config
}

type Violation struct {
	File    string
	Line    int
	Message string
}

type fileName struct {
	scope string
	kind  string
}

func New(root string, options ...rulekit.Option) Rules {
	values := rulekit.NewRuleOptions(root, options...)
	return Rules{root: values.Root, config: values.Config}
}

func (r Rules) Assert(t *testing.T) {
	t.Helper()

	violations, err := r.Violations()
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) == 0 {
		return
	}

	lines := make([]string, 0, len(violations))
	for _, violation := range violations {
		lines = append(lines, fmt.Sprintf("%s:%d: %s", violation.File, violation.Line, violation.Message))
	}
	t.Fatalf("invalid file layout:\n  - %s", strings.Join(lines, "\n  - "))
}

func (r Rules) Violations() ([]Violation, error) {
	root, err := rulekit.ResolveRuleRoot(r.root, RuleID)
	if err != nil {
		return nil, err
	}
	files, err := rulekit.ModuleFiles(root, RuleID, r.config, func(name string) bool {
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	})
	if err != nil {
		return nil, err
	}

	config := r.config.WithDefaults()
	var violations []Violation
	for _, file := range files {
		layer, ok := layerForFile(root, file, config)
		if !ok {
			continue
		}
		fileViolations, err := violationsInFile(config, layer, file)
		if err != nil {
			return nil, err
		}
		violations = append(violations, fileViolations...)
	}
	return violations, nil
}

func violationsInFile(config rulekit.Config, layer string, filename string) ([]Violation, error) {
	base := filepath.Base(filename)
	parsed, ok := parseFileName(base)
	if !ok {
		return []Violation{{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("%s file must use {scope}.{type}.go naming", layer),
		}}, nil
	}

	var violations []Violation
	if rulekit.StringIn(parsed.scope, config.ForbiddenWeakScopes) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("scope %q is too weak; use a resource name or approved shared scope", parsed.scope),
		})
	}
	if !strings.HasPrefix(parsed.scope, "x_") {
		if escapedKind, ok := escapedKindScope(parsed.scope); ok {
			violations = append(violations, Violation{
				File:    rulekit.DisplayFilename(filename),
				Line:    1,
				Message: fmt.Sprintf("scope %q must not encode file type %q; use the resource scope and a single type suffix", parsed.scope, escapedKind),
			})
		}
	}
	if !strings.HasPrefix(parsed.scope, "x_") && reservedArchitectureScope(parsed.scope) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q must use x_%s prefix", parsed.scope, parsed.scope),
		})
	}
	if strings.HasPrefix(parsed.scope, "x_") && !reservedArchitectureScope(strings.TrimPrefix(parsed.scope, "x_")) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q is not reserved", parsed.scope),
		})
	}
	if !allowedKind(layer, parsed.kind) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("%s file type %q is not allowed", layer, parsed.kind),
		})
	}
	if strings.HasPrefix(parsed.scope, "x_") && !allowedArchitectureKind(layer, parsed.scope, parsed.kind) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q must not use %s.%s.go in %s", parsed.scope, parsed.scope, parsed.kind, layer),
		})
	}
	scopeViolations, err := inferredScopeViolations(parsed, filename)
	if err != nil {
		return nil, err
	}
	violations = append(violations, scopeViolations...)

	declViolations, err := declarationViolations(layer, parsed, filename)
	if err != nil {
		return nil, err
	}
	violations = append(violations, declViolations...)
	return violations, nil
}

func inferredScopeViolations(name fileName, filename string) ([]Violation, error) {
	if strings.HasPrefix(name.scope, "x_") {
		return nil, nil
	}
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, filename, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	for _, identifier := range fileIdentifiers(parsedFile) {
		if expectedScope, ok := inferredSnakeScope(name.scope, identifier); ok && expectedScope != name.scope {
			return []Violation{{
				File:    rulekit.DisplayFilename(filename),
				Line:    1,
				Message: fmt.Sprintf("scope %q must use snake_case resource name %q inferred from %s", name.scope, expectedScope, identifier),
			}}, nil
		}
	}
	return nil, nil
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

func parseFileName(base string) (fileName, bool) {
	if !strings.HasSuffix(base, ".go") {
		return fileName{}, false
	}
	name := strings.TrimSuffix(base, ".go")
	parts := strings.Split(name, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fileName{}, false
	}
	if strings.Contains(parts[1], "_") {
		return fileName{}, false
	}
	return fileName{scope: parts[0], kind: parts[1]}, true
}

func layerForFile(root string, file string, config rulekit.Config) (string, bool) {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return "", false
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if slices.Contains(config.LayerDirs, part) {
			return part, true
		}
	}
	return "", false
}

func allowedKind(layer string, kind string) bool {
	switch layer {
	case "handler":
		return oneOf(kind, "dto", "handler", "mapper", "utils")
	case "service":
		return oneOf(kind, "commands", "constants", "errors", "mapper", "models", "service", "utils", "validation", "writes")
	case "repository":
		return oneOf(kind, "constants", "database", "errors", "model", "models", "repository", "schema", "store", "utils")
	default:
		return false
	}
}

func allowedArchitectureKind(layer string, scope string, kind string) bool {
	switch layer {
	case "handler":
		switch scope {
		case "x_api", "x_frontend", "x_router", "x_shared":
			return oneOf(kind, "handler", "utils")
		default:
			return false
		}
	case "service":
		switch scope {
		case "x_batch":
			return kind == "service"
		case "x_id":
			return kind == "validation"
		case "x_shared":
			return oneOf(kind, "errors", "mapper", "models", "utils", "validation", "writes")
		default:
			return false
		}
	case "repository":
		switch scope {
		case "x_database":
			return kind == "repository"
		case "x_schema":
			return oneOf(kind, "repository", "utils")
		case "x_store":
			return kind == "repository"
		case "x_shared":
			return oneOf(kind, "constants", "errors", "models", "utils")
		default:
			return false
		}
	default:
		return false
	}
}

func escapedKindScope(scope string) (string, bool) {
	for _, kind := range []string{
		"commands",
		"constants",
		"create",
		"delete",
		"dto",
		"errors",
		"handler",
		"list",
		"mapper",
		"model",
		"models",
		"patch",
		"repository",
		"schema",
		"service",
		"store",
		"update",
		"upsert",
		"utils",
		"validation",
		"writes",
	} {
		if strings.HasSuffix(scope, "_"+kind) {
			return kind, true
		}
	}
	return "", false
}

func reservedArchitectureScope(scope string) bool {
	return oneOf(
		scope,
		"api",
		"batch",
		"database",
		"frontend",
		"id",
		"ids",
		"repository",
		"router",
		"schema",
		"shared",
		"store",
		"write",
	)
}

func declarationViolations(layer string, name fileName, filename string) ([]Violation, error) {
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, filename, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	switch name.kind {
	case "dto":
		return dtoViolations(fileSet, filename, parsedFile), nil
	case "handler":
		if layer == "handler" && !strings.HasPrefix(name.scope, "x_") {
			return handlerViolations(fileSet, filename, name, parsedFile), nil
		}
	case "mapper":
		return mapperViolations(fileSet, filename, parsedFile), nil
	case "commands":
		return commandsViolations(fileSet, filename, parsedFile), nil
	case "constants":
		return constantsViolations(fileSet, filename, parsedFile), nil
	case "validation":
		return validationViolations(fileSet, filename, parsedFile), nil
	case "errors":
		return errorsViolations(fileSet, filename, parsedFile), nil
	case "utils":
		return utilsViolations(fileSet, filename, parsedFile), nil
	case "service":
		if layer == "service" {
			if name.scope == "x_batch" {
				return nil, nil
			}
			return serviceViolations(fileSet, filename, name, parsedFile), nil
		}
	case "store":
		if layer == "repository" {
			return storeViolations(fileSet, filename, parsedFile), nil
		}
	case "schema":
		if layer == "repository" {
			return schemaViolations(fileSet, filename, parsedFile), nil
		}
	case "model", "models", "writes":
		if layer == "repository" && name.kind == "model" {
			return repositoryModelViolations(fileSet, filename, parsedFile), nil
		}
		return typeOnlyViolations(fileSet, filename, parsedFile, name.kind+" files must only declare types"), nil
	}
	return nil, nil
}

func dtoViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "dto files must not declare functions"))
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			if typed.Tok != token.TYPE {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "dto files must only declare DTO types"))
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && !strings.HasSuffix(typeSpec.Name.Name, "DTO") && !strings.HasSuffix(typeSpec.Name.Name, "DTOs") {
					violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "DTO type "+typeSpec.Name.Name+" must end with DTO or DTOs"))
				}
			}
		}
	}
	return violations
}

func commandsViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			violations = append(violations, violationAt(fileSet, filename, decl.Pos(), "commands files must only declare types"))
			continue
		}
		if genDecl.Tok == token.IMPORT {
			continue
		}
		if genDecl.Tok != token.TYPE {
			violations = append(violations, violationAt(fileSet, filename, genDecl.Pos(), "commands files must only declare types"))
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && !commandTypeName(typeSpec.Name.Name) {
				violations = append(violations, violationAt(fileSet, filename, typeSpec.Pos(), "command type "+typeSpec.Name.Name+" must end with Request, Response, Result, or Item"))
			}
		}
	}
	return violations
}

func commandTypeName(name string) bool {
	return strings.HasSuffix(name, "Request") ||
		strings.HasSuffix(name, "Response") ||
		strings.HasSuffix(name, "Result") ||
		strings.HasSuffix(name, "Item")
}

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

func mapperViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				for _, importSpec := range typed.Specs {
					spec, ok := importSpec.(*ast.ImportSpec)
					if !ok {
						continue
					}
					importPath := strings.Trim(spec.Path.Value, `"`)
					if forbiddenMapperImport(importPath) {
						violations = append(violations, violationAt(fileSet, filename, spec.Pos(), "mapper files must not import "+importPath))
					}
				}
			} else {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "mapper files must only declare functions"))
			}
		case *ast.FuncDecl:
			if typed.Recv != nil {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "mapper functions must not use receivers"))
			}
			if !strings.HasPrefix(typed.Name.Name, "To") {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "mapper function "+typed.Name.Name+" must start with To"))
			}
		}
	}
	return violations
}

func utilsViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "utils files must only declare util* functions"))
		case *ast.FuncDecl:
			if typed.Recv != nil {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "utils functions must not use receivers"))
			}
			if !strings.HasPrefix(typed.Name.Name, "util") {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "utils function "+typed.Name.Name+" must start with util"))
			}
		}
	}
	return violations
}

func forbiddenMapperImport(importPath string) bool {
	return oneOf(
		importPath,
		"context",
		"database/sql",
		"net/http",
		"github.com/danielgtaylor/huma/v2",
		"github.com/uptrace/bun",
		"gorm.io/gorm",
	)
}

func typeOnlyViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File, message string) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && (genDecl.Tok == token.IMPORT || genDecl.Tok == token.TYPE) {
			continue
		}
		violations = append(violations, violationAt(fileSet, filename, decl.Pos(), message))
	}
	return violations
}

func repositoryModelViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT || typed.Tok == token.TYPE {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository model files must only declare types and model receiver methods"))
		case *ast.FuncDecl:
			if typed.Recv != nil {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "repository model files must not declare package-level functions"))
		}
	}
	return violations
}

func constantsViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && (genDecl.Tok == token.IMPORT || genDecl.Tok == token.CONST) {
			continue
		}
		violations = append(violations, violationAt(fileSet, filename, decl.Pos(), "constants files must only declare const"))
	}
	return violations
}

func validationViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			if typed.Recv != nil || !(strings.HasPrefix(typed.Name.Name, "validate") || strings.HasPrefix(typed.Name.Name, "normalize")) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "validation functions must be package-level validate* or normalize*"))
			}
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT || typed.Tok == token.CONST || typed.Tok == token.VAR {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "validation files must not declare types"))
		}
	}
	return violations
}

func errorsViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			name := typed.Name.Name
			if typed.Recv != nil {
				if !oneOf(name, "Error", "Is", "As", "Unwrap") {
					violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "error receiver methods must be Error, Is, As, or Unwrap"))
				}
				continue
			}
			if !(strings.HasPrefix(name, "Wrap") || strings.HasPrefix(name, "Is") || strings.HasPrefix(name, "As")) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "error helper functions must start with Wrap, Is, or As"))
			}
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT || typed.Tok == token.VAR || typed.Tok == token.TYPE {
				continue
			}
			violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "errors files must only declare error vars, error types, and error helpers"))
		}
	}
	return violations
}

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

func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch typed := recv.List[0].Type.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		if ident, ok := typed.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func upperCamelName(value string) string {
	parts := strings.Split(value, "_")
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	return builder.String()
}

func lowerCamelName(value string) string {
	upper := upperCamelName(value)
	if upper == "" {
		return ""
	}
	return strings.ToLower(upper[:1]) + upper[1:]
}

func snakeName(value string) string {
	var builder strings.Builder
	for index, char := range value {
		if index > 0 && 'A' <= char && char <= 'Z' {
			builder.WriteByte('_')
		}
		builder.WriteRune(char)
	}
	return strings.ToLower(builder.String())
}

func storeViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Recv == nil && !strings.HasPrefix(funcDecl.Name.Name, "New") {
				violations = append(violations, violationAt(fileSet, filename, funcDecl.Pos(), "store files must declare constructors or Store receiver methods"))
			}
		}
	}
	return violations
}

func schemaViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Recv == nil {
			if !oneOf(funcDecl.Name.Name, "Schema", "CreateIndexes", "CreateSchema", "EnsureSchema") &&
				!strings.HasSuffix(funcDecl.Name.Name, "Schema") {
				violations = append(violations, violationAt(fileSet, filename, funcDecl.Pos(), "schema package-level functions must be schema lifecycle functions"))
			}
		}
	}
	return violations
}

func violationAt(fileSet *token.FileSet, filename string, pos token.Pos, message string) Violation {
	return Violation{File: rulekit.DisplayFilename(filename), Line: fileSet.Position(pos).Line, Message: message}
}

func oneOf(value string, values ...string) bool {
	return slices.Contains(values, value)
}
