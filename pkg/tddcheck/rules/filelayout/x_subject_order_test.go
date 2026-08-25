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
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "type UserAPIKeyCreateInput contains file subject APIKey after a qualifier")
	assertViolationContains(t, violations, "type UserAPIKeyDTO contains file subject APIKey after a qualifier")
	assertViolationContains(t, violations, "type CreateUserInput contains file subject User after a qualifier")
	if len(violations) != 3 {
		t.Fatalf("expected three subject-order violations, got %#v", violations)
	}
}
