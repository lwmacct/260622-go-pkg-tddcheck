package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRejectsJoinedMultiWordResourceScope(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/devicegroup.mapper.go": `package service
func ToDeviceGroupRow() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `scope "devicegroup" must use snake_case subject name "device_group"`)
}

func TestViolationsRejectsArchitectureScopeWithoutPrefix(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/shared.models.go": `package service
type SharedModel struct{}
`,
		"internal/handler/http.endpoint.go": `package handler
type Endpoint struct{}
func NewEndpoint() *Endpoint { return &Endpoint{} }
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `architecture scope "shared" must use x_shared prefix`)
	assertViolationContains(t, violations, `architecture scope "http" must use x_http prefix`)
}

func TestViolationsRejectsEscapedTypeInResourceScope(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device_update.utils.go": `package service
func updateDevice() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `scope "device_update" must not encode file type`)
}
