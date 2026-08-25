package layerdeps

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func checkRoot(root string, configs ...rulekit.Config) ([]Violation, error) {
	var config rulekit.Config
	if len(configs) > 0 {
		config = configs[0]
	}
	return Check(context.Background(), root, config)
}

func TestViolationsRejectsForbiddenHSRImports(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.handler.go": `package handler
import _ "example.com/app/internal/repository"
`,
		"internal/service/device.service.go": `package service
import _ "example.com/app/internal/repository"
`,
		"internal/repository/device.store.go": `package repository
import _ "example.com/app/internal/service"
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %#v", violations)
	}
	assertLayerViolationContains(t, violations, "handler must not import repository")
	assertLayerViolationContains(t, violations, "repository must not import service")
}

func TestViolationsRejectsForbiddenImportsFromNestedStrictLayer(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device/device.handler.go": `package handler
import _ "example.com/app/internal/repository"
`,
		"internal/repository/device.store.go": `package repository
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertLayerViolationContains(t, violations, "handler must not import repository")
}

func TestViolationsChecksSharedFreeFileImports(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/x.shared.free.go": `package handler
import _ "example.com/app/internal/repository"
`,
		"internal/repository/x.shared.free.go": `package repository
import _ "example.com/app/internal/service"
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %#v", violations)
	}
	assertLayerViolationContains(t, violations, "handler must not import repository")
	assertLayerViolationContains(t, violations, "repository must not import service")
}

func TestViolationsChecksImportsInsideAndOutsideFreeFiles(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/x.shared.free.go": `package handler
import _ "example.com/app/internal/repository"
`,
		"internal/handler/device.handler.go": `package handler
import _ "example.com/app/internal/repository"
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %#v", violations)
	}
	assertLayerViolationContains(t, violations, "handler must not import repository")
}

func TestViolationsAllowsConfiguredTargetExceptions(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/adapter/sshcmd/middleware.go": `package sshcmd
import _ "example.com/app/internal/adapter/sshauth"
`,
		"internal/adapter/httpauth/service.go": `package httpauth
import _ "example.com/app/internal/adapter/wsworkspace"
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{
		LayerDirs: []string{"adapter"},
		LayerRules: []rulekit.LayerDependencyRule{
			{
				SourceLayer:             "adapter",
				TargetLayer:             "adapter",
				ExceptTargetRelPrefixes: []string{"adapter/sshauth"},
				Message:                 "adapter must not import adapter",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %#v", violations)
	}
	assertLayerViolationContains(t, violations, "adapter must not import adapter")
}

func TestViolationsTreatsPrefixesAsPathSegments(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/adapter/source/endpoint.go": `package source
import _ "example.com/app/internal/adapter/sshauth-extra"
`,
		"internal/adapter/sshauth-extra/service.go": `package sshauthextra
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{
		LayerDirs: []string{"adapter"},
		LayerRules: []rulekit.LayerDependencyRule{{
			SourceLayer: "adapter", TargetLayer: "adapter", TargetRelPrefix: "adapter/sshauth",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected adjacent path name not to match prefix, got %#v", violations)
	}
}

func TestViolationsUsesDependencyLayerDirsWithoutFileLayoutLayers(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/runtime/nodepool/pool.go": `package nodepool
import _ "example.com/app/internal/adapter/wsworkspace"
`,
		"internal/adapter/wsworkspace/endpoint.go": `package wsworkspace
import _ "example.com/app/internal/runtime/nodepool"
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{
		LayerDirs:           []string{"adapter"},
		DependencyLayerDirs: []string{"adapter", "runtime"},
		LayerRules: []rulekit.LayerDependencyRule{
			{
				SourceLayer: "runtime",
				TargetLayer: "adapter",
				Message:     "runtime must not import adapter",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %#v", violations)
	}
	assertLayerViolationContains(t, violations, "runtime must not import adapter")
}

func TestViolationsAllowsConfiguredSourceAndTargetExceptions(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/adapter/sshcmd/middleware.go": `package sshcmd
import _ "example.com/app/internal/adapter/sshauth"
`,
		"internal/adapter/sshproxyjump/server.go": `package sshproxyjump
import _ "example.com/app/internal/adapter/sshauth"
`,
		"internal/adapter/httpauth/middleware.go": `package httpauth
import _ "example.com/app/internal/adapter/sshauth"
`,
		"internal/adapter/wsworkspace/endpoint.go": `package wsworkspace
import _ "example.com/app/internal/adapter/wsnodetunnel"
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{
		LayerDirs: []string{"adapter"},
		LayerRules: []rulekit.LayerDependencyRule{
			{
				SourceLayer: "adapter",
				TargetLayer: "adapter",
				ExceptSourceRelPrefixes: []string{
					"adapter/sshcmd",
					"adapter/sshproxyjump",
				},
				TargetRelPrefix: "adapter/sshauth",
				Message:         "only ssh command adapters may import sshauth",
			},
			{
				SourceLayer: "adapter",
				TargetLayer: "adapter",
				ExceptTargetRelPrefixes: []string{
					"adapter/sshauth",
				},
				Message: "adapter must not import adapter",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %#v", violations)
	}
	assertLayerViolationContains(t, violations, "only ssh command adapters may import sshauth")
	assertLayerViolationContains(t, violations, "adapter must not import adapter")
}

func fixture(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.27.0\n")
	for name, content := range files {
		writeFile(t, root, name, content)
	}
	return root
}

func writeFile(t *testing.T, root string, name string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertLayerViolationContains(t *testing.T, violations []Violation, needle string) {
	t.Helper()

	for _, violation := range violations {
		if strings.Contains(violation.Message, needle) {
			return
		}
	}
	t.Fatalf("expected violation containing %q, got %#v", needle, violations)
}
