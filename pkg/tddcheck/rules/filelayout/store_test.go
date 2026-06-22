package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksStoreContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.store.go": `package repository
type DeviceStore struct{}
func NewDeviceStore() {}
func (s *DeviceStore) ListDevices() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "store files must only declare Store receiver methods")
}
