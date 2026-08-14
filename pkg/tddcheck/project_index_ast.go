package tddcheck

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func handlerIndexesFromFile(context *rulekit.Context, file rulekit.GoFile) []HandlerIndex {
	handlers := map[string]*HandlerIndex{}
	var registerNames []string
	for _, decl := range file.AST.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok != token.TYPE {
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || !strings.HasSuffix(typeSpec.Name.Name, "Handler") {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); !ok {
					continue
				}
				handler := ensureHandlerIndex(handlers, context, file, typeSpec.Name.Name)
				handler.Scope = scopeFromTypeName(typeSpec.Name.Name, "Handler")
			}
		case *ast.FuncDecl:
			if typed.Recv == nil {
				if strings.HasPrefix(typed.Name.Name, "Register") {
					registerNames = append(registerNames, typed.Name.Name)
				}
				continue
			}
			receiver := receiverTypeName(typed.Recv)
			if !strings.HasSuffix(receiver, "Handler") {
				continue
			}
			handler := ensureHandlerIndex(handlers, context, file, receiver)
			handler.Methods = append(handler.Methods, MethodIndex{
				Name: typed.Name.Name,
				Line: file.Fset.Position(typed.Name.Pos()).Line,
			})
		}
	}

	result := make([]HandlerIndex, 0, len(handlers))
	if len(handlers) == 0 && len(registerNames) > 0 {
		handler := ensureHandlerIndex(handlers, context, file, "")
		handler.Scope = scopeFromFilename(file.Base)
	}
	for _, handler := range handlers {
		handler.Registers = uniqueStrings(registerNames)
		slicesSortMethods(handler.Methods)
		result = append(result, *handler)
	}
	return result
}

func literalString(expr ast.Expr) string {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return ""
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return ""
	}
	return value
}

func ensureHandlerIndex(handlers map[string]*HandlerIndex, context *rulekit.Context, file rulekit.GoFile, name string) *HandlerIndex {
	handler := handlers[name]
	if handler == nil {
		handler = &HandlerIndex{
			Scope: scopeFromTypeName(name, "Handler"),
			Type:  name,
			File:  context.DisplayPath(file.AbsPath),
		}
		handlers[name] = handler
	}
	return handler
}

func serviceIndexesFromFile(context *rulekit.Context, file rulekit.GoFile) []ServiceIndex {
	services := map[string]*ServiceIndex{}
	for _, decl := range file.AST.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Tok != token.TYPE {
				continue
			}
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || !strings.HasSuffix(typeSpec.Name.Name, "Service") {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); !ok {
					continue
				}
				service := ensureServiceIndex(services, context, file, typeSpec.Name.Name)
				service.Scope = scopeFromTypeName(typeSpec.Name.Name, "Service")
				service.Dependencies = serviceDependencies(typeSpec)
			}
		case *ast.FuncDecl:
			if typed.Recv == nil {
				if strings.HasPrefix(typed.Name.Name, "New") && strings.HasSuffix(typed.Name.Name, "Service") {
					name := strings.TrimPrefix(typed.Name.Name, "New")
					service := ensureServiceIndex(services, context, file, name)
					service.Constructor = typed.Name.Name
				}
				continue
			}
			receiver := receiverTypeName(typed.Recv)
			if !strings.HasSuffix(receiver, "Service") {
				continue
			}
			service := ensureServiceIndex(services, context, file, receiver)
			service.Methods = append(service.Methods, MethodIndex{
				Name: typed.Name.Name,
				Line: file.Fset.Position(typed.Name.Pos()).Line,
			})
		}
	}

	result := make([]ServiceIndex, 0, len(services))
	for _, service := range services {
		if service.Scope == "" {
			service.Scope = scopeFromTypeName(service.Type, "Service")
		}
		slicesSortMethods(service.Methods)
		slicesSortStrings(service.Dependencies)
		result = append(result, *service)
	}
	return result
}

