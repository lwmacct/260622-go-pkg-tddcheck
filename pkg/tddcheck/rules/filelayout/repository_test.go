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
		"internal/repository/x_store.repository.go": `package repository
type Store struct{}
type StoreConfig struct{}
func BuildStore() {}
func (r Repository) List() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "repository files must use x_store scope")
	assertViolationContains(t, violations, "repository files must declare Store struct")
	assertViolationContains(t, violations, "repository files must declare NewStore")
	assertViolationContains(t, violations, "repository files must only declare Store")
	assertViolationContains(t, violations, "repository receiver methods must use Store")
	assertViolationContains(t, violations, "repository package-level functions must be NewStore or private helpers")
}

func TestViolationsAcceptsStoreRepositoryRoot(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/x_store.repository.go": `package repository
type Store struct{}
func NewStore() *Store { return &Store{} }
func (s *Store) RunInTx() {}
func privateHelper() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}
