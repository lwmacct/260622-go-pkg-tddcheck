package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsAcceptsApprovedDotFileLayout(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.dto.go": `package handler
type DeviceDTO struct{}
`,
		"internal/handler/device.mapper.go": `package handler
func ToDeviceDTO() DeviceDTO { return DeviceDTO{} }
`,
		"internal/handler/x_shared.dto.go": `package handler
type BodyDTO struct{}
`,
		"internal/handler/x_http.endpoint.go": `package handler
type Endpoint struct{}
type Config struct{}
type ServiceConfig struct{}
type NodeRoutes interface{}
func NewEndpoint(cfg Config) *Endpoint { return &Endpoint{} }
func (e *Endpoint) Handler() {}
func (e *Endpoint) register() {}
`,
		"internal/handler/x_http.middleware.go": `package handler
type Middleware func()
func (e *Endpoint) sessionMiddleware() Middleware { return nil }
func utilMiddleware() {}
`,
		"internal/handler/x_http.support.go": `package handler
type AuthConfig struct{}
type EndpointRoutes interface{}
func currentUser() {}
func IsEnabled() bool { return true }
`,
		"internal/handler/x_http.context.go": `package handler
import "context"
type requestKey struct{}
func ContextWithRequest(ctx context.Context) context.Context { return ctx }
func RequestFromContext(ctx context.Context) bool { return true }
`,
		"internal/handler/device_group.handler.go": `package handler
type deviceGroupHandler struct{}
func RegisterDeviceGroups() {}
func RegisterPublicDeviceGroups() {}
func (h deviceGroupHandler) list() {}
`,
		"internal/service/device.commands.go": `package service
type CreateDeviceRequest struct{}
type BatchDeviceResponse struct{}
type BatchDeviceItemResult struct{}
type BatchUpdateDeviceItem struct{}
`,
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
		"internal/service/device.support.go": `package service
func validateDevice() {}
func normalizeDevice() {}
`,
		"internal/service/device_group.service.go": `package service
type DeviceGroupService struct{}
func NewDeviceGroupService() *DeviceGroupService { return &DeviceGroupService{} }
func (s *DeviceGroupService) ListGroups() {}
`,
		"internal/service/x_shared.support.go": `package service
type DeviceRow struct{}
`,
		"internal/service/x_shared.mapper.go": `package service
func ToRepositoryDevice() {}
`,
		"internal/repository/device.support.go": `package repository
type DeviceRow struct{}
`,
		"internal/repository/device.store.go": `package repository
import "context"
func (s *Store) ListDevices(ctx context.Context) ([]string, error) { return nil, nil }
`,
		"internal/repository/x_store.repository.go": `package repository
type Store struct{}
func NewStore() *Store { return &Store{} }
`,
		"internal/repository/x_shared.support.go": `package repository
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

func TestViolationsRejectsInvalidArchitectureEndpointReceiver(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/x_http.endpoint.go": `package handler
type Endpoint struct{}
type Other struct{}
func (o *Other) Handler() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "architecture endpoint receiver methods must use Endpoint or private adapter types")
}

func TestViolationsRequiresArchitectureEndpointEntrypoint(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/x_http.endpoint.go": `package handler
type Config struct{}
func helper() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "architecture endpoint files must declare Endpoint struct")
	assertViolationContains(t, violations, "architecture endpoint files must declare NewEndpoint")
}

func TestViolationsRejectsServiceSubjectWithoutServiceFile(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/node_ws_tunnel.commands.go": `package service
type ValidateNodeWSTunnelTargetRequest struct{}
`,
		"internal/service/node_ws_tunnel.support.go": `package service
func AsNodeWSTunnelTargetValidation() error { return nil }
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `service subject "node_ws_tunnel" must declare node_ws_tunnel.service.go with NewNodeWSTunnelService`)
}

func TestViolationsAllowsSharedServiceScopeWithoutServiceFile(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/x_shared.support.go": `package service
type SharedConfig struct{}
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

func TestViolationsAllowsFreeFileInAnyLayer(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/x_free.go": `package handler
import _ "example.com/app/internal/repository"
type AnythingDTO struct { Name string ` + "`json:\"name\"`" + ` }
var BadName = 1
func BuildWhatever() {}
`,
		"internal/service/x_free.go": `package service
import (
	"net/http"
	_ "example.com/app/internal/handler"
)
type DeviceDTO struct { Name string ` + "`json:\"name\"`" + ` }
func (d DeviceDTO) Handler(w http.ResponseWriter, r *http.Request) {}
`,
		"internal/repository/x_free.go": `package repository
import _ "example.com/app/internal/service"
type Free struct{}
func BuildWhatever() {}
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
