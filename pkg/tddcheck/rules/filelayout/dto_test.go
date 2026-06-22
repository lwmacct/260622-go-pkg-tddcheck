package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksDTOContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.dto.go": `package handler
type Device struct{}
func bad() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "must end with DTO or DTOs")
	assertViolationContains(t, violations, "dto files must not declare functions")
}
