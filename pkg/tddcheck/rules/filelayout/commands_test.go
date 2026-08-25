package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksCommandsContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.commands.go": `package service
type DeviceRequest struct{}
func bad() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "commands files must only declare types")
}

func TestViolationsChecksCommandsTypeNames(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.commands.go": `package service
type DevicePayload struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "command type DevicePayload must end with Request, Response, Result, or Item")
}

func TestViolationsRequireCommandsSubjectPrefix(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.commands.go": `package service
type CreateDeviceRequest struct{}
type DeviceCreateRequest struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "subject-specific declaration CreateDeviceRequest must start with Device")
	assertNoViolationContains(t, violations, "subject-specific declaration DeviceCreateRequest")
}
