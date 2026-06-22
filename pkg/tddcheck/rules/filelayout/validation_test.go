package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksValidationContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.validation.go": `package service
type DeviceValidator struct{}
func checkDevice() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "validation files must not declare types")
	assertViolationContains(t, violations, "validation functions must be package-level validate* or normalize*")
}
