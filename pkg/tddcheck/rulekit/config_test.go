package rulekit

import "testing"

func TestCompileDeepCopiesConfiguration(t *testing.T) {
	input := DefaultConfig()
	compiled, err := input.Compile()
	if err != nil {
		t.Fatal(err)
	}
	input.LayerDirs[0] = "changed"
	input.LayerKindPolicies["handler"]["free"] = "changed"
	input.LayerSubjectAnchorKinds["service"] = "changed"
	if compiled.LayerDirs[0] != "handler" ||
		compiled.LayerKindPolicies["handler"]["free"] != "free" ||
		compiled.LayerSubjectAnchorKinds["service"] != "service" {
		t.Fatalf("compiled config retained caller-owned data: %#v", compiled)
	}
}

func TestValidateFileLayoutRejectsInvalidSubjectAnchor(t *testing.T) {
	tests := []struct {
		name   string
		mode   string
		anchor string
	}{
		{name: "package kind", mode: FileNameModePackageKind, anchor: "doc"},
		{name: "unknown kind", mode: FileNameModeQualifiedKind, anchor: "missing"},
		{name: "free", mode: FileNameModeQualifiedKind, anchor: "free"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := Config{
				LayerDirs:               []string{"adapter"},
				LayerRules:              []LayerDependencyRule{},
				LayerFileNameModes:      map[string]string{"adapter": test.mode},
				LayerKindPolicies:       map[string]map[string]string{"adapter": {"doc": "free", "free": "free"}},
				LayerSubjectAnchorKinds: map[string]string{"adapter": test.anchor},
				ArchitectureNamespaces:  map[string][]string{},
				EscapedSubjectSuffixes:  []string{},
				ForbiddenWeakSubjects:   []string{},
			}
			compiled, err := config.Compile()
			if err != nil {
				t.Fatal(err)
			}
			if err := compiled.ValidateFileLayout(); err == nil {
				t.Fatal("expected invalid subject anchor to fail")
			}
		})
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

func TestCompileRejectsNegativeSupportDeclarationLimit(t *testing.T) {
	config := DefaultConfig()
	config.MaxSupportDeclarationLines = -1
	if _, err := config.Compile(); err == nil {
		t.Fatal("expected negative support declaration limit to fail")
	}
}
