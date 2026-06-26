package tddcheck

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

type ProjectMap struct {
	Root       string       `json:"root"`
	ModulePath string       `json:"modulePath"`
	Services   []ServiceMap `json:"services"`
	Tables     []TableMap   `json:"tables"`

	projectRoot string
}

type ServiceMap struct {
	Scope       string      `json:"scope"`
	Name        string      `json:"name"`
	Constructor string      `json:"constructor,omitempty"`
	File        string      `json:"file"`
	Methods     []MethodMap `json:"methods,omitempty"`
}

type MethodMap struct {
	Name string `json:"name"`
	Line int    `json:"line,omitempty"`
}

type TableMap struct {
	Scope       string     `json:"scope"`
	Model       string     `json:"model"`
	Table       string     `json:"table"`
	Alias       string     `json:"alias,omitempty"`
	File        string     `json:"file"`
	Fields      []FieldMap `json:"fields,omitempty"`
	ForeignKeys []string   `json:"foreignKeys,omitempty"`
}

type FieldMap struct {
	Name          string `json:"name"`
	Column        string `json:"column"`
	GoType        string `json:"goType"`
	PrimaryKey    bool   `json:"primaryKey,omitempty"`
	AutoIncrement bool   `json:"autoIncrement,omitempty"`
	NotNull       bool   `json:"notNull,omitempty"`
	Nullable      bool   `json:"nullable,omitempty"`
	Unique        string `json:"unique,omitempty"`
}

func (m ProjectMap) Text() string {
	var builder strings.Builder
	_, _ = fmt.Fprintf(&builder, "tddcheck map: %s\n", m.ModulePath)
	_, _ = fmt.Fprintf(&builder, "root: %s\n", m.Root)

	builder.WriteString("\nservices:\n")
	if len(m.Services) == 0 {
		builder.WriteString("- none\n")
	} else {
		for _, service := range m.Services {
			_, _ = fmt.Fprintf(&builder, "- %s (%s) %s\n", service.Name, service.Scope, service.File)
			if service.Constructor != "" {
				_, _ = fmt.Fprintf(&builder, "  constructor: %s\n", service.Constructor)
			}
			if len(service.Methods) > 0 {
				names := make([]string, 0, len(service.Methods))
				for _, method := range service.Methods {
					names = append(names, method.Name)
				}
				_, _ = fmt.Fprintf(&builder, "  methods: %s\n", strings.Join(names, ", "))
			}
		}
	}

	builder.WriteString("\ntables:\n")
	if len(m.Tables) == 0 {
		builder.WriteString("- none\n")
	} else {
		for _, table := range m.Tables {
			name := table.Table
			if name == "" {
				name = "(unknown)"
			}
			_, _ = fmt.Fprintf(&builder, "- %s (%s, %s) %s\n", name, table.Model, table.Scope, table.File)
			if table.Alias != "" {
				_, _ = fmt.Fprintf(&builder, "  alias: %s\n", table.Alias)
			}
			for _, field := range table.Fields {
				options := fieldOptions(field)
				if options != "" {
					options = " " + options
				}
				_, _ = fmt.Fprintf(&builder, "  - %s %s `%s`%s\n", field.Name, field.GoType, field.Column, options)
			}
			for _, foreignKey := range table.ForeignKeys {
				_, _ = fmt.Fprintf(&builder, "  foreign key: %s\n", foreignKey)
			}
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}

func fieldOptions(field FieldMap) string {
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

func mapFromContext(context *rulekit.Context) ProjectMap {
	result := ProjectMap{
		Root:        context.DisplayPath(context.Root),
		ModulePath:  context.ModulePath,
		projectRoot: context.ProjectRoot(),
	}
	for _, file := range context.Files {
		if rulekit.FreeFile(file.Base) {
			continue
		}
		switch file.Layer {
		case "service":
			if strings.HasSuffix(file.Base, ".service.go") {
				result.Services = append(result.Services, serviceMapFromFile(context, file)...)
			}
		case "repository":
			if strings.HasSuffix(file.Base, ".schema.go") {
				result.Tables = append(result.Tables, tableMapsFromFile(context, file)...)
			}
		}
	}
	slices.SortFunc(result.Services, func(a, b ServiceMap) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(result.Tables, func(a, b TableMap) int {
		if a.Table != b.Table {
			return strings.Compare(a.Table, b.Table)
		}
		return strings.Compare(a.Model, b.Model)
	})
	return result
}
