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

func serviceMapFromFile(context *rulekit.Context, file rulekit.GoFile) []ServiceMap {
	services := map[string]*ServiceMap{}
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
				service := ensureServiceMap(services, context, file, typeSpec.Name.Name)
				service.Scope = scopeFromTypeName(typeSpec.Name.Name, "Service")
			}
		case *ast.FuncDecl:
			if typed.Recv == nil {
				if strings.HasPrefix(typed.Name.Name, "New") && strings.HasSuffix(typed.Name.Name, "Service") {
					name := strings.TrimPrefix(typed.Name.Name, "New")
					service := ensureServiceMap(services, context, file, name)
					service.Constructor = typed.Name.Name
				}
				continue
			}
			receiver := receiverTypeName(typed.Recv)
			if !strings.HasSuffix(receiver, "Service") {
				continue
			}
			service := ensureServiceMap(services, context, file, receiver)
			service.Methods = append(service.Methods, MethodMap{
				Name: typed.Name.Name,
				Line: file.Fset.Position(typed.Name.Pos()).Line,
			})
		}
	}

	result := make([]ServiceMap, 0, len(services))
	for _, service := range services {
		if service.Scope == "" {
			service.Scope = scopeFromTypeName(service.Name, "Service")
		}
		slicesSortMethods(service.Methods)
		result = append(result, *service)
	}
	return result
}

func ensureServiceMap(services map[string]*ServiceMap, context *rulekit.Context, file rulekit.GoFile, name string) *ServiceMap {
	service := services[name]
	if service == nil {
		service = &ServiceMap{
			Scope: scopeFromTypeName(name, "Service"),
			Name:  name,
			File:  context.DisplayPath(file.AbsPath),
		}
		services[name] = service
	}
	return service
}

func tableMapsFromFile(context *rulekit.Context, file rulekit.GoFile) []TableMap {
	var tables []TableMap
	foreignKeys := foreignKeysFromFile(file)
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
			table := TableMap{
				Scope:       scopeFromTypeName(typeSpec.Name.Name, "Model"),
				Model:       typeSpec.Name.Name,
				File:        context.DisplayPath(file.AbsPath),
				ForeignKeys: foreignKeys,
			}
			for _, field := range structType.Fields.List {
				if embeddedBunBaseModel(field) {
					table.Table, table.Alias = tableNameFromField(field)
					continue
				}
				table.Fields = append(table.Fields, fieldMapsFromAST(file.Fset, field)...)
			}
			tables = append(tables, table)
		}
	}
	return tables
}

func fieldMapsFromAST(fileSet *token.FileSet, field *ast.Field) []FieldMap {
	if len(field.Names) == 0 {
		return nil
	}
	tag := bunTag(field)
	if tag == "-" {
		return nil
	}
	parts := splitTag(tag)
	goType := exprString(fileSet, field.Type)
	fields := make([]FieldMap, 0, len(field.Names))
	for _, name := range field.Names {
		column := nameFromBunParts(parts)
		if column == "" {
			column = snakeName(name.Name)
		}
		item := FieldMap{
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

func foreignKeysFromFile(file rulekit.GoFile) []string {
	var keys []string
	ast.Inspect(file.AST, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "ForeignKey" || len(call.Args) == 0 {
			return true
		}
		literal, ok := call.Args[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		keys = append(keys, value)
		return true
	})
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

func slicesSortMethods(methods []MethodMap) {
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
