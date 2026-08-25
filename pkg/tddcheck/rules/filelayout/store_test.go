package filelayout

import (
	"path/filepath"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func TestViolationsChecksStoreContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.store.go": `package repository
import "context"
type DeviceStore struct{}
func NewDeviceStore() {}
func (s *DeviceStore) ListDevices() {}
func (s *Store) HandleDevice(ctx context.Context) error { return nil }
func (s *Store) List(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) Listdevice(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) ListDevicesWhere(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) ListDevicesEnabled(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) ListMachineRooms(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) MissingContext() error { return nil }
func (s *Store) MissingError(ctx context.Context) bool { return true }
func (s *Store) ListDevice(ctx context.Context) (*string, error) { return nil, nil }
func (s *Store) FetchDevices(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) CreateDevice(ctx context.Context) (bool, error) { return false, nil }
func (s *Store) DeleteDevice(ctx context.Context) (*string, error) { return nil, nil }
func (s *Store) AddDevice(ctx context.Context) (bool, error) { return false, nil }
func (s *Store) listDevicesWhere(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) listDevicesMissingContext() ([]string, error) { return nil, nil }
func (s *Store) listDevicesMissingError(ctx context.Context) []string { return nil }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "store files must only declare Store receiver methods")
	assertViolationContains(t, violations, "store method action is not configured")
	assertViolationContains(t, violations, "store method names must include a subject after the action")
	assertViolationContains(t, violations, "store method names must use Action+UpperCamelSubject")
	assertViolationContains(t, violations, "exported store method subjects must start with Device as an exact resource segment")
	assertViolationContains(t, violations, "List store method qualifiers after plural subjects must start with By, For, With, or Without")
	assertViolationContains(t, violations, "store method names must not expose query implementation details")
	assertViolationContains(t, violations, "store methods must accept context.Context as the first parameter")
	assertViolationContains(t, violations, "store methods must return error as the last result")
	assertViolationContains(t, violations, "List store methods must return a slice and error")
	assertViolationContains(t, violations, "Delete store methods must return optional bool and error")
	assertViolationContains(t, violations, "Add store methods must only return error")
}

func TestViolationsAllowsConfiguredStoreDomainAction(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.store.go": `package repository
import "context"
func (s *Store) ReserveDevice(ctx context.Context) error { return nil }
`,
	})

	config := rulekit.Config{
		StoreMethodActions: []string{"Reserve"},
	}
	violations, err := checkRoot(filepath.Join(root, "internal"), config)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestViolationsReportsStoreStyleAsWarning(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.store.go": `package repository
import "context"
func (s *Store) ReserveDevice(ctx context.Context) error { return nil }
func (s *Store) ListDevice(ctx context.Context) (*string, error) { return nil, nil }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	var warnings int
	for _, violation := range violations {
		if violation.Code == "filelayout/store-style" && violation.Severity == rulekit.SeverityWarning {
			warnings++
		}
	}
	if warnings != 2 {
		t.Fatalf("expected two store style warnings, got %#v", violations)
	}
}

func TestViolationsAllowsStoreActionSubjectNames(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.store.go": `package repository
import "context"
func (s *Store) ListDevices(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) ListDevicesByEnabled(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) FetchDeviceByID(ctx context.Context) (string, error) { return "", nil }
func (s *Store) CountDevices(ctx context.Context) (int64, error) { return 0, nil }
func (s *Store) ExistsDeviceByID(ctx context.Context) (bool, error) { return false, nil }
func (s *Store) CreateDevice(ctx context.Context) (string, error) { return "", nil }
func (s *Store) UpdateDevice(ctx context.Context) (bool, error) { return false, nil }
func (s *Store) UpsertDevice(ctx context.Context) (string, string, error) { return "", "", nil }
func (s *Store) DeleteDeviceByID(ctx context.Context) (bool, error) { return false, nil }
func (s *Store) RemoveDevice(ctx context.Context) error { return nil }
func (s *Store) AddDevice(ctx context.Context) error { return nil }
func (s *Store) ReplaceDeviceItems(ctx context.Context) error { return nil }
func (s *Store) listDevices(ctx context.Context, enabledOnly bool) ([]string, error) { return nil, nil }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("expected no violations, got: %#v", violations)
	}
}

func TestViolationsAllowsStorePluralSubjectAfterVowelY(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/api_key.store.go": `package repository
import "context"
func (s *Store) ListAPIKeys(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) ListAPIKeysByStatus(ctx context.Context) ([]string, error) { return nil, nil }
func (s *Store) CountAPIKeys(ctx context.Context) (int, error) { return 0, nil }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("expected no violations, got: %#v", violations)
	}
}

func TestViolationsResolveAliasedContextAndErrorTypes(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.store.go": `package repository
import ctx "context"
type Failure = error
func (s *Store) ListDevices(_ ctx.Context) ([]string, Failure) { return nil, nil }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoViolationContains(t, violations, "context.Context")
	assertNoViolationContains(t, violations, "return error")
}
