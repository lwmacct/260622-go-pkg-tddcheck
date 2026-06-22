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
		"internal/handler/x_router.handler.go": `package handler
func registerRoutes() {}
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
		"internal/service/device.validation.go": `package service
func validateDevice() {}
func normalizeDevice() {}
`,
		"internal/service/x_batch.service.go": `package service
func NewBatchService() {}
`,
		"internal/service/device_group.service.go": `package service
type DeviceGroupService struct{}
func NewDeviceGroupService() *DeviceGroupService { return &DeviceGroupService{} }
func (s *DeviceGroupService) ListGroups() {}
`,
		"internal/service/x_shared.models.go": `package service
type DeviceRow struct{}
`,
		"internal/service/x_shared.mapper.go": `package service
func ToRepositoryDevice() {}
`,
		"internal/repository/device.model.go": `package repository
type DeviceRow struct{}
`,
		"internal/repository/device.store.go": `package repository
func (s *Store) ListDevices() {}
`,
		"internal/repository/x_store.repository.go": `package repository
type Store struct{}
func NewStore() *Store { return &Store{} }
`,
		"internal/repository/x_schema.repository.go": `package repository
func Schema() {}
`,
		"internal/repository/x_shared.models.go": `package repository
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
