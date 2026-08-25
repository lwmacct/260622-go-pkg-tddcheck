// Package invariants checks explicitly configured schema-level business
// invariants. It deliberately does not infer concurrency or transaction
// semantics from service syntax.
package invariants

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

const RuleID = "invariants"

type Violation struct {
	File    string
	Line    int
	Column  int
	Code    string
	Message string
}

func Register(engine *rulekit.Engine) {
	engine.Register(RuleID, rulekit.SnapshotScope, checkSnapshot)
}

func Check(ctx context.Context, root string, config rulekit.Config) ([]Violation, error) {
	snapshot, err := rulekit.Load(ctx, root, config)
	if err != nil {
		return nil, err
	}
	return violationsInSnapshot(snapshot), nil
}

func checkSnapshot(_ context.Context, snapshot *rulekit.Snapshot, _ *rulekit.Snapshot) ([]rulekit.Diagnostic, error) {
	values := violationsInSnapshot(snapshot)
	diagnostics := make([]rulekit.Diagnostic, 0, len(values))
	for _, value := range values {
		position := rulekit.Position{File: value.File, Line: value.Line, Column: value.Column}
		code := value.Code
		if code == "" {
			code = RuleID + "/schema-invariant"
		}
		diagnostics = append(diagnostics, rulekit.NewDiagnostic(RuleID, code, rulekit.SeverityError, value.Message, position, position))
	}
	return diagnostics, nil
}

func violationsInSnapshot(snapshot *rulekit.Snapshot) []Violation {
	var violations []Violation
	for _, invariant := range snapshot.Config.SchemaInvariants {
		model, ok := findModel(snapshot, invariant)
		if !ok {
			violations = append(violations, Violation{
				File:    "schema-invariants",
				Line:    1,
				Code:    RuleID + "/missing-model",
				Message: fmt.Sprintf("schema invariant for subject %q and table %q has no matching repository schema model", invariant.Subject, invariant.Table),
			})
			continue
		}
		violations = append(violations, checkModel(model, invariant)...)
	}
	return violations
}

type schemaModel struct {
	file     rulekit.GoFile
	name     string
	position token.Position
	fields   map[string]schemaField
	unique   map[string][]string
	global   map[string]bool
}

type schemaField struct {
	line   int
	column int
}

func findModel(snapshot *rulekit.Snapshot, invariant rulekit.SchemaInvariant) (schemaModel, bool) {
	for _, file := range snapshot.Files {
		if file.IsTest || file.Layer != "repository" || !file.IdentityOK || file.Identity.Kind != "schema" || file.Identity.Subject != invariant.Subject {
			continue
		}
		for _, decl := range file.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || !strings.HasSuffix(typeSpec.Name.Name, "Model") {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				table := tableName(structType)
				if table != invariant.Table {
					continue
				}
				return parseModel(file, typeSpec, structType), true
			}
		}
	}
	return schemaModel{}, false
}

func parseModel(file rulekit.GoFile, typeSpec *ast.TypeSpec, structType *ast.StructType) schemaModel {
	model := schemaModel{
		file:     file,
		name:     typeSpec.Name.Name,
		position: file.Fset.Position(typeSpec.Pos()),
		fields:   map[string]schemaField{},
		unique:   map[string][]string{},
		global:   map[string]bool{},
	}
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		parts := bunParts(field)
		column := ""
		if len(parts) > 0 {
			column = strings.TrimSpace(parts[0])
		}
		for _, name := range field.Names {
			if column == "" {
				column = snakeName(name.Name)
			}
			position := file.Fset.Position(field.Pos())
			model.fields[column] = schemaField{line: position.Line, column: position.Column}
			for _, option := range parts[1:] {
				switch {
				case option == "unique":
					model.global[column] = true
				case strings.HasPrefix(option, "unique:"):
					group := strings.TrimPrefix(option, "unique:")
					model.unique[group] = append(model.unique[group], column)
				}
			}
		}
	}
	for group := range model.unique {
		sort.Strings(model.unique[group])
	}
	return model
}

func checkModel(model schemaModel, invariant rulekit.SchemaInvariant) []Violation {
	var violations []Violation
	for _, columns := range invariant.Unique {
		want := sortedUnique(columns)
		found := false
		for _, actual := range model.unique {
			if equalStrings(actual, want) {
				found = true
				break
			}
		}
		if found || (len(want) == 1 && model.global[want[0]]) {
			continue
		}
		line, column := model.position.Line, model.position.Column
		for _, name := range want {
			if field, ok := model.fields[name]; ok {
				line, column = field.line, field.column
				break
			}
		}
		message := fmt.Sprintf("schema model %s must define one unique group covering columns (%s)", model.name, strings.Join(want, ", "))
		for _, name := range want {
			if model.global[name] && len(want) > 1 {
				message += fmt.Sprintf("; column %s currently uses global unique", name)
				break
			}
		}
		violations = append(violations, Violation{File: rulekit.DisplayFilename(model.file.AbsPath), Line: line, Column: column, Code: RuleID + "/missing-unique-group", Message: message})
	}
	return violations
}

func tableName(structType *ast.StructType) string {
	for _, field := range structType.Fields.List {
		if len(field.Names) != 0 {
			continue
		}
		selector, ok := field.Type.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "BaseModel" {
			continue
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok || ident.Name != "bun" {
			continue
		}
		for _, part := range bunParts(field) {
			if strings.HasPrefix(part, "table:") {
				return strings.TrimPrefix(part, "table:")
			}
		}
	}
	return ""
}

func bunParts(field *ast.Field) []string {
	if field.Tag == nil {
		return nil
	}
	raw := strings.Trim(field.Tag.Value, "`")
	value := reflect.StructTag(raw).Get("bun")
	if value == "" || value == "-" {
		return nil
	}
	return strings.Split(value, ",")
}

func sortedUnique(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func snakeName(value string) string {
	var result []rune
	for index, char := range value {
		if unicode.IsUpper(char) && index > 0 {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(char))
	}
	return string(result)
}
