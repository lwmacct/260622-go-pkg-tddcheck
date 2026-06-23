package filelayout

import (
	"path/filepath"
	"testing"
)

func TestUpperCamelNamePreservesCommonInitialisms(t *testing.T) {
	got := upperCamelName("node_ws_tunnel")
	if got != "NodeWSTunnel" {
		t.Fatalf("upperCamelName() = %q, want NodeWSTunnel", got)
	}
}

func TestSnakeNamePreservesCommonInitialisms(t *testing.T) {
	tests := map[string]string{
		"IdentitySSHKey": "identity_ssh_key",
		"NodeWSTunnel":   "node_ws_tunnel",
	}
	for input, want := range tests {
		if got := snakeName(input); got != want {
			t.Fatalf("snakeName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestViolationsRejectsWeakScopeAndLegacyNames(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/default.service.go": `package service
func NewDefaultService() {}
`,
		"internal/service/device_validation.go": `package service
func validateDevice() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "scope \"default\" is too weak")
	assertViolationContains(t, violations, "must use {scope}.{type}.go")
}
