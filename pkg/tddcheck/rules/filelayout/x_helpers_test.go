package filelayout

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

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

func checkRoot(root string, configs ...rulekit.Config) ([]Violation, error) {
	var config rulekit.Config
	if len(configs) > 0 {
		config = configs[0]
	}
	return Check(context.Background(), root, config)
}

func assertViolationContains(t *testing.T, violations []Violation, needle string) {
	t.Helper()

	for _, violation := range violations {
		if strings.Contains(violation.Message, needle) {
			return
		}
	}
	t.Fatalf("expected violation containing %q, got %#v", needle, violations)
}

func assertNoViolationContains(t *testing.T, violations []Violation, needle string) {
	t.Helper()

	for _, violation := range violations {
		if strings.Contains(violation.Message, needle) {
			t.Fatalf("unexpected violation containing %q: %#v", needle, violations)
		}
	}
}
