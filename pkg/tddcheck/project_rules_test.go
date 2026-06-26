package tddcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectAnalyzePassesValidHSRLayout(t *testing.T) {
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
func NewStore() *Store { return &Store{} }
`,
	})

	analysis, err := Project{Root: filepath.Join(root, "internal")}.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.Passed() {
		t.Fatal(analysis.Text())
	}
}

func TestProjectAnalyzePassesArchitectureEndpointWithResourceHandlers(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/x_http.endpoint.go": `package handler
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
		"internal/handler/x_http.support.go": `package handler
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
func NewStore() *Store { return &Store{} }
`,
	})

	analysis, err := Project{Root: filepath.Join(root, "internal")}.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.Passed() {
		t.Fatal(analysis.Text())
	}
}

func TestProjectAnalyzeReportsViolations(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.go": `package handler
import _ "example.com/app/internal/repository"
`,
	})

	analysis, err := Project{Root: filepath.Join(root, "internal")}.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Passed() {
		t.Fatal("expected failure")
	}
	text := analysis.Text()
	if !strings.Contains(text, "[filelayout]") {
		t.Fatalf("expected filelayout violation, got:\n%s", text)
	}
	if !strings.Contains(text, "[layerdeps]") {
		t.Fatalf("expected layerdeps violation, got:\n%s", text)
	}
}

func TestProjectAnalyzeSupportsDependencyOnlyLayers(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/adapter/wsworkspace/endpoint.go": `package wsworkspace
import _ "example.com/app/internal/service"
`,
		"internal/runtime/nodepool/pool.go": `package nodepool
import _ "example.com/app/internal/adapter/wsworkspace"
`,
	})

	analysis, err := Project{
		Root: filepath.Join(root, "internal"),
		Config: Config{
			LayerDirs:           []string{"adapter"},
			DependencyLayerDirs: []string{"adapter", "runtime", "service"},
			LayerFileNameModes: map[string]string{
				"adapter": FileNameModePackageKind,
			},
			LayerFileKinds: map[string][]string{
				"adapter": {"endpoint"},
			},
			ArchitectureScopes: map[string][]string{},
			LayerRules: []LayerDependencyRule{
				{SourceLayer: "runtime", TargetLayer: "adapter", Message: "runtime must not import adapter"},
			},
		},
	}.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Passed() {
		t.Fatal("expected failure")
	}
	text := analysis.Text()
	if !strings.Contains(text, "runtime must not import adapter") {
		t.Fatalf("expected runtime dependency violation, got:\n%s", text)
	}
	if strings.Contains(text, "runtime file must use") {
		t.Fatalf("runtime should not be checked by filelayout, got:\n%s", text)
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
