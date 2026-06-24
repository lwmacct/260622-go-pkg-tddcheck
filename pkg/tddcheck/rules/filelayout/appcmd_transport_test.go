package filelayout

import (
	"path/filepath"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func TestViolationsRejectsAppcmdTransportOwnership(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/appcmd/server/app.httpapi.go": `package server
import "github.com/danielgtaylor/huma/v2"
type authUserDTO struct{}
type authUserTDO struct{}
func register(api huma.API) {
	huma.Register(api, huma.Operation{}, nil)
	huma.NewGroup(api, "/auth")
}
`,
	})

	violations, err := New(filepath.Join(root, "internal"), rulekit.WithConfig(rulekit.Config{
		DependencyLayerDirs: []string{"appcmd"},
	})).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "appcmd must not import huma")
	assertViolationContains(t, violations, "appcmd must not declare DTO/TDO types")
	assertViolationContains(t, violations, "appcmd must not register huma routes")
}

func TestViolationsAllowsAppcmdComposition(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/appcmd/server/app.httpapi.go": `package server
import "net/http"
type App struct{}
func (app *App) newHTTPAPIHandler() http.Handler {
	return http.NewServeMux()
}
`,
	})

	violations, err := New(filepath.Join(root, "internal"), rulekit.WithConfig(rulekit.Config{
		DependencyLayerDirs: []string{"appcmd"},
	})).Violations()
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}
