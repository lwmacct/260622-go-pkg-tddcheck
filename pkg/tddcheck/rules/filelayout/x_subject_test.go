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

func TestViolationsAllowsBusinessSubjectMatchingArchitectureNamespace(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/shared.service.go": `package service
type SharedService struct{}
func NewSharedService() *SharedService { return &SharedService{} }
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

func TestViolationsRequiresArchitectureNamespaceForEndpoint(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/http.endpoint.go": `package handler
type Endpoint struct{}
func NewEndpoint() *Endpoint { return &Endpoint{} }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `file kind "endpoint" requires architecture namespace "http"`)
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
	if len(violations) != 1 || violations[0].Code != "filelayout/legacy-namespace" || violations[0].Fix == nil || violations[0].Fix.Rename == nil {
		t.Fatalf("expected structured rename fix, got %#v", violations)
	}
	if got := violations[0].Fix.Rename.To; got != "internal/handler/x.http.support.go" {
		t.Fatalf("unexpected rename target %q", got)
	}
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
