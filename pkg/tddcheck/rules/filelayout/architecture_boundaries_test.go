package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRejectDuplicateQualifiedIdentity(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/a/device.service.go": `package a
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
		"internal/service/b/device.service.go": `package b
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "file identity device.service.go")
}

func TestViolationsRejectPublicRepositoryPersistenceTypes(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.support.go": `package repository
type DeviceRow struct{}
type DeviceCreate struct{}
`,
		"internal/service/device.service.go": `package service
import "example.com/app/internal/repository"
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
func (s *DeviceService) Ingest() (*repository.DeviceRow, error) { return nil, nil }
func (s *DeviceService) create(input repository.DeviceCreate) {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "service public method Ingest must not expose repository type *repository.DeviceRow")
	assertNoViolationContains(t, violations, "public method create")
}
