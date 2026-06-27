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

func apiIndexesFromFile(context *rulekit.Context, file rulekit.GoFile) []APIIndex {
	var result []APIIndex
	for _, decl := range file.AST.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			continue
		}
		result = append(result, apiIndexesFromFunc(context, file, funcDecl)...)
	}
	return result
}

func apiIndexesFromFunc(context *rulekit.Context, file rulekit.GoFile, funcDecl *ast.FuncDecl) []APIIndex {
	register := registerName(funcDecl)
	scope := scopeFromFilename(file.Base)
	groups := map[string]string{}
	var apis []APIIndex
	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			recordAPIGroups(file, groups, typed.Lhs, typed.Rhs)
		case *ast.ValueSpec:
			recordAPIGroups(file, groups, identExprs(typed.Names), typed.Values)
		case *ast.CallExpr:
			api, ok := apiIndexFromCall(context, file, scope, register, groups, typed)
			if ok {
				apis = append(apis, api)
			}
		}
		return true
	})
	return apis
}

func registerName(funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv == nil {
		return funcDecl.Name.Name
	}
	receiver := receiverTypeName(funcDecl.Recv)
	if receiver == "" {
		return funcDecl.Name.Name
	}
	return receiver + "." + funcDecl.Name.Name
}

func identExprs(names []*ast.Ident) []ast.Expr {
	values := make([]ast.Expr, 0, len(names))
	for _, name := range names {
		values = append(values, name)
	}
	return values
}

func recordAPIGroups(file rulekit.GoFile, groups map[string]string, left []ast.Expr, right []ast.Expr) {
	for index, expr := range right {
		if index >= len(left) {
			return
		}
		name, ok := left[index].(*ast.Ident)
		if !ok {
			continue
		}
		prefix, ok := apiGroupPrefix(file, groups, expr)
		if ok {
			groups[name.Name] = prefix
		}
	}
}

func apiGroupPrefix(file rulekit.GoFile, groups map[string]string, expr ast.Expr) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) < 2 {
		return "", false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "NewGroup" {
		return "", false
	}
	prefix := literalString(call.Args[1])
	if prefix == "" {
		return "", false
	}
	parent := groups[exprString(file.Fset, call.Args[0])]
	return joinAPIPath(parent, prefix), true
}

func apiIndexFromCall(context *rulekit.Context, file rulekit.GoFile, scope string, register string, groups map[string]string, call *ast.CallExpr) (APIIndex, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Register" || len(call.Args) < 2 {
		return APIIndex{}, false
	}
	operation, ok := operationIndexFromExpr(file, call.Args[1])
	if !ok {
		return APIIndex{}, false
	}
	operation.Scope = scope
	operation.OperationPath = joinAPIPath(groups[exprString(file.Fset, call.Args[0])], operation.OperationPath)
	operation.FullPath = operation.OperationPath
	operation.Register = register
	operation.File = context.DisplayPath(file.AbsPath)
	operation.Line = file.Fset.Position(call.Lparen).Line
	if len(call.Args) >= 3 {
		operation.Handler = apiHandlerName(file, call.Args[2])
	}
	return operation, true
}

func operationIndexFromExpr(file rulekit.GoFile, expr ast.Expr) (APIIndex, bool) {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok {
		return APIIndex{}, false
	}
	var api APIIndex
	for _, element := range literal.Elts {
		keyValue, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := keyValue.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "OperationID":
			api.OperationID = literalString(keyValue.Value)
		case "Method":
			api.Method = httpMethodName(file, keyValue.Value)
		case "Path":
			api.OperationPath = literalString(keyValue.Value)
		case "Tags":
			api.Tags = literalStringSlice(keyValue.Value)
		}
	}
	api.FullPath = api.OperationPath
	return api, api.OperationID != "" || api.Method != "" || api.OperationPath != ""
}

func apiHandlerName(file rulekit.GoFile, expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.SelectorExpr:
		return typed.Sel.Name
	case *ast.FuncLit:
		return "inline"
	}
	return exprString(file.Fset, expr)
}

