package filelayout

import (
	"path/filepath"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func TestUpperCamelNamePreservesCommonInitialisms(t *testing.T) {
	got := upperCamelName("node_ws_tunnel_llm")
	if got != "NodeWSTunnelLLM" {
		t.Fatalf("upperCamelName() = %q, want NodeWSTunnelLLM", got)
	}
}

func TestSnakeNamePreservesCommonInitialisms(t *testing.T) {
	tests := map[string]string{
		"IdentitySSHKey": "identity_ssh_key",
		"NodeWSTunnel":   "node_ws_tunnel",
		"LLMProvider":    "llm_provider",
	}
	for input, want := range tests {
		if got := snakeName(input); got != want {
			t.Fatalf("snakeName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestViolationsRejectsWeakSubjectAndLegacyNames(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/default.service.go": `package service
func NewDefaultService() {}
`,
		"internal/service/device_validation.go": `package service
func validateDevice() {}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "subject \"default\" is too weak")
	assertViolationContains(t, violations, "must use {subject}.{kind}.go or x.{namespace}.{kind}.go")
}

func TestParseFileNameSeparatesArchitectureNamespace(t *testing.T) {
	tests := map[string]fileName{
		"device_group.service.go": {subject: "device_group", kind: "service"},
		"device.free.go":          {subject: "device", kind: "free"},
		"x.http.endpoint.go":      {namespace: "http", kind: "endpoint"},
		"x.shared.free.go":        {namespace: "shared", kind: "free"},
	}
	for base, want := range tests {
		got, ok := parseFileName(base, rulekit.FileNameModeQualifiedKind)
		if !ok || got != want {
			t.Errorf("parseFileName(%q) = %#v, %v; want %#v, true", base, got, ok, want)
		}
	}
}

func TestParseFileNameRejectsMalformedArchitectureNames(t *testing.T) {
	for _, base := range []string{"x.shared.go", "x.shared.extra.support.go", "x..support.go"} {
		if got, ok := parseFileName(base, rulekit.FileNameModeQualifiedKind); ok {
			t.Errorf("parseFileName(%q) = %#v, true; want false", base, got)
		}
	}
}
