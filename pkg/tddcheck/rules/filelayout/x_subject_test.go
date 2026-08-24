package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRejectsJoinedMultiWordSubject(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/devicegroup.mapper.go": `package service
func ToDeviceGroupRow() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `subject "devicegroup" must use snake_case name "device_group"`)
}

func TestViolationsRejectsArchitectureNamespaceWithoutMarker(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/shared.models.go": `package service
type SharedModel struct{}
`,
		"internal/handler/http.endpoint.go": `package handler
type Endpoint struct{}
func NewEndpoint() *Endpoint { return &Endpoint{} }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `architecture namespace "shared" must use x.shared.{kind}.go naming`)
	assertViolationContains(t, violations, `architecture namespace "http" must use x.http.{kind}.go naming`)
}

func TestViolationsRejectsLegacyArchitecturePrefix(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/x_http.support.go": `package handler
type Config struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `architecture namespace "http" must use x.http.{kind}.go naming`)
}

func TestViolationsRejectsEscapedKindInSubject(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device_update.utils.go": `package service
func updateDevice() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `subject "device_update" must not encode file kind`)
}