func applyRouteMounts(context *rulekit.Context, index *Index) {
	funcs := routeFuncsFromContext(context)
	mounts := map[string]string{}
	queue := []routeCall{{Func: "", Arg: "", MountPath: ""}}
	for _, call := range rootRouteCalls(context) {
		queue = append(queue, call)
	}
	for _, fn := range funcs {
		for _, event := range fn.Events {
			if event.MountPath == "" {
				continue
			}
			queue = append(queue, routeCall{Func: event.Callee, Arg: event.Arg, MountPath: event.MountPath})
		}
	}
	seen := map[routeCall]bool{}
	for len(queue) > 0 {
		call := queue[0]
		queue = queue[1:]
		if seen[call] {
			continue
		}
		seen[call] = true
		fn := funcs[call.Func]
		if fn == nil {
			continue
		}
		for _, event := range fn.Events {
			mountPath := event.MountPath
			if mountPath == "" && event.Arg == call.Arg {
				mountPath = call.MountPath
			}
			if mountPath == "" && event.Arg != "" {
				mountPath = call.MountPath
			}
			switch event.Kind {
			case routeEventRegister:
				if mountPath == "" {
					continue
				}
				key := apiMountKey(event.Callee, event.File)
				if previous, ok := mounts[key]; !ok || len(mountPath) > len(previous) {
					mounts[key] = mountPath
				}
			case routeEventCall:
				queue = append(queue, routeCall{Func: event.Callee, Arg: event.Arg, MountPath: mountPath})
			}
		}
	}
	for apiIndex := range index.APIs {
		api := &index.APIs[apiIndex]
		mountPath := mounts[apiMountKey(api.Register, api.File)]
		api.MountPath = mountPath
		api.FullPath = joinAPIPath(mountPath, api.OperationPath)
	}
}

type routeEventKind int

const (
	routeEventCall routeEventKind = iota
	routeEventRegister
)

type routeCall struct {
	Func      string
	Arg       string
	MountPath string
}

type routeFunc struct {
	Name   string
	File   string
	Events []routeEvent
}

type routeEvent struct {
	Kind      routeEventKind
	Callee    string
	Arg       string
	MountPath string
	File      string
}

func routeFuncsFromContext(context *rulekit.Context) map[string]*routeFunc {
	funcs := map[string]*routeFunc{}
	for _, file := range context.Files {
		if file.Layer != "handler" || rulekit.FreeFile(file.Base) {
			continue
		}
		displayFile := context.DisplayPath(file.AbsPath)
		for _, decl := range file.AST.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Body == nil {
				continue
			}
			name := registerName(funcDecl)
			fn := &routeFunc{Name: name, File: displayFile}
			receivers := receiverAliases(funcDecl)
			apiAliases := apiParamAliases(funcDecl)
			ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.AssignStmt:
					recordMountedAPIAliases(file, apiAliases, typed.Lhs, typed.Rhs)
				case *ast.ValueSpec:
					recordMountedAPIAliases(file, apiAliases, identExprs(typed.Names), typed.Values)
				case *ast.CallExpr:
					fn.Events = append(fn.Events, routeEventsFromCall(file, displayFile, receivers, apiAliases, typed)...)
				}
				return true
			})
			funcs[name] = fn
		}
	}
	return funcs
}

func rootRouteCalls(context *rulekit.Context) []routeCall {
	var calls []routeCall
	for _, file := range context.Files {
		if file.Layer != "handler" || rulekit.FreeFile(file.Base) {
			continue
		}
		for _, decl := range file.AST.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Body == nil {
				continue
			}
			receivers := receiverAliases(funcDecl)
			ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				callee := callName(file, receivers, call.Fun)
				if callee == "" || (!strings.HasPrefix(callee, "Register") && !strings.Contains(callee, ".Register")) {
					return true
				}
				if len(call.Args) == 0 {
					return true
				}
				mount, ok := mountPathFromAPIExpr(file, call.Args[0])
				if ok {
					calls = append(calls, routeCall{Func: callee, Arg: "", MountPath: mount})
				}
				return true
			})
		}
	}
	return calls
}

func apiParamAliases(funcDecl *ast.FuncDecl) map[string]string {
	aliases := map[string]string{}
	if funcDecl.Type.Params == nil {
		return aliases
	}
	for _, field := range funcDecl.Type.Params.List {
		if !isHumaAPIType(field.Type) {
			continue
		}
		for _, name := range field.Names {
			aliases[name.Name] = ""
		}
	}
	return aliases
}

func receiverAliases(funcDecl *ast.FuncDecl) map[string]string {
	aliases := map[string]string{}
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return aliases
	}
	receiverType := receiverTypeName(funcDecl.Recv)
	for _, name := range funcDecl.Recv.List[0].Names {
		aliases[name.Name] = receiverType
	}
	return aliases
}

