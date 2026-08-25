package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksUtilsContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.utils.go": `package handler

const defaultName = "device"
type DeviceHelper struct{}
func (h DeviceHelper) utilDevice() {}
func buildDevice() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "utils files must only declare util* functions")
	assertViolationContains(t, violations, "utils functions must not use receivers")
	assertViolationContains(t, violations, "utils function buildDevice must start with util")
}
