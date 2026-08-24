package rulekit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHonorsBuildConstraints(t *testing.T) {
	root := snapshotFixture(t, map[string]string{
		"internal/service/linux.go": `//go:build linux

package service

const Platform = "linux"
`,
		"internal/service/windows.go": `//go:build windows

package service

this is intentionally invalid Go
`,
	})

	snapshot, err := Load(context.Background(), filepath.Join(root, "internal"), Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].Base != "linux.go" {
		t.Fatalf("expected only the active linux file, got %#v", snapshot.Files)
	}
}

func TestLoadCanIncludeTestsWithoutDuplicatingProductionFiles(t *testing.T) {
	root := snapshotFixture(t, map[string]string{
		"internal/service/service.go":      "package service\nconst Production = true\n",
		"internal/service/service_test.go": "package service\nconst TestOnly = true\n",
	})

	snapshot, err := Load(context.Background(), filepath.Join(root, "internal"), Config{IncludeTests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Files) != 2 {
		t.Fatalf("expected one production and one test file, got %#v", snapshot.Files)
	}
}

func TestLoadStrictPackagesControlsTypeErrors(t *testing.T) {
	root := snapshotFixture(t, map[string]string{
		"internal/service/service.go": "package service\nvar Broken = missingName\n",
	})

	snapshot, err := Load(context.Background(), filepath.Join(root, "internal"), Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.LoadErrors) == 0 {
		t.Fatal("expected best-effort load to retain the type error")
	}
	if _, err := Load(context.Background(), filepath.Join(root, "internal"), Config{StrictPackages: true}); err == nil {
		t.Fatal("expected strict package loading to fail")
	}
}

func TestLoadExcludesConfiguredPackageDirectoriesBeforeReportingErrors(t *testing.T) {
	root := snapshotFixture(t, map[string]string{
		"internal/service/service.go":  "package service\nconst Ready = true\n",
		"internal/generated/broken.go": "package generated\nthis is invalid\n",
	})

	snapshot, err := Load(context.Background(), filepath.Join(root, "internal"), Config{
		SkipDirs:       []string{"generated"},
		StrictPackages: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].Base != "service.go" {
		t.Fatalf("configured skip directory leaked into snapshot: %#v", snapshot.Files)
	}
}

func snapshotFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files["go.mod"] = "module example.com/app\n\ngo 1.27.0\n"
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
