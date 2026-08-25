package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRejectCrossSubjectTypeDeclarations(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/api_key.service.go": `package service
type APIKeyService struct{}
func NewAPIKeyService() *APIKeyService { return &APIKeyService{} }
`,
		"internal/service/api_key.types.go": `package service
type APIKey struct { Label AudienceLabel }
type AudienceLabel struct{}
`,
		"internal/service/audience_label.service.go": `package service
type AudienceLabelService struct{}
func NewAudienceLabelService() *AudienceLabelService { return &AudienceLabelService{} }
`,
		"internal/service/audience_label.types.go": `package service
type AudienceLabelValue struct{}
`,
		"internal/handler/api_key.dto.go": `package handler
type APIKeyDTO struct { Label AudienceLabelDTO }
type AudienceLabelDTO struct{}
`,
		"internal/handler/audience_label.dto.go": `package handler
type AudienceLabelValueDTO struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "subject-specific declaration AudienceLabel must start with APIKey")
	assertViolationContains(t, violations, "subject-specific declaration AudienceLabelDTO must start with APIKey")
}

func TestViolationsAllowsCrossSubjectReferencesAndSharedDeclarations(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/api_key.service.go": `package service
type APIKeyService struct{}
func NewAPIKeyService() *APIKeyService { return &APIKeyService{} }
`,
		"internal/service/api_key.types.go": `package service
type APIKey struct { Label AudienceLabel }
`,
		"internal/service/audience_label.service.go": `package service
type AudienceLabelService struct{}
func NewAudienceLabelService() *AudienceLabelService { return &AudienceLabelService{} }
`,
		"internal/service/audience_label.types.go": `package service
type AudienceLabel struct{}
`,
		"internal/service/x.shared.types.go": `package service
type AudienceLabelReference struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoViolationContains(t, violations, "subject-specific declaration APIKey")
	assertNoViolationContains(t, violations, "subject-specific declaration AudienceLabelReference")
}
