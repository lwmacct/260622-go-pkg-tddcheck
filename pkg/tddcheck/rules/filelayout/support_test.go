package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksServiceSupportContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.support.go": `package service

import (
	"net/http"
)

type DeviceDTO struct {
	ID string ` + "`json:\"id\"`" + `
}
type DeviceRequest struct{}
func bad() {}
var status = http.StatusOK
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "support files must not import net/http")
	assertViolationContains(t, violations, "service support type DeviceDTO must not use transport or command suffixes")
	assertViolationContains(t, violations, "service support types must not declare transport or persistence tags")
	assertViolationContains(t, violations, "service support type DeviceRequest must not use transport or command suffixes")
	assertViolationContains(t, violations, "support functions must start with util, validate, normalize, Wrap, Is, or As")
}

func TestViolationsChecksRepositorySupportContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.support.go": `package repository

import (
	_ "example.com/app/internal/service"
	"net/http"
)

type Device struct{}
type DeviceModel struct{}
type DeviceTagged struct {
	ID int64 ` + "`bun:\"id,pk\" gorm:\"primaryKey\"`" + `
}
var status = http.StatusOK
func Bad() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "support files must not import example.com/app/internal/service")
	assertViolationContains(t, violations, "support files must not import net/http")
	assertViolationContains(t, violations, "repository support files must not declare schema models")
	assertViolationContains(t, violations, "support vars must start with Err")
	assertViolationContains(t, violations, "support functions must start with util, validate, normalize, Wrap, Is, or As")
}

func TestViolationsChecksRepositorySupportTypeSubject(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/identity_ssh_key.support.go": `package repository

type IdentitySSHKey struct{}
type IdentitySSHKeyCreate struct{}
type IdentitySSHKeyRow struct{}
type IdentitySSHKeyUserRow struct{}
type IdentityUserRow struct{}
type IdentitySSHKeychain struct{}
type localHelper struct{}
`,
		"internal/repository/x_shared.support.go": `package repository

type SchemaDef struct{}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "repository support type IdentityUserRow must start with IdentitySSHKey")
	assertViolationContains(t, violations, "repository support type IdentitySSHKeychain must start with IdentitySSHKey")
	if len(violations) != 2 {
		t.Fatalf("expected two repository support type violations, got %#v", violations)
	}
}
