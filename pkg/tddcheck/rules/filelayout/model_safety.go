package filelayout

import "strings"

func forbiddenServiceModelImport(importPath string) bool {
	return strings.Contains(importPath, "/internal/handler") ||
		oneOf(
			importPath,
			"net/http",
			"github.com/danielgtaylor/huma/v2",
			"github.com/uptrace/bun",
			"gorm.io/gorm",
		)
}

func forbiddenServiceModelType(name string) bool {
	return strings.HasSuffix(name, "DTO") ||
		strings.HasSuffix(name, "DTOs") ||
		strings.HasSuffix(name, "Request") ||
		strings.HasSuffix(name, "Response") ||
		strings.HasSuffix(name, "Result") ||
		strings.HasSuffix(name, "Item")
}

func forbiddenRepositoryModelImport(importPath string) bool {
	return strings.Contains(importPath, "/internal/handler") ||
		strings.Contains(importPath, "/internal/service") ||
		oneOf(
			importPath,
			"net/http",
			"github.com/danielgtaylor/huma/v2",
		)
}
