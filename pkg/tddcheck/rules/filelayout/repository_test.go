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
		"internal/repository/x.store.repository.go": `package repository
type Store struct{}
type StoreConfig struct{}
func BuildStore() {}
func (r Repository) List() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `file kind "repository" requires architecture namespace "store"`)
	assertViolationContains(t, violations, "repository files must declare NewStore")
	assertViolationContains(t, violations, "repository files must only declare Store")
	assertViolationContains(t, violations, "repository receiver methods must use Store")
	assertViolationContains(t, violations, "repository package-level functions must be NewStore or private helpers")
}

func TestViolationsAcceptsStoreRepositoryRoot(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/x.store.repository.go": `package repository
type Store struct{}
func NewStore() *Store { return &Store{} }
func (s *Store) RunInTx() {}
func privateHelper() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}