func ensureServiceIndex(services map[string]*ServiceIndex, context *rulekit.Context, file rulekit.GoFile, name string) *ServiceIndex {
	service := services[name]
	if service == nil {
		service = &ServiceIndex{
			Scope: scopeFromTypeName(name, "Service"),
			Type:  name,
			File:  context.DisplayPath(file.AbsPath),
		}
		services[name] = service
	}
	return service
}

func serviceDependencies(typeSpec *ast.TypeSpec) []string {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	for _, field := range structType.Fields.List {
		name := exprTypeName(field.Type)
		if !strings.HasSuffix(name, "Service") || name == typeSpec.Name.Name {
			continue
		}
		seen[name] = true
	}
	values := make([]string, 0, len(seen))
	for name := range seen {
		values = append(values, name)
	}
	return values
}

func storeIndexesFromFile(context *rulekit.Context, file rulekit.GoFile) []StoreIndex {
	store := StoreIndex{
		Scope: scopeFromFilename(file.Base),
		File:  context.DisplayPath(file.AbsPath),
	}
	for _, decl := range file.AST.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || receiverTypeName(funcDecl.Recv) != "Store" {
			continue
		}
		store.Methods = append(store.Methods, MethodIndex{
			Name: funcDecl.Name.Name,
			Line: file.Fset.Position(funcDecl.Name.Pos()).Line,
		})
	}
	if len(store.Methods) == 0 {
		return nil
	}
	slicesSortMethods(store.Methods)
	return []StoreIndex{store}
}

func schemaIndexesFromFile(context *rulekit.Context, file rulekit.GoFile) ([]TableIndex, []ProjectionIndex) {
	var tables []TableIndex
	var projections []ProjectionIndex
	foreignKeys := foreignKeysByModelFromFile(file)
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
			table := TableIndex{
				Scope:       scopeFromTypeName(typeSpec.Name.Name, "Model"),
				Model:       typeSpec.Name.Name,
				File:        context.DisplayPath(file.AbsPath),
				ForeignKeys: foreignKeys[typeSpec.Name.Name],
			}
			projection := ProjectionIndex{
				Scope: scopeFromTypeName(typeSpec.Name.Name, "Model"),
				Model: typeSpec.Name.Name,
				File:  context.DisplayPath(file.AbsPath),
			}
			for _, field := range structType.Fields.List {
				if embeddedBunBaseModel(field) {
					table.Table, table.Alias = tableNameFromField(field)
					continue
				}
				if embeddedProjectionField(field) {
					projection.Extends = append(projection.Extends, exprTypeName(field.Type))
					continue
				}
				fields := fieldMapsFromAST(file.Fset, field)
				table.Fields = append(table.Fields, fields...)
				projection.Fields = append(projection.Fields, fields...)
			}
			if table.Table != "" {
				tables = append(tables, table)
			} else {
				projection.Extends = uniqueStrings(projection.Extends)
				projections = append(projections, projection)
			}
		}
	}
	return tables, projections
}

func fieldMapsFromAST(fileSet *token.FileSet, field *ast.Field) []FieldIndex {
	if len(field.Names) == 0 {
		return nil
	}
	tag := bunTag(field)
	if tag == "-" {
		return nil
	}
	parts := splitTag(tag)
	goType := exprString(fileSet, field.Type)
	fields := make([]FieldIndex, 0, len(field.Names))
	for _, name := range field.Names {
		column := nameFromBunParts(parts)
		if column == "" {
			column = snakeName(name.Name)
		}
		item := FieldIndex{
			Name:   name.Name,
			Column: column,
			GoType: goType,
		}
		for _, part := range parts[1:] {
			switch {
			case part == "pk":
				item.PrimaryKey = true
			case part == "autoincrement":
				item.AutoIncrement = true
			case part == "notnull":
				item.NotNull = true
			case part == "nullzero":
				item.Nullable = true
			case part == "unique":
				item.Unique = "true"
			case strings.HasPrefix(part, "unique:"):
				item.Unique = strings.TrimPrefix(part, "unique:")
			}
		}
		fields = append(fields, item)
	}
	return fields
}

