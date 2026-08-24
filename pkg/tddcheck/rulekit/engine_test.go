package rulekit

import (
	"context"
	"testing"
)

func TestEngineRegisterRunsTypedFileRuleAndSortsDiagnostics(t *testing.T) {
	snapshot := &Snapshot{Files: []GoFile{
		{Base: "z.go", RelPath: "z.go"},
		{Base: "a.go", RelPath: "a.go"},
	}}
	var engine Engine
	engine.Register("filename", FileScope, func(_ context.Context, _ *Snapshot, file GoFile) ([]Diagnostic, error) {
		position := Position{File: file.RelPath, Line: 1}
		return []Diagnostic{NewDiagnostic("", SeverityWarning, file.Base, position, position)}, nil
	})

	diagnostics, err := engine.Run(context.Background(), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 2 || diagnostics[0].Message != "a.go" || diagnostics[1].Message != "z.go" {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if diagnostics[0].RuleID != "filename" || diagnostics[0].Severity != SeverityWarning {
		t.Fatalf("rule metadata not normalized: %#v", diagnostics[0])
	}
}
