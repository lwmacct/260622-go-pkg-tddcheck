package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsAllowsModelAsDomainScope(t *testing.T) {
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

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertNoViolationContains(t, violations, `scope "vendor_model" must not encode file type`)
	assertNoViolationContains(t, violations, `scope "admin_model" must not encode file type`)
}
