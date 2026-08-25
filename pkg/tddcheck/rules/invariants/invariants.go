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
	for _, model := range schemaModels(snapshot) {
		violations = append(violations, automaticViolations(model)...)
	}
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
	table    string
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
	for _, model := range schemaModels(snapshot) {
		if model.file.Identity.Subject == invariant.Subject && model.table == invariant.Table {
			return model, true
		}
	}
	return schemaModel{}, false
}

func schemaModels(snapshot *rulekit.Snapshot) []schemaModel {
	var models []schemaModel
	for _, file := range snapshot.Files {
		if file.IsTest || file.Layer != "repository" || !file.IdentityOK || file.Identity.Kind != "schema" {
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
				if !ok || tableName(structType) == "" {
					continue
				}
				models = append(models, parseModel(file, typeSpec, structType))
			}
		}
	}
	return models
}

func parseModel(file rulekit.GoFile, typeSpec *ast.TypeSpec, structType *ast.StructType) schemaModel {
	model := schemaModel{
		file:     file,
		name:     typeSpec.Name.Name,
		table:    tableName(structType),
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
			fieldColumn := column
			if fieldColumn == "" {
				fieldColumn = snakeName(name.Name)
			}
			position := file.Fset.Position(field.Pos())
			model.fields[fieldColumn] = schemaField{line: position.Line, column: position.Column}
			for _, option := range parts[1:] {
				switch {
				case option == "unique":
					model.global[fieldColumn] = true
				case strings.HasPrefix(option, "unique:"):
					group := strings.TrimPrefix(option, "unique:")
					model.unique[group] = append(model.unique[group], fieldColumn)
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

func automaticViolations(model schemaModel) []Violation {
	var violations []Violation
	if model.global["idempotency_key"] {
		if scope := firstScopeField(model); scope != "" {
			violations = append(violations, modelViolation(model, RuleID+"/global-idempotency-key", fmt.Sprintf("schema model %s marks idempotency_key globally unique despite %s scope; use a composite unique group", model.name, scope)))
		}
	}
	if isNamedModel(model, "consumption", "idempotency") && firstScopeField(model) != "" && hasField(model, "idempotency_key") && !model.global["idempotency_key"] && !hasUniqueGroupContaining(model, "idempotency_key", firstScopeField(model)) {
		violations = append(violations, modelViolation(model, RuleID+"/unscoped-idempotency-key", fmt.Sprintf("schema model %s must scope idempotency_key with %s in one unique group", model.name, firstScopeField(model))))
	}
	if isNamedModel(model, "claim", "trial") && hasField(model, "user_id") && hasField(model, "claim_date") && !hasUniqueGroupContaining(model, "user_id", "claim_date") {
		violations = append(violations, modelViolation(model, RuleID+"/unscoped-claim-date", fmt.Sprintf("schema model %s must define one unique group covering user_id and claim_date", model.name)))
	}
	if isNamedModel(model, "package") && hasField(model, "user_id") && hasField(model, "source_type") && hasField(model, "source_id") && !hasUniqueGroupContaining(model, "user_id", "source_type", "source_id") {
		violations = append(violations, modelViolation(model, RuleID+"/unscoped-source", fmt.Sprintf("schema model %s must define one unique group covering user_id, source_type, and source_id", model.name)))
	}
	return violations
}

func modelViolation(model schemaModel, code string, message string) Violation {
	line, column := model.position.Line, model.position.Column
	return Violation{File: rulekit.DisplayFilename(model.file.AbsPath), Line: line, Column: column, Code: code, Message: message}
}

func hasField(model schemaModel, column string) bool {
	_, ok := model.fields[column]
	return ok
}

func firstScopeField(model schemaModel) string {
	for _, column := range []string{"user_id", "tenant_id", "account_id", "workspace_id", "organization_id", "owner_id"} {
		if hasField(model, column) {
			return column
		}
	}
	return ""
}

func hasUniqueGroupContaining(model schemaModel, columns ...string) bool {
	for _, group := range model.unique {
		found := make(map[string]bool, len(group))
		for _, column := range group {
			found[column] = true
		}
		if len(found) != len(group) {
			continue
		}
		matched := true
		for _, column := range columns {
			if !found[column] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func isNamedModel(model schemaModel, tokens ...string) bool {
	value := strings.ToLower(model.name + " " + model.table)
	for _, token := range tokens {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
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
