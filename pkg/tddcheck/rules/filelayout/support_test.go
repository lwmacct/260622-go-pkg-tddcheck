package filelayout

import (
	"path/filepath"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
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

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
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

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "support files must not import example.com/app/internal/service")
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
		"internal/repository/x.shared.support.go": `package repository

type SchemaDef struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "repository support type IdentityUserRow must start with IdentitySSHKey")
	assertViolationContains(t, violations, "repository support type IdentitySSHKeychain must start with IdentitySSHKey")
	if len(violations) != 2 {
		t.Fatalf("expected two repository support type violations, got %#v", violations)
	}
}

func TestViolationsChecksRepositorySupportValueSubject(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/device.support.go": `package repository
import "errors"
const DeviceTable = "devices"
const Table = "devices"
var ErrDeviceMissing = errors.New("missing")
var ErrMissing = errors.New("missing")
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "subject-specific declaration Table must start with Device")
	assertViolationContains(t, violations, "subject-specific declaration ErrMissing must start with ErrDevice")
	assertNoViolationContains(t, violations, "subject-specific declaration DeviceTable")
	assertNoViolationContains(t, violations, "subject-specific declaration ErrDeviceMissing")
}

func TestViolationsChecksSupportFunctionSubject(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
		"internal/service/device.support.go": `package service
func WrapDeviceError() error { return nil }
func IsDeviceError() bool { return false }
func AsDeviceError() error { return nil }
func WrapUserError() error { return nil }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "subject-specific declaration WrapUserError must start with WrapDevice")
	assertNoViolationContains(t, violations, "subject-specific declaration WrapDeviceError")
	assertNoViolationContains(t, violations, "subject-specific declaration IsDeviceError")
	assertNoViolationContains(t, violations, "subject-specific declaration AsDeviceError")
}

func TestViolationsLimitsSupportDeclarationLines(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.support.go": `package service

import "errors"

type Device struct {
	ID string
}

const DeviceStatusActive = "active"

var ErrDeviceUnavailable = errors.New("device unavailable")
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{MaxSupportDeclarationLines: 4})
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "support declarations occupy 5 lines (maximum 4); move types, consts, and Err* vars to device.types.go")
}

func TestViolationsLimitsArchitectureSupportDeclarationLines(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/x.http.support.go": `package handler

type Config struct {
	Name string
}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{MaxSupportDeclarationLines: 2})
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "move types, consts, and Err* vars to x.http.types.go")
}
