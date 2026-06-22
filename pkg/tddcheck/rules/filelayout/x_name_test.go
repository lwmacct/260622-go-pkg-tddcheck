package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRejectsWeakScopeAndLegacyNames(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/default.service.go": `package service
func NewDefaultService() {}
`,
		"internal/service/device_validation.go": `package service
func validateDevice() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "scope \"default\" is too weak")
	assertViolationContains(t, violations, "must use {scope}.{type}.go")
}
