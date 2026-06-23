package filelayout

import (
	"path/filepath"
	"testing"
)

func TestProviderFilesAllowServicePortImplementations(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/image_captcha.service.go": `package service
type ImageCaptchaService struct{}
func NewImageCaptchaService() *ImageCaptchaService { return &ImageCaptchaService{} }
`,
		"internal/service/image_captcha.provider.go": `package service
import "context"
type ImageCaptchaProvider struct{}
func NewImageCaptchaProvider() *ImageCaptchaProvider { return &ImageCaptchaProvider{} }
func (p *ImageCaptchaProvider) Name() string { return "image" }
func (p *ImageCaptchaProvider) Create(context.Context) error { return nil }
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestProviderFilesRequireOwningServiceSubject(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/image_captcha.provider.go": `package service
type ImageCaptchaProvider struct{}
func NewImageCaptchaProvider() *ImageCaptchaProvider { return &ImageCaptchaProvider{} }
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `service subject "image_captcha" must declare image_captcha.service.go with NewImageCaptchaService`)
}

func TestProviderFilesRejectTransportAndPersistenceCoupling(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/image_captcha.provider.go": `package service
import (
	"net/http"
	"example.com/app/internal/adapter"
	"example.com/app/internal/runtime/nodepool"
	"example.com/app/internal/repository"
)
type ImageAuthChallengeRequest struct{}
type RemoteProvider struct{ Name string ` + "`json:\"name\"`" + ` }
var ErrProvider = http.ErrAbortHandler
func BuildProvider() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "provider files must not import net/http")
	assertViolationContains(t, violations, "provider files must not import example.com/app/internal/adapter")
	assertViolationContains(t, violations, "provider files must not import example.com/app/internal/runtime/nodepool")
	assertViolationContains(t, violations, "provider files must not import example.com/app/internal/repository")
	assertViolationContains(t, violations, "provider type ImageAuthChallengeRequest must not use transport or command suffixes")
	assertViolationContains(t, violations, "provider types must not declare transport or persistence tags")
	assertViolationContains(t, violations, "provider files must only declare provider types and functions")
	assertViolationContains(t, violations, "provider package-level functions must start with New")
}

func TestProviderFilesRejectMethodsOnExternalTypes(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/image_captcha.provider.go": `package service
type ImageCaptchaProvider struct{}
type CaptchaService struct{}
func NewImageCaptchaProvider() *ImageCaptchaProvider { return &ImageCaptchaProvider{} }
func (s *CaptchaService) Name() string { return "bad" }
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "provider receiver methods must use ImageCaptchaProvider")
}

func TestProviderFilesRejectConstructorsWithoutProviderReturn(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/image_captcha.provider.go": `package service
type ImageCaptchaProvider struct{}
func NewImageCaptchaProvider() string { return "" }
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "provider constructors must return ImageCaptchaProvider")
}
