package filelayout

import (
	"path/filepath"
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
		"internal/adapter/wsworkspace/service.handler.go": `package wsworkspace
func handleMessage() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal"), rulekit.WithConfig(rulekit.Config{
		LayerDirs: []string{"adapter"},
		LayerFileNameModes: map[string]string{
			"adapter": rulekit.FileNameModePackageKind,
		},
		LayerFileKinds: map[string][]string{
			"adapter": {"doc", "service", "service.handler"},
		},
		ArchitectureFileKinds: map[string]map[string][]string{},
		EscapedScopeSuffixes:  []string{},
		LayerRules:            []rulekit.LayerDependencyRule{},
	})).Violations()
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

	violations, err := New(filepath.Join(root, "internal"), rulekit.WithConfig(rulekit.Config{
		LayerDirs: []string{"adapter"},
		LayerFileNameModes: map[string]string{
			"adapter": rulekit.FileNameModePackageKind,
		},
		LayerFileKinds: map[string][]string{
			"adapter": {"doc", "service"},
		},
		ArchitectureFileKinds: map[string]map[string][]string{},
		EscapedScopeSuffixes:  []string{},
		LayerRules:            []rulekit.LayerDependencyRule{},
	})).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `adapter file type "random" is not allowed`)
}
