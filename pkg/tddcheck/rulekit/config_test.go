package rulekit

import "testing"

func TestCompileDeepCopiesConfiguration(t *testing.T) {
	input := DefaultConfig()
	compiled, err := input.Compile()
	if err != nil {
		t.Fatal(err)
	}
	input.LayerDirs[0] = "changed"
	input.LayerFileKinds["handler"][0] = "changed"
	if compiled.LayerDirs[0] != "handler" || compiled.LayerFileKinds["handler"][0] != "free" {
		t.Fatalf("compiled config retained caller-owned data: %#v", compiled)
	}
}

func TestValidateFileLayoutRejectsIncompleteCustomLayer(t *testing.T) {
	compiled, err := (Config{LayerDirs: []string{"adapter"}}).Compile()
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.ValidateFileLayout(); err == nil {
		t.Fatal("expected custom layer without filename configuration to fail")
	}
}

func TestValidateFileLayoutRejectsNamespacedArchitectureNamespace(t *testing.T) {
	for _, namespace := range []string{"x.shared", "x_shared"} {
		config := DefaultConfig()
		config.ArchitectureNamespaces["handler"] = []string{namespace}
		compiled, err := config.Compile()
		if err != nil {
			t.Fatal(err)
		}
		if err := compiled.ValidateFileLayout(); err == nil {
			t.Errorf("expected architecture namespace %q to fail", namespace)
		}
	}
}

func TestCompileRejectsLegacyScopeKindMode(t *testing.T) {
	config := DefaultConfig()
	config.LayerFileNameModes["handler"] = "scope_kind"
	if _, err := config.Compile(); err == nil {
		t.Fatal("expected legacy scope_kind mode to fail")
	}
}
