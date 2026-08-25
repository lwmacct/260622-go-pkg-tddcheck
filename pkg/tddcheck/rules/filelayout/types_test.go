package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsAcceptsTypesFiles(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
		"internal/service/device.types.go": `package service
import "errors"
const DeviceStatusActive = "active"
var ErrDeviceUnavailable = errors.New("device unavailable")
type Device struct{}
type DeviceError struct{}
func (DeviceError) Error() string { return "device error" }
func (DeviceError) Unwrap() error { return nil }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestViolationsRejectsInvalidTypesDeclarations(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
		"internal/service/device.types.go": `package service
type Device struct{}
func BuildDevice() {}
func (Device) Unwrap() error { return nil }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "types files must only declare types, consts, Err* vars, and Error/Unwrap methods")
	assertViolationContains(t, violations, "types Error/Unwrap methods must use receivers that implement error")
}
