package tddcheck

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

// Index is the structured architecture discovered during project analysis.
type Index struct {
	Root        string            `json:"root"`
	ModulePath  string            `json:"modulePath"`
	Handlers    []HandlerIndex    `json:"handlers"`
	Services    []ServiceIndex    `json:"services"`
	Stores      []StoreIndex      `json:"stores"`
	Tables      []TableIndex      `json:"tables"`
	Projections []ProjectionIndex `json:"projections,omitempty"`

	projectRoot string
}

// HandlerIndex describes a handler type, its registrations, and methods.
type HandlerIndex struct {
	Identity  FileIdentity  `json:"identity"`
	Type      string        `json:"type,omitempty"`
	Registers []string      `json:"registers,omitempty"`
	File      string        `json:"file"`
	Methods   []MethodIndex `json:"methods,omitempty"`
}

// ServiceIndex describes a service type, constructor, methods, and declared
// dependencies.
type ServiceIndex struct {
	Identity     FileIdentity  `json:"identity"`
	Type         string        `json:"type"`
	Constructor  string        `json:"constructor,omitempty"`
	File         string        `json:"file"`
	Methods      []MethodIndex `json:"methods,omitempty"`
	Dependencies []string      `json:"dependencies,omitempty"`
}

// StoreIndex describes the methods associated with a repository store scope.
type StoreIndex struct {
	Identity FileIdentity  `json:"identity"`
	File     string        `json:"file"`
	Methods  []MethodIndex `json:"methods,omitempty"`
}

// MethodIndex identifies a discovered method and its source line.
type MethodIndex struct {
	Name string `json:"name"`
	Line int    `json:"line,omitempty"`
}

// TableIndex describes a database table model discovered from a schema file.
type TableIndex struct {
	Identity    FileIdentity `json:"identity"`
	Model       string       `json:"model"`
	Table       string       `json:"table"`
	Alias       string       `json:"alias,omitempty"`
	File        string       `json:"file"`
	Fields      []FieldIndex `json:"fields,omitempty"`
	ForeignKeys []string     `json:"foreignKeys,omitempty"`
}

// ProjectionIndex describes a schema projection and the models it extends.
type ProjectionIndex struct {
	Identity FileIdentity `json:"identity"`
	Model    string       `json:"model"`
	File     string       `json:"file"`
	Extends  []string     `json:"extends,omitempty"`
	Fields   []FieldIndex `json:"fields,omitempty"`
}

// FieldIndex describes a Go model field and its database column attributes.
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

// Text renders a human-readable architecture index.
func (i Index) Text() string {
	var builder strings.Builder
	_, _ = fmt.Fprintf(&builder, "tddcheck index: %s\n", i.ModulePath)
	_, _ = fmt.Fprintf(&builder, "root: %s\n", i.Root)
	writeHandlerText(&builder, i.Handlers)
	writeServiceText(&builder, i.Services)
	writeStoreText(&builder, i.Stores)
	writeTableText(&builder, i.Tables)
	writeProjectionText(&builder, i.Projections)
	return strings.TrimRight(builder.String(), "\n")
}

func writeHandlerText(builder *strings.Builder, handlers []HandlerIndex) {
	builder.WriteString("\nhandlers:\n")
	if len(handlers) == 0 {
		builder.WriteString("- none\n")
		return
	}
	for _, handler := range handlers {
		_, _ = fmt.Fprintf(builder, "- %s (%s) %s\n", handler.Type, identityLabel(handler.Identity), handler.File)
		if len(handler.Registers) > 0 {
			_, _ = fmt.Fprintf(builder, "  registers: %s\n", strings.Join(handler.Registers, ", "))
		}
		writeMethodNames(builder, "  methods", handler.Methods)
	}
}

func writeProjectionText(builder *strings.Builder, projections []ProjectionIndex) {
	builder.WriteString("\nprojections:\n")
	if len(projections) == 0 {
		builder.WriteString("- none\n")
		return
	}
	for _, projection := range projections {
		_, _ = fmt.Fprintf(builder, "- %s (%s) %s\n", projection.Model, identityLabel(projection.Identity), projection.File)
		if len(projection.Extends) > 0 {
			_, _ = fmt.Fprintf(builder, "  extends: %s\n", strings.Join(projection.Extends, ", "))
		}
		for _, field := range projection.Fields {
			options := fieldOptions(field)
			if options != "" {
				options = " " + options
			}
			_, _ = fmt.Fprintf(builder, "  - %s %s `%s`%s\n", field.Name, field.GoType, field.Column, options)
		}
	}
}

