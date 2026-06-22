package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksStoreContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.store.go": `package repository
import "context"
type DeviceStore struct{}
func NewDeviceStore() {}
func (s *DeviceStore) ListDevices() {}
func (s *Store) MissingContext() error { return nil }
func (s *Store) MissingError(ctx context.Context) bool { return true }
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "store files must only declare Store receiver methods")
	assertViolationContains(t, violations, "store methods must accept context.Context as the first parameter")
	assertViolationContains(t, violations, "store methods must return error as the last result")
}
