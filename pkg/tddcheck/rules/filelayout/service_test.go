package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksServiceContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/devicegroup.service.go": `package service
type DeviceGroupService struct{}
type DeviceGroupHelper struct{}
func NewDeviceGroupService() *DeviceGroupService { return &DeviceGroupService{} }
func NewDeviceGroupHelper() {}
func (s *DeviceGroupHelper) Bad() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `service file for DeviceGroupService must use scope "device_group"`)
	assertViolationContains(t, violations, "service files must only declare DeviceGroupService")
	assertViolationContains(t, violations, "service files must only declare NewDeviceGroupService as a package-level function")
	assertViolationContains(t, violations, "service receiver methods must use DeviceGroupService")
}
