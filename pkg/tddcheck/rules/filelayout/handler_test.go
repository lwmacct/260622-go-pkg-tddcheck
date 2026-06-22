package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksHandlerContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device_group.handler.go": `package handler
type deviceGroupHandler struct{}
type deviceGroupHelper struct{}
func RegisterDeviceGroups() {}
func newDeviceGroupHelper() {}
func (h deviceGroupHelper) list() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "handler files must only declare deviceGroupHandler")
	assertViolationContains(t, violations, "handler files must only declare Register* package-level functions")
	assertViolationContains(t, violations, "handler receiver methods must use deviceGroupHandler")
}
