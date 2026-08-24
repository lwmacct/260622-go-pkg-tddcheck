package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithArgsIndexCommand(t *testing.T) {
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
	code := runWithArgs(context.Background(), []string{"index", "--root", filepath.Join(root, "internal")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr:\n%s", code, stderr.String())
	}
	for _, needle := range []string{
		"tddcheck index: example.com/app",
		"- DeviceService (device)",
		"- devices (DeviceModel, device)",
	} {
		if !strings.Contains(stdout.String(), needle) {
			t.Fatalf("expected stdout to contain %q, got:\n%s", needle, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "apis:") {
		t.Fatalf("expected text index not to contain APIs, got:\n%s", stdout.String())
	}
}

func TestRunWithArgsIndexCommandJSON(t *testing.T) {
	root := cliFixture(t, map[string]string{
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs(context.Background(), []string{"index", "--root", filepath.Join(root, "internal"), "--format", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"modulePath": "example.com/app"`) {
		t.Fatalf("expected json module path, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"schemaVersion": "1"`) {
		t.Fatalf("expected versioned json schema, got:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), `"duration"`) {
		t.Fatalf("expected deterministic json without duration, got:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), `"apis"`) {
		t.Fatalf("expected json index not to contain APIs, got:\n%s", stdout.String())
	}
}

func TestRunWithArgsDocCommand(t *testing.T) {
	root := cliFixture(t, map[string]string{
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
	})
	output := filepath.Join(root, "docs", "index.md")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs(context.Background(), []string{"doc", "--root", filepath.Join(root, "internal"), "--output", output}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr:\n%s", code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# tddcheck Architecture Index") {
		t.Fatalf("expected generated doc, got:\n%s", string(data))
	}
}

func TestRunWithArgsRejectsShortFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs(context.Background(), []string{"index", "-root", "internal"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "short flags are not supported: -root") {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func TestRunWithArgsRejectsUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs(context.Background(), []string{"unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command unknown") {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func TestRunWithArgsRejectsUnknownConfigFields(t *testing.T) {
	root := cliFixture(t, map[string]string{
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
	})
	configFile := filepath.Join(root, "tddcheck.json")
	cliWriteFile(t, root, "tddcheck.json", `{"unknownField":true}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs(context.Background(), []string{"check", "--root", filepath.Join(root, "internal"), "--config", configFile}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknownField") {
		t.Fatalf("expected strict config error, got:\n%s", stderr.String())
	}
}

func TestRunWithArgsPrintsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithArgs(context.Background(), []string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	for _, needle := range []string{"check", "index", "doc", "version"} {
		if !strings.Contains(stderr.String(), needle) {
			t.Fatalf("expected usage to contain %q, got:\n%s", needle, stderr.String())
		}
	}
}

func cliFixture(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	cliWriteFile(t, root, "go.mod", "module example.com/app\n\ngo 1.27.0\n")
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
