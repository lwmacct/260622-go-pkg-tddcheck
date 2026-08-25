package filelayout

import "strings"

func forbiddenServiceModelImport(importPath string) bool {
	return strings.Contains(importPath, "/internal/handler")
}

func forbiddenRepositoryModelImport(importPath string) bool {
	return strings.Contains(importPath, "/internal/handler") ||
		strings.Contains(importPath, "/internal/service")
}