func isHumaAPIType(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "API" {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == "huma"
}

func recordMountedAPIAliases(file rulekit.GoFile, aliases map[string]string, left []ast.Expr, right []ast.Expr) {
	for index, expr := range right {
		if index >= len(left) {
			return
		}
		name, ok := left[index].(*ast.Ident)
		if !ok {
			continue
		}
		if mountPath, ok := mountPathFromAPIExpr(file, expr); ok {
			aliases[name.Name] = mountPath
		}
	}
}

func routeEventsFromCall(file rulekit.GoFile, displayFile string, receivers map[string]string, apiAliases map[string]string, call *ast.CallExpr) []routeEvent {
	var events []routeEvent
	if prefix, body, ok := chiRouteCall(call); ok {
		events = append(events, routeEventsFromRouteBody(file, displayFile, receivers, apiAliases, prefix, body)...)
		return events
	}
	callee := callName(file, receivers, call.Fun)
	if callee == "" {
		return nil
	}
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "Register" {
		return nil
	}
	if len(call.Args) == 0 {
		return nil
	}
	argName := exprString(file.Fset, call.Args[0])
	mountPath, apiArg := apiAliases[argName]
	if !apiArg && !strings.HasPrefix(callee, "Register") && !strings.Contains(callee, ".Register") {
		return nil
	}
	kind := routeEventCall
	if registerMethodCall(callee) {
		kind = routeEventRegister
	}
	events = append(events, routeEvent{Kind: kind, Callee: callee, Arg: argName, MountPath: mountPath, File: displayFile})
	return events
}

func routeEventsFromRouteBody(file rulekit.GoFile, displayFile string, receivers map[string]string, parentAliases map[string]string, prefix string, body *ast.FuncLit) []routeEvent {
	if body.Type.Params == nil || len(body.Type.Params.List) == 0 {
		return nil
	}
	aliases := map[string]string{}
	for name, mountPath := range parentAliases {
		aliases[name] = mountPath
	}
	for _, param := range body.Type.Params.List {
		for _, name := range param.Names {
			aliases[name.Name] = joinAPIPath("", prefix)
		}
	}
	var events []routeEvent
	ast.Inspect(body.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			recordMountedAPIAliases(file, aliases, typed.Lhs, typed.Rhs)
		case *ast.ValueSpec:
			recordMountedAPIAliases(file, aliases, identExprs(typed.Names), typed.Values)
		case *ast.CallExpr:
			events = append(events, routeEventsFromCall(file, displayFile, receivers, aliases, typed)...)
		}
		return true
	})
	return events
}

func chiRouteCall(call *ast.CallExpr) (string, *ast.FuncLit, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Route" || len(call.Args) < 2 {
		return "", nil, false
	}
	prefix := literalString(call.Args[0])
	if prefix == "" {
		return "", nil, false
	}
	body, ok := call.Args[1].(*ast.FuncLit)
	return prefix, body, ok
}

func mountPathFromAPIExpr(file rulekit.GoFile, expr ast.Expr) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return "", false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "New" {
		return "", false
	}
	pkg, ok := selector.X.(*ast.Ident)
	if !ok || pkg.Name != "humachi" {
		return "", false
	}
	if len(call.Args) >= 2 {
		if path, ok := httpConfigBasePath(file, call.Args[1]); ok {
			return path, true
		}
	}
	return "", true
}

func httpConfigBasePath(file rulekit.GoFile, expr ast.Expr) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) < 2 {
		return "", false
	}
	name := callName(file, nil, call.Fun)
	if name == "" || (!strings.Contains(strings.ToLower(name), "config") && !strings.Contains(strings.ToLower(name), "http")) {
		return "", false
	}
	return literalString(call.Args[1]), true
}

func callName(file rulekit.GoFile, receivers map[string]string, expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		left := selectorReceiverName(file, receivers, typed.X)
		if left == "" {
			return typed.Sel.Name
		}
		return left + "." + typed.Sel.Name
	}
	return ""
}

func selectorReceiverName(file rulekit.GoFile, receivers map[string]string, expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		if receiverType := receivers[typed.Name]; receiverType != "" {
			return receiverType
		}
		return typed.Name
	case *ast.CompositeLit:
		return exprTypeName(typed.Type)
	case *ast.StarExpr:
		return selectorReceiverName(file, receivers, typed.X)
	}
	return exprString(file.Fset, expr)
}

func registerMethodCall(callee string) bool {
	name := callee
	if index := strings.LastIndex(name, "."); index >= 0 {
		name = name[index+1:]
	}
	return strings.HasPrefix(name, "register")
}

func apiMountKey(register string, file string) string {
	return file + "\x00" + register
}

func httpMethodName(file rulekit.GoFile, expr ast.Expr) string {
	if value := literalString(expr); value != "" {
		return value
	}
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return exprString(file.Fset, expr)
	}
	if !strings.HasPrefix(selector.Sel.Name, "Method") {
		return exprString(file.Fset, expr)
	}
	return strings.ToUpper(strings.TrimPrefix(selector.Sel.Name, "Method"))
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

func literalStringSlice(expr ast.Expr) []string {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	var values []string
	for _, element := range literal.Elts {
		value := literalString(element)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func joinAPIPath(prefix string, path string) string {
	if prefix == "" {
		if path == "" {
			return ""
		}
		if strings.HasPrefix(path, "/") {
			return path
		}
		return "/" + path
	}
	if path == "" || path == "/" {
		return prefix
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
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
