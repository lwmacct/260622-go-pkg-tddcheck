package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsAcceptsRepositorySchemaModels(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/identity_user.schema.go": `package repository
import "context"
type IdentityUserModel struct{}
func (*IdentityUserModel) BeforeCreateTable(context.Context) error { return nil }
func IdentityUserSchema() {}
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

func TestViolationsChecksRepositorySchemaContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/identity_user.schema.go": `package repository
type IdentityUserRow struct{}
type UserModel struct{}
type IdentityUserID int64
var tableName = "users"
func (*IdentityUserRow) BeforeCreateTable() error { return nil }
func BuildUser() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "schema model type IdentityUserRow must start with IdentityUser and end with Model")
	assertViolationContains(t, violations, "schema model type UserModel must start with IdentityUser and end with Model")
	assertViolationContains(t, violations, "schema files must only declare model structs")
	assertViolationContains(t, violations, "schema files must only declare model structs and schema lifecycle functions")
	assertViolationContains(t, violations, "schema receiver methods must use *Model receivers")
	assertViolationContains(t, violations, "schema package-level functions must be schema lifecycle functions")
}
