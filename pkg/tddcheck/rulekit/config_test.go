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
	if compiled.LayerDirs[0] != "handler" || compiled.LayerFileKinds["handler"][0] != "support" {
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
