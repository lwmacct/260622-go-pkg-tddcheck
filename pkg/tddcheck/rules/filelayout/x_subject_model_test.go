package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsAllowsModelAsDomainSubject(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/vendor_model.service.go": `package service
type VendorModelService struct{}
func NewVendorModelService() *VendorModelService { return &VendorModelService{} }
`,
		"internal/handler/admin_model.handler.go": `package handler
type adminModelHandler struct{}
func RegisterAdminModel() {}
`,
		"internal/repository/vendor_model.schema.go": `package repository
type VendorModelModel struct{}
func VendorModelSchema() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoViolationContains(t, violations, `subject "vendor_model" must not encode file kind`)
	assertNoViolationContains(t, violations, `subject "admin_model" must not encode file kind`)
}
