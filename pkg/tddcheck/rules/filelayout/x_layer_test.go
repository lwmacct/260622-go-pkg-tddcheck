package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRejectsArchitectureNamespaceInWrongLayer(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/x.store.support.go": `package service
type StoreConfig struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `architecture namespace "store" is not allowed in service`)
}

func TestViolationsRejectsRemovedFileKinds(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.models.go": `package service
type Device struct{}
`,
		"internal/service/device.writes.go": `package service
type DeviceWrite struct{}
`,
		"internal/repository/device.database.go": `package repository
func Open() {}
`,
		"internal/repository/device.models.go": `package repository
type Device struct{}
`,
		"internal/service/device.model.go": `package service
type Device struct{}
`,
		"internal/service/device.constants.go": `package service
const maxName = 128
`,
		"internal/service/device.errors.go": `package service
func WrapDeviceError() {}
`,
		"internal/service/device.utils.go": `package service
func utilDevice() {}
`,
		"internal/service/device.validation.go": `package service
func validateDevice() {}
`,
		"internal/repository/device.model.go": `package repository
type Device struct{}
`,
		"internal/repository/device.constants.go": `package repository
const maxName = 128
`,
		"internal/repository/device.errors.go": `package repository
var ErrDevice = errors.New("device")
`,
		"internal/repository/device.utils.go": `package repository
func utilDevice() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `service file kind "models" is not allowed`)
	assertViolationContains(t, violations, `service file kind "writes" is not allowed`)
	assertViolationContains(t, violations, `repository file kind "database" is not allowed`)
	assertViolationContains(t, violations, `repository file kind "models" is not allowed`)
	assertViolationContains(t, violations, `service file kind "model" is not allowed`)
	assertViolationContains(t, violations, `service file kind "constants" is not allowed`)
	assertViolationContains(t, violations, `service file kind "errors" is not allowed`)
	assertViolationContains(t, violations, `service file kind "utils" is not allowed`)
	assertViolationContains(t, violations, `service file kind "validation" is not allowed`)
	assertViolationContains(t, violations, `repository file kind "model" is not allowed`)
	assertViolationContains(t, violations, `repository file kind "constants" is not allowed`)
	assertViolationContains(t, violations, `repository file kind "errors" is not allowed`)
	assertViolationContains(t, violations, `repository file kind "utils" is not allowed`)
}
