package filelayout

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func TestViolationsAcceptsConfiguredPackageKindLayer(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/adapter/httpauth/doc.go": `package httpauth
`,
		"internal/adapter/httpauth/service.go": `package httpauth
type Service struct{}
`,
		"internal/adapter/wsworkspace/handler.go": `package wsworkspace
func handleMessage() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{
		LayerDirs: []string{"adapter"},
		LayerFileNameModes: map[string]string{
			"adapter": rulekit.FileNameModePackageKind,
		},
		LayerKindPolicies: map[string]map[string]string{
			"adapter": {"doc": "free", "service": "free", "handler": "free"},
		},
		ArchitectureNamespaces: map[string][]string{},
		EscapedSubjectSuffixes: []string{},
		LayerRules:             []rulekit.LayerDependencyRule{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestViolationsRejectsUnconfiguredPackageKindFile(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/adapter/httpauth/random.go": `package httpauth
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{
		LayerDirs: []string{"adapter"},
		LayerFileNameModes: map[string]string{
			"adapter": rulekit.FileNameModePackageKind,
		},
		LayerKindPolicies: map[string]map[string]string{
			"adapter": {"doc": "free", "service": "free"},
		},
		ArchitectureNamespaces: map[string][]string{},
		EscapedSubjectSuffixes: []string{},
		LayerRules:             []rulekit.LayerDependencyRule{},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `adapter file kind "random" is not allowed`)
}

func TestCheckRejectsUnknownDeclarationPolicy(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/adapter/httpauth/doc.go": "package httpauth\n",
	})

	_, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{
		LayerDirs:              []string{"adapter"},
		LayerFileNameModes:     map[string]string{"adapter": rulekit.FileNameModePackageKind},
		LayerKindPolicies:      map[string]map[string]string{"adapter": {"doc": "missing"}},
		ArchitectureNamespaces: map[string][]string{},
		LayerRules:             []rulekit.LayerDependencyRule{},
	})
	if err == nil || !strings.Contains(err.Error(), `unknown declaration policy "missing"`) {
		t.Fatalf("expected unknown declaration policy error, got %v", err)
	}
}
