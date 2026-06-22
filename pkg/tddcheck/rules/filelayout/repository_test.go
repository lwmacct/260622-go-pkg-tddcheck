package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksRepositoryContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.repository.go": `package repository
type DeviceRepository struct{}
func (r DeviceRepository) List() {}
func OpenDeviceRepository() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "repository files must use an architecture scope")
	assertViolationContains(t, violations, "repository receiver methods must belong to x_store Store")
}
