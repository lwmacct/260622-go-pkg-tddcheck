package rulekit

import "testing"

func TestParseFileIdentity(t *testing.T) {
	tests := map[string]FileIdentity{
		"device_group.service.go": {Layer: "service", Subject: "device_group", Kind: "service"},
		"x.shared.free.go":        {Layer: "service", Namespace: "shared", Kind: "free"},
	}
	for base, want := range tests {
		got, ok := ParseFileIdentity("service", base, FileNameModeQualifiedKind)
		if !ok || got != want {
			t.Errorf("ParseFileIdentity(%q) = %#v, %v; want %#v, true", base, got, ok, want)
		}
	}
}

func TestParseFileIdentityRejectsNonCanonicalComponents(t *testing.T) {
	for _, base := range []string{
		"Device.service.go",
		"device-group.service.go",
		"device__group.service.go",
		"x.Shared.support.go",
		"device.service_kind.go",
	} {
		if got, ok := ParseFileIdentity("service", base, FileNameModeQualifiedKind); ok {
			t.Errorf("ParseFileIdentity(%q) = %#v, true; want false", base, got)
		}
	}
}

func TestParsePackageKindRequiresSingleAtom(t *testing.T) {
	if got, ok := ParseFileIdentity("adapter", "handler.go", FileNameModePackageKind); !ok || got.Kind != "handler" {
		t.Fatalf("ParseFileIdentity(handler.go) = %#v, %v", got, ok)
	}
	for _, base := range []string{"service.handler.go", "service_kind.go", "Handler.go"} {
		if got, ok := ParseFileIdentity("adapter", base, FileNameModePackageKind); ok {
			t.Errorf("ParseFileIdentity(%q) = %#v, true; want false", base, got)
		}
	}
}
