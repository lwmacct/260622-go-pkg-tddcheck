package layerdeps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestViolationsRejectsForbiddenHSRImports(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.handler.go": `package handler
import _ "example.com/app/internal/repository"
`,
		"internal/service/device.service.go": `package service
import _ "example.com/app/internal/repository"
`,
		"internal/repository/device.store.go": `package repository
import _ "example.com/app/internal/service"
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %#v", violations)
	}
	assertLayerViolationContains(t, violations, "handler must not import repository")
	assertLayerViolationContains(t, violations, "repository must not import service")
}

func fixture(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.26.4\n")
	for name, content := range files {
		writeFile(t, root, name, content)
	}
	return root
}

func writeFile(t *testing.T, root string, name string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertLayerViolationContains(t *testing.T, violations []Violation, needle string) {
	t.Helper()

	for _, violation := range violations {
		if strings.Contains(violation.Message, needle) {
			return
		}
	}
	t.Fatalf("expected violation containing %q, got %#v", needle, violations)
}