func tableNameFromField(field *ast.Field) (string, string) {
	parts := splitTag(bunTag(field))
	var table string
	var alias string
	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, "table:"):
			table = strings.TrimPrefix(part, "table:")
		case strings.HasPrefix(part, "alias:"):
			alias = strings.TrimPrefix(part, "alias:")
		}
	}
	return table, alias
}

func foreignKeysByModelFromFile(file rulekit.GoFile) map[string][]string {
	keys := map[string][]string{}
	for _, decl := range file.AST.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil || funcDecl.Name.Name != "BeforeCreateTable" {
			continue
		}
		receiver := receiverTypeName(funcDecl.Recv)
		if receiver == "" {
			continue
		}
		ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "ForeignKey" || len(call.Args) == 0 {
				return true
			}
			value := literalString(call.Args[0])
			if value != "" {
				keys[receiver] = append(keys[receiver], value)
			}
			return true
		})
	}
	for model, values := range keys {
		keys[model] = uniqueStrings(values)
	}
	return keys
}

func embeddedBunBaseModel(field *ast.Field) bool {
	if len(field.Names) > 0 {
		return false
	}
	selector, ok := field.Type.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "BaseModel" {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == "bun"
}

func embeddedProjectionField(field *ast.Field) bool {
	if len(field.Names) > 0 {
		return false
	}
	if embeddedBunBaseModel(field) {
		return false
	}
	name := exprTypeName(field.Type)
	return strings.HasSuffix(name, "Model")
}

func bunTag(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	value, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return ""
	}
	return reflect.StructTag(value).Get("bun")
}

func splitTag(tag string) []string {
	if tag == "" {
		return nil
	}
	parts := strings.Split(tag, ",")
	for index, part := range parts {
		parts[index] = strings.TrimSpace(part)
	}
	return parts
}

func nameFromBunParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	first := parts[0]
	if first == "" || strings.Contains(first, ":") {
		return ""
	}
	return first
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

func exprString(fileSet *token.FileSet, expr ast.Expr) string {
	var buffer bytes.Buffer
	if err := printer.Fprint(&buffer, fileSet, expr); err != nil {
		return ""
	}
	return buffer.String()
}

func scopeFromTypeName(name string, suffix string) string {
	return snakeName(strings.TrimSuffix(name, suffix))
}

func snakeName(value string) string {
	var builder strings.Builder
	runes := []rune(value)
	for index, char := range runes {
		if index > 0 && upperRune(char) {
			prev := runes[index-1]
			var next rune
			if index+1 < len(runes) {
				next = runes[index+1]
			}
			if !upperRune(prev) || (next != 0 && !upperRune(next)) {
				builder.WriteByte('_')
			}
		}
		builder.WriteRune(char)
	}
	return strings.ToLower(builder.String())
}

func upperRune(value rune) bool {
	return 'A' <= value && value <= 'Z'
}

func exprTypeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return exprTypeName(typed.X)
	case *ast.SelectorExpr:
		return typed.Sel.Name
	case *ast.ArrayType:
		return exprTypeName(typed.Elt)
	}
	return ""
}

func scopeFromFilename(base string) string {
	name := strings.TrimSuffix(base, ".go")
	scope, _, ok := strings.Cut(name, ".")
	if !ok {
		return ""
	}
	return scope
}

func slicesSortMethods(methods []MethodIndex) {
	for i := 1; i < len(methods); i++ {
		value := methods[i]
		j := i - 1
		for j >= 0 && methods[j].Name > value.Name {
			methods[j+1] = methods[j]
			j--
		}
		methods[j+1] = value
	}
}

func slicesSortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		value := values[i]
		j := i - 1
		for j >= 0 && values[j] > value {
			values[j+1] = values[j]
			j--
		}
		values[j+1] = value
	}
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	slicesSortStrings(result)
	return result
}
