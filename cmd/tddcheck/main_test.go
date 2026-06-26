package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithArgsUsesLongFlagsForMap(t *testing.T) {
	root := cliFixture(t, map[string]string{
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
func (s *DeviceService) List() {}
`,
		"internal/repository/device.schema.go": `package repository
import "github.com/uptrace/bun"
type DeviceModel struct {
	bun.BaseModel ` + "`" + `bun:"table:devices,alias:d"` + "`" + `
	ID int64 ` + "`" + `bun:"id,pk,autoincrement"` + "`" + `
}
func DeviceSchema() {}
`,
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs([]string{"--root", filepath.Join(root, "internal"), "--map"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr:\n%s", code, stderr.String())
	}
	for _, needle := range []string{
		"tddcheck map: example.com/app",
		"- DeviceService (device)",
		"- devices (DeviceModel, device)",
	} {
		if !strings.Contains(stdout.String(), needle) {
			t.Fatalf("expected stdout to contain %q, got:\n%s", needle, stdout.String())
		}
	}
}

func TestRunWithArgsRejectsShortFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs([]string{"-map"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "short flags are not supported: -map") {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func TestRunWithArgsRejectsOldMapFormatArgument(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs([]string{"--map", "text"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unexpected argument text") {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func TestRunWithArgsRejectsFormatWithoutMap(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs([]string{"--format", "json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--format requires --map") {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func TestRunWithArgsPrintsLongFlagUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	for _, needle := range []string{"--root string", "--map", "--format string", "--version"} {
		if !strings.Contains(stderr.String(), needle) {
			t.Fatalf("expected usage to contain %q, got:\n%s", needle, stderr.String())
		}
	}
}

func cliFixture(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	cliWriteFile(t, root, "go.mod", "module example.com/app\n\ngo 1.26.4\n")
	for name, content := range files {
		cliWriteFile(t, root, name, content)
	}
	return root
}

func cliWriteFile(t *testing.T, root string, name string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
