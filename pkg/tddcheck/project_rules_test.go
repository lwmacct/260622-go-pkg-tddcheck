package tddcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectRulesCheckPassesValidHSRLayout(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.handler.go": `package handler
import _ "example.com/app/internal/service"
type deviceHandler struct{}
func RegisterDevices() {}
func (h deviceHandler) list() {}
`,
		"internal/service/device.service.go": `package service
import _ "example.com/app/internal/repository"
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
		"internal/repository/device.store.go": `package repository
import "context"
func (s *Store) ListDevices(ctx context.Context) ([]string, error) { return nil, nil }
`,
		"internal/repository/x_store.repository.go": `package repository
type Store struct{}
`,
	})

	result := ProjectRules{Root: filepath.Join(root, "internal")}.Check()
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if !result.Passed {
		t.Fatal(result.Text())
	}
}

func TestProjectRulesCheckPassesArchitectureEndpointWithResourceHandlers(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/x_api.endpoint.go": `package handler
import _ "example.com/app/internal/service"
type Endpoint struct{}
type Config struct{}
type nodeAccess struct{}
func NewEndpoint(cfg Config) *Endpoint { return &Endpoint{} }
func (e *Endpoint) Handler() {}
func (e *Endpoint) register() {}
func (a nodeAccess) resolve() {}
func currentUser() {}
`,
		"internal/handler/x_api.support.go": `package handler
type ServiceConfig struct{}
type RuntimeConfig struct{}
type AuthConfig struct{}
type RouteConfig struct{}
type EndpointConfig struct{}
type LimitConfig struct{}
`,
		"internal/handler/node_query.handler.go": `package handler
import _ "example.com/app/internal/service"
type nodeQueryHandler struct{}
func RegisterNodeQueries() {}
func (h nodeQueryHandler) list() {}
`,
		"internal/service/node_query.service.go": `package service
import _ "example.com/app/internal/repository"
type NodeQueryService struct{}
func NewNodeQueryService() *NodeQueryService { return &NodeQueryService{} }
`,
		"internal/repository/x_store.repository.go": `package repository
type Store struct{}
`,
	})

	result := ProjectRules{Root: filepath.Join(root, "internal")}.Check()
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if !result.Passed {
		t.Fatal(result.Text())
	}
}

func TestProjectRulesCheckReportsViolations(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.go": `package handler
import _ "example.com/app/internal/repository"
`,
	})

	result := ProjectRules{Root: filepath.Join(root, "internal")}.Check()
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if result.Passed {
		t.Fatal("expected failure")
	}
	text := result.Text()
	if !strings.Contains(text, "[filelayout]") {
		t.Fatalf("expected filelayout violation, got:\n%s", text)
	}
	if !strings.Contains(text, "[layerdeps]") {
		t.Fatalf("expected layerdeps violation, got:\n%s", text)
	}
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
