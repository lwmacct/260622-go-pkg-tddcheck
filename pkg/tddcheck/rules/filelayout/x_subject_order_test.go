package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRequireFileSubjectBeforeTypeQualifiers(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/api_key.service.go": `package service
type APIKeyService struct{}
func NewAPIKeyService() *APIKeyService { return &APIKeyService{} }
`,
		"internal/service/api_key.types.go": `package service
type APIKey struct{}
type APIKeyOwnerCreateInput struct{}
type UserAPIKeyCreateInput struct{}
type AudienceLabel struct{}
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
type ResourceLedger struct{}
type UserResourceLedgerListInput struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "type UserAPIKeyCreateInput contains file subject token API after a qualifier")
	assertViolationContains(t, violations, "type UserAPIKeyDTO contains file subject token API after a qualifier")
	assertViolationContains(t, violations, "type CreateUserInput contains file subject token User after a qualifier")
	assertViolationContains(t, violations, "type UserUsageChargeListInput contains file subject token Usage after a qualifier")
	assertViolationContains(t, violations, "type UserUsageChargeListInputDTO contains file subject token Usage after a qualifier")
	assertViolationContains(t, violations, "type UserResourceLedgerListInput contains file subject token Resource after a qualifier")
	if len(violations) != 6 {
		t.Fatalf("expected six subject-order violations, got %#v", violations)
	}
}