func writeServiceText(builder *strings.Builder, services []ServiceIndex) {
	builder.WriteString("\nservices:\n")
	if len(services) == 0 {
		builder.WriteString("- none\n")
		return
	}
	for _, service := range services {
		_, _ = fmt.Fprintf(builder, "- %s (%s) %s\n", service.Type, identityLabel(service.Identity), service.File)
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
		_, _ = fmt.Fprintf(builder, "- %s %s\n", identityLabel(store.Identity), store.File)
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
		_, _ = fmt.Fprintf(builder, "- %s (%s, %s) %s\n", name, table.Model, identityLabel(table.Identity), table.File)
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

func indexFromSnapshot(snapshot *rulekit.Snapshot) Index {
	result := Index{
		Root:        snapshot.DisplayPath(snapshot.Root),
		ModulePath:  snapshot.ModulePath,
		projectRoot: snapshot.ProjectRoot,
	}
	for _, file := range snapshot.Files {
		if file.IsTest || !file.IdentityOK {
			continue
		}
		switch file.Layer {
		case "handler":
			if file.Identity.Kind == "handler" {
				result.Handlers = append(result.Handlers, handlerIndexesFromFile(snapshot, file)...)
			}
		case "service":
			if file.Identity.Kind == "service" {
				result.Services = append(result.Services, serviceIndexesFromFile(snapshot, file)...)
			}
		case "repository":
			if file.Identity.Kind == "store" {
				result.Stores = append(result.Stores, storeIndexesFromFile(snapshot, file)...)
			}
			if file.Identity.Kind == "schema" {
				tables, projections := schemaIndexesFromFile(snapshot, file)
				result.Tables = append(result.Tables, tables...)
				result.Projections = append(result.Projections, projections...)
			}
		}
	}
	sortIndex(result)
	return result
}

func sortIndex(index Index) {
	slices.SortFunc(index.Handlers, func(a, b HandlerIndex) int {
		return strings.Compare(identitySortKey(a.Identity)+"\x00"+a.Type+"\x00"+a.File, identitySortKey(b.Identity)+"\x00"+b.Type+"\x00"+b.File)
	})
	slices.SortFunc(index.Services, func(a, b ServiceIndex) int {
		return strings.Compare(identitySortKey(a.Identity)+"\x00"+a.Type+"\x00"+a.File, identitySortKey(b.Identity)+"\x00"+b.Type+"\x00"+b.File)
	})
	slices.SortFunc(index.Stores, func(a, b StoreIndex) int {
		return strings.Compare(identitySortKey(a.Identity)+"\x00"+a.File, identitySortKey(b.Identity)+"\x00"+b.File)
	})
	slices.SortFunc(index.Tables, func(a, b TableIndex) int {
		keyA := a.Table + "\x00" + a.Model + "\x00" + identitySortKey(a.Identity) + "\x00" + a.File
		keyB := b.Table + "\x00" + b.Model + "\x00" + identitySortKey(b.Identity) + "\x00" + b.File
		return strings.Compare(keyA, keyB)
	})
	slices.SortFunc(index.Projections, func(a, b ProjectionIndex) int {
		keyA := a.Model + "\x00" + identitySortKey(a.Identity) + "\x00" + a.File
		keyB := b.Model + "\x00" + identitySortKey(b.Identity) + "\x00" + b.File
		return strings.Compare(keyA, keyB)
	})
}

func identityLabel(identity FileIdentity) string {
	if identity.Namespace != "" {
		return "x." + identity.Namespace
	}
	if identity.Subject != "" {
		return identity.Subject
	}
	return identity.Kind
}

func identitySortKey(identity FileIdentity) string {
	return identity.Layer + "\x00" + identity.Subject + "\x00" + identity.Namespace + "\x00" + identity.Kind
}
