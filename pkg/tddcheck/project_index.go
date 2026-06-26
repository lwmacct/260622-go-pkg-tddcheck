package tddcheck

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

type Index struct {
	Root       string         `json:"root"`
	ModulePath string         `json:"modulePath"`
	APIs       []APIIndex     `json:"apis"`
	Handlers   []HandlerIndex `json:"handlers"`
	Services   []ServiceIndex `json:"services"`
	Stores     []StoreIndex   `json:"stores"`
	Tables     []TableIndex   `json:"tables"`

	projectRoot string
}

type HandlerIndex struct {
	Scope    string        `json:"scope"`
	Type     string        `json:"type,omitempty"`
	Register string        `json:"register,omitempty"`
	File     string        `json:"file"`
	Methods  []MethodIndex `json:"methods,omitempty"`
}

type APIIndex struct {
	Scope       string   `json:"scope"`
	Method      string   `json:"method,omitempty"`
	Path        string   `json:"path,omitempty"`
	OperationID string   `json:"operationId,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Handler     string   `json:"handler,omitempty"`
	Register    string   `json:"register,omitempty"`
	File        string   `json:"file"`
	Line        int      `json:"line,omitempty"`
}

type ServiceIndex struct {
	Scope        string        `json:"scope"`
	Type         string        `json:"type"`
	Constructor  string        `json:"constructor,omitempty"`
	File         string        `json:"file"`
	Methods      []MethodIndex `json:"methods,omitempty"`
	Dependencies []string      `json:"dependencies,omitempty"`
}

type StoreIndex struct {
	Scope   string        `json:"scope"`
	File    string        `json:"file"`
	Methods []MethodIndex `json:"methods,omitempty"`
}

type MethodIndex struct {
	Name string `json:"name"`
	Line int    `json:"line,omitempty"`
}

type TableIndex struct {
	Scope       string       `json:"scope"`
	Model       string       `json:"model"`
	Table       string       `json:"table"`
	Alias       string       `json:"alias,omitempty"`
	File        string       `json:"file"`
	Fields      []FieldIndex `json:"fields,omitempty"`
	ForeignKeys []string     `json:"foreignKeys,omitempty"`
}

type FieldIndex struct {
	Name          string `json:"name"`
	Column        string `json:"column"`
	GoType        string `json:"goType"`
	PrimaryKey    bool   `json:"primaryKey,omitempty"`
	AutoIncrement bool   `json:"autoIncrement,omitempty"`
	NotNull       bool   `json:"notNull,omitempty"`
	Nullable      bool   `json:"nullable,omitempty"`
	Unique        string `json:"unique,omitempty"`
}

func (i Index) Text() string {
	var builder strings.Builder
	_, _ = fmt.Fprintf(&builder, "tddcheck index: %s\n", i.ModulePath)
	_, _ = fmt.Fprintf(&builder, "root: %s\n", i.Root)
	writeAPIText(&builder, i.APIs)
	writeHandlerText(&builder, i.Handlers)
	writeServiceText(&builder, i.Services)
	writeStoreText(&builder, i.Stores)
	writeTableText(&builder, i.Tables)
	return strings.TrimRight(builder.String(), "\n")
}

func writeAPIText(builder *strings.Builder, apis []APIIndex) {
	builder.WriteString("\napis:\n")
	if len(apis) == 0 {
		builder.WriteString("- none\n")
		return
	}
	for _, api := range apis {
		_, _ = fmt.Fprintf(builder, "- %s %s (%s) %s\n", valueOrUnknown(api.Method), valueOrUnknown(api.Path), api.Scope, api.File)
		if api.OperationID != "" {
			_, _ = fmt.Fprintf(builder, "  operation: %s\n", api.OperationID)
		}
		if api.Handler != "" {
			_, _ = fmt.Fprintf(builder, "  handler: %s\n", api.Handler)
		}
		if len(api.Tags) > 0 {
			_, _ = fmt.Fprintf(builder, "  tags: %s\n", strings.Join(api.Tags, ", "))
		}
	}
}

func writeHandlerText(builder *strings.Builder, handlers []HandlerIndex) {
	builder.WriteString("\nhandlers:\n")
	if len(handlers) == 0 {
		builder.WriteString("- none\n")
		return
	}
	for _, handler := range handlers {
		_, _ = fmt.Fprintf(builder, "- %s (%s) %s\n", handler.Type, handler.Scope, handler.File)
		if handler.Register != "" {
			_, _ = fmt.Fprintf(builder, "  register: %s\n", handler.Register)
		}
		writeMethodNames(builder, "  methods", handler.Methods)
	}
}

func writeServiceText(builder *strings.Builder, services []ServiceIndex) {
	builder.WriteString("\nservices:\n")
	if len(services) == 0 {
		builder.WriteString("- none\n")
		return
	}
	for _, service := range services {
		_, _ = fmt.Fprintf(builder, "- %s (%s) %s\n", service.Type, service.Scope, service.File)
		if service.Constructor != "" {
			_, _ = fmt.Fprintf(builder, "  constructor: %s\n", service.Constructor)
		}
		if len(service.Dependencies) > 0 {
			_, _ = fmt.Fprintf(builder, "  dependencies: %s\n", strings.Join(service.Dependencies, ", "))
		}
		writeMethodNames(builder, "  methods", service.Methods)
	}
}

func writeStoreText(builder *strings.Builder, stores []StoreIndex) {
	builder.WriteString("\nstores:\n")
	if len(stores) == 0 {
		builder.WriteString("- none\n")
		return
	}
	for _, store := range stores {
		_, _ = fmt.Fprintf(builder, "- %s %s\n", store.Scope, store.File)
		writeMethodNames(builder, "  methods", store.Methods)
	}
}

func writeTableText(builder *strings.Builder, tables []TableIndex) {
	builder.WriteString("\ntables:\n")
	if len(tables) == 0 {
		builder.WriteString("- none\n")
		return
	}
	for _, table := range tables {
		name := table.Table
		if name == "" {
			name = "(unknown)"
		}
		_, _ = fmt.Fprintf(builder, "- %s (%s, %s) %s\n", name, table.Model, table.Scope, table.File)
		if table.Alias != "" {
			_, _ = fmt.Fprintf(builder, "  alias: %s\n", table.Alias)
		}
		for _, field := range table.Fields {
			options := fieldOptions(field)
			if options != "" {
				options = " " + options
			}
			_, _ = fmt.Fprintf(builder, "  - %s %s `%s`%s\n", field.Name, field.GoType, field.Column, options)
		}
		for _, foreignKey := range table.ForeignKeys {
			_, _ = fmt.Fprintf(builder, "  foreign key: %s\n", foreignKey)
		}
	}
}

func writeMethodNames(builder *strings.Builder, label string, methods []MethodIndex) {
	if len(methods) == 0 {
		return
	}
	names := make([]string, 0, len(methods))
	for _, method := range methods {
		names = append(names, method.Name)
	}
	_, _ = fmt.Fprintf(builder, "%s: %s\n", label, strings.Join(names, ", "))
}

func fieldOptions(field FieldIndex) string {
	var options []string
	if field.PrimaryKey {
		options = append(options, "pk")
	}
	if field.AutoIncrement {
		options = append(options, "autoincrement")
	}
	if field.NotNull {
		options = append(options, "notnull")
	}
	if field.Nullable {
		options = append(options, "nullable")
	}
	if field.Unique != "" {
		if field.Unique == "true" {
			options = append(options, "unique")
		} else {
			options = append(options, "unique:"+field.Unique)
		}
	}
	if len(options) == 0 {
		return ""
	}
	return "[" + strings.Join(options, ", ") + "]"
}

func indexFromContext(context *rulekit.Context) Index {
	result := Index{
		Root:        context.DisplayPath(context.Root),
		ModulePath:  context.ModulePath,
		projectRoot: context.ProjectRoot(),
	}
	for _, file := range context.Files {
		if rulekit.FreeFile(file.Base) {
			continue
		}
		switch file.Layer {
		case "handler":
			result.APIs = append(result.APIs, apiIndexesFromFile(context, file)...)
			if strings.HasSuffix(file.Base, ".handler.go") {
				result.Handlers = append(result.Handlers, handlerIndexesFromFile(context, file)...)
			}
		case "service":
			if strings.HasSuffix(file.Base, ".service.go") {
				result.Services = append(result.Services, serviceIndexesFromFile(context, file)...)
			}
		case "repository":
			if strings.HasSuffix(file.Base, ".store.go") {
				result.Stores = append(result.Stores, storeIndexesFromFile(context, file)...)
			}
			if strings.HasSuffix(file.Base, ".schema.go") {
				result.Tables = append(result.Tables, tableIndexesFromFile(context, file)...)
			}
		}
	}
	sortIndex(result)
	return result
}

func sortIndex(index Index) {
	slices.SortFunc(index.APIs, func(a, b APIIndex) int {
		if a.Path != b.Path {
			return strings.Compare(a.Path, b.Path)
		}
		if a.Method != b.Method {
			return strings.Compare(a.Method, b.Method)
		}
		return strings.Compare(a.OperationID, b.OperationID)
	})
	slices.SortFunc(index.Handlers, func(a, b HandlerIndex) int {
		return strings.Compare(a.Scope, b.Scope)
	})
	slices.SortFunc(index.Services, func(a, b ServiceIndex) int {
		return strings.Compare(a.Type, b.Type)
	})
	slices.SortFunc(index.Stores, func(a, b StoreIndex) int {
		return strings.Compare(a.Scope, b.Scope)
	})
	slices.SortFunc(index.Tables, func(a, b TableIndex) int {
		if a.Table != b.Table {
			return strings.Compare(a.Table, b.Table)
		}
		return strings.Compare(a.Model, b.Model)
	})
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "(unknown)"
	}
	return value
}
