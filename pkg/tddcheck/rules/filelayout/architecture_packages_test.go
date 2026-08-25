package filelayout

import (
	"path/filepath"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func TestViolationsRejectNestedStrictLayerPackage(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `service layer must be a direct package`)
}

func TestViolationsRejectWrongStrictLayerPackageName(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package application
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `service layer package must be named "service"`)
}

func TestViolationsAllowCustomLayerWithoutPackageContract(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/adapter/device.service.go": `package device
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
	})
	config := rulekit.Config{
		LayerDirs:          []string{"adapter"},
		LayerFileNameModes: map[string]string{"adapter": rulekit.FileNameModeQualifiedKind},
		LayerKindPolicies: map[string]map[string]string{
			"adapter": {"service": "free"},
		},
		ArchitectureNamespaces: map[string][]string{"adapter": {}},
	}

	violations, err := checkRoot(filepath.Join(root, "internal"), config)
	if err != nil {
		t.Fatal(err)
	}
	assertNoViolationContains(t, violations, "package must be named")
}
