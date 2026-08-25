package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksMapperContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.mapper.go": `package service

import "context"

type DeviceMapper struct{}
func (m DeviceMapper) ToDevice() {}
func MapDevice() {}
func FromDevice() {}
func BuildDevice() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "mapper files must not import context")
	assertViolationContains(t, violations, "mapper files must only declare functions")
	assertViolationContains(t, violations, "mapper functions must not use receivers")
	assertViolationContains(t, violations, "mapper function MapDevice must start with To")
	assertViolationContains(t, violations, "mapper function FromDevice must start with To")
	assertViolationContains(t, violations, "mapper function BuildDevice must start with To")
}

func TestViolationsRequireMapperSubjectPrefix(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.mapper.go": `package service
func ToRepositoryDevice() {}
func ToDeviceRow() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "subject-specific declaration ToRepositoryDevice must start with ToDevice")
	assertNoViolationContains(t, violations, "subject-specific declaration ToDeviceRow")
}
