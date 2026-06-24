package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRejectsArchitectureScopeInWrongLayer(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/x_store.support.go": `package service
type StoreConfig struct{}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `architecture scope "x_store" is not allowed in service`)
}

func TestViolationsRejectsRemovedFileTypes(t *testing.T) {
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

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `service file type "models" is not allowed`)
	assertViolationContains(t, violations, `service file type "writes" is not allowed`)
	assertViolationContains(t, violations, `repository file type "database" is not allowed`)
	assertViolationContains(t, violations, `repository file type "models" is not allowed`)
	assertViolationContains(t, violations, `service file type "model" is not allowed`)
	assertViolationContains(t, violations, `service file type "constants" is not allowed`)
	assertViolationContains(t, violations, `service file type "errors" is not allowed`)
	assertViolationContains(t, violations, `service file type "utils" is not allowed`)
	assertViolationContains(t, violations, `service file type "validation" is not allowed`)
	assertViolationContains(t, violations, `repository file type "model" is not allowed`)
	assertViolationContains(t, violations, `repository file type "constants" is not allowed`)
	assertViolationContains(t, violations, `repository file type "errors" is not allowed`)
	assertViolationContains(t, violations, `repository file type "utils" is not allowed`)
}
