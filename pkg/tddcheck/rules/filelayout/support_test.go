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
	assertViolationContains(t, violations, "support vars must start with Err")
	assertViolationContains(t, violations, "support functions must start with util, validate, normalize, Wrap, Is, or As")
}
