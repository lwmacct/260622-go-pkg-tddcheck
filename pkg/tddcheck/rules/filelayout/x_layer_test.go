package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRejectsArchitectureScopeInWrongLayer(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/x_database.service.go": `package service
func NewDatabaseService() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `architecture scope "x_database" must not use x_database.service.go in service`)
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
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `service file type "models" is not allowed`)
	assertViolationContains(t, violations, `service file type "writes" is not allowed`)
	assertViolationContains(t, violations, `repository file type "database" is not allowed`)
	assertViolationContains(t, violations, `repository file type "models" is not allowed`)
}
