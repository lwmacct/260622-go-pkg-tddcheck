package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRequireSubjectPrefixForExportedTypes(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/api_key.service.go": `package service
type APIKeyService struct{}
func NewAPIKeyService() *APIKeyService { return &APIKeyService{} }
`,
		"internal/service/api_key.types.go": `package service
type APIKey struct{}
type APIKeyOwnerCreateInput struct{}
type UserAPIKeyCreateInput struct{}
type apiKeyRuntime struct{}
`,
		"internal/handler/api_key.dto.go": `package handler
type APIKeyOwnerDTO struct{}
type UserAPIKeyDTO struct{}
`,
		"internal/service/user.support.go": `package service
type User struct{}
type UserCreateInput struct{}
type CreateUserInput struct{}
`,
		"internal/service/user.service.go": `package service
type UserService struct{}
func NewUserService() *UserService { return &UserService{} }
`,
		"internal/service/usage.service.go": `package service
type UsageService struct{}
func NewUsageService() *UsageService { return &UsageService{} }
`,
		"internal/service/usage.types.go": `package service
type UsageCharge struct{}
type UserUsageChargeListInput struct{}
`,
		"internal/handler/usage.dto.go": `package handler
type UsageChargeOwnerListInputDTO struct{}
type UserUsageChargeListInputDTO struct{}
`,
		"internal/service/resource_package.service.go": `package service
type ResourcePackageService struct{}
func NewResourcePackageService() *ResourcePackageService { return &ResourcePackageService{} }
`,
		"internal/service/resource_package.types.go": `package service
type ResourcePackage struct{}
type UserResourceLedgerListInput struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "subject-specific declaration UserAPIKeyCreateInput must start with APIKey")
	assertViolationContains(t, violations, "subject-specific declaration UserAPIKeyDTO must start with APIKey")
	assertViolationContains(t, violations, "subject-specific declaration CreateUserInput must start with User")
	assertViolationContains(t, violations, "subject-specific declaration UserUsageChargeListInput must start with Usage")
	assertViolationContains(t, violations, "subject-specific declaration UserUsageChargeListInputDTO must start with Usage")
	assertViolationContains(t, violations, "subject-specific declaration UserResourceLedgerListInput must start with ResourcePackage")
	if len(violations) != 6 {
		t.Fatalf("expected six subject-order violations, got %#v", violations)
	}
}
