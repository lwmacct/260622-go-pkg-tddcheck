package filelayout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestViolationsAcceptsApprovedDotFileLayout(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.dto.go": `package handler
type DeviceDTO struct{}
`,
		"internal/handler/device.mapper.go": `package handler
func toDeviceDTO() DeviceDTO { return DeviceDTO{} }
`,
		"internal/handler/router.handler.go": `package handler
func registerRoutes() {}
`,
		"internal/service/device.commands.go": `package service
type CreateDeviceRequest struct{}
`,
		"internal/service/device.validation.go": `package service
func validateDevice() {}
func normalizeDevice() {}
`,
		"internal/service/batch.service.go": `package service
func NewBatchService() {}
`,
		"internal/service/repository.models.go": `package service
func fromRepositoryDevice() {}
`,
		"internal/repository/device.model.go": `package repository
type DeviceRow struct{}
`,
		"internal/repository/device.store.go": `package repository
func (s *Store) ListDevices() {}
`,
		"internal/repository/store.repository.go": `package repository
type Store struct{}
func NewStore() *Store { return &Store{} }
`,
		"internal/repository/schema.repository.go": `package repository
func Schema() {}
`,
		"internal/repository/write.models.go": `package repository
type DeviceCreate struct{}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestViolationsRejectsWeakScopeAndLegacyNames(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/default.service.go": `package service
func NewDefaultService() {}
`,
		"internal/service/device_validation.go": `package service
func validateDevice() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "scope \"default\" is too weak")
	assertViolationContains(t, violations, "must use {scope}.{type}.go")
}

func TestViolationsChecksTypeContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.dto.go": `package handler
type Device struct{}
func bad() {}
`,
		"internal/service/device.commands.go": `package service
type DeviceRequest struct{}
func bad() {}
`,
		"internal/service/device.validation.go": `package service
type DeviceValidator struct{}
func checkDevice() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "must end with DTO or DTOs")
	assertViolationContains(t, violations, "dto files must not declare functions")
	assertViolationContains(t, violations, "commands files must only declare types")
	assertViolationContains(t, violations, "validation files must not declare types")
	assertViolationContains(t, violations, "validation functions must be package-level validate* or normalize*")
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

func assertViolationContains(t *testing.T, violations []Violation, needle string) {
	t.Helper()

	for _, violation := range violations {
		if strings.Contains(violation.Message, needle) {
			return
		}
	}
	t.Fatalf("expected violation containing %q, got %#v", needle, violations)
}
