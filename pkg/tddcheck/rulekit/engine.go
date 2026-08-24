package rulekit

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"strings"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Position struct {
	File   string `json:"file"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	RuleID   string   `json:"ruleId"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Range    Range    `json:"range"`
}

func (d Diagnostic) String() string {
	location := d.Range.Start.File
	if d.Range.Start.Line > 0 {
		location = fmt.Sprintf("%s:%d", location, d.Range.Start.Line)
		if d.Range.Start.Column > 0 {
			location = fmt.Sprintf("%s:%d", location, d.Range.Start.Column)
		}
	}
	if location == "" {
		location = "-"
	}
	return fmt.Sprintf("%s [%s] %s", location, d.RuleID, d.Message)
}

func NewDiagnostic(ruleID string, severity Severity, message string, start Position, end Position) Diagnostic {
	if severity == "" {
		severity = SeverityError
	}
	if end.File == "" {
		end = start
	}
	return Diagnostic{
		RuleID:   ruleID,
		Severity: severity,
		Message:  message,
		Range:    Range{Start: start, End: end},
	}
}

// Scope enumerates the typed subjects inspected by a rule pass.
type Scope[T any] func(*Snapshot) iter.Seq[T]

// CheckFunc checks one subject. Returning diagnostics does not stop later
// subjects; returning an error stops the analysis.
type CheckFunc[T any] func(context.Context, *Snapshot, T) ([]Diagnostic, error)

type pass struct {
	id  string
	run func(context.Context, *Snapshot) ([]Diagnostic, error)
}

type Engine struct {
	passes []pass
}

// Register adds a strongly typed rule pass. Generic methods are available
// starting in Go 1.27; the stored closure keeps Engine.Run non-generic.
func (e *Engine) Register[T any](id string, scope Scope[T], check CheckFunc[T]) {
	if id == "" {
		panic("rule ID is empty")
	}
	if scope == nil || check == nil {
		panic("rule scope and check function must be non-nil")
	}
	e.passes = append(e.passes, pass{
		id: id,
		run: func(ctx context.Context, snapshot *Snapshot) ([]Diagnostic, error) {
			var diagnostics []Diagnostic
			for subject := range scope(snapshot) {
				if err := ctx.Err(); err != nil {
					return diagnostics, err
				}
				values, err := check(ctx, snapshot, subject)
				if err != nil {
					return diagnostics, fmt.Errorf("rule %s: %w", id, err)
				}
				for index := range values {
					if values[index].RuleID == "" {
						values[index].RuleID = id
					}
					if values[index].Severity == "" {
						values[index].Severity = SeverityError
					}
				}
				diagnostics = append(diagnostics, values...)
			}
			return diagnostics, nil
		},
	})
}

func (e *Engine) Run(ctx context.Context, snapshot *Snapshot) ([]Diagnostic, error) {
	var diagnostics []Diagnostic
	for _, rulePass := range e.passes {
		if err := ctx.Err(); err != nil {
			return diagnostics, err
		}
		values, err := rulePass.run(ctx, snapshot)
		if err != nil {
			return diagnostics, err
		}
		diagnostics = append(diagnostics, values...)
	}
	slices.SortFunc(diagnostics, compareDiagnostics)
	return diagnostics, nil
}

func SnapshotScope(snapshot *Snapshot) iter.Seq[*Snapshot] {
	return func(yield func(*Snapshot) bool) {
		yield(snapshot)
	}
}

func FileScope(snapshot *Snapshot) iter.Seq[GoFile] {
	return slices.Values(snapshot.Files)
}

func PackageScope(snapshot *Snapshot) iter.Seq[GoPackage] {
	return slices.Values(snapshot.Packages)
}

func compareDiagnostics(a Diagnostic, b Diagnostic) int {
	valuesA := []string{a.Range.Start.File, fmt.Sprintf("%09d", a.Range.Start.Line), fmt.Sprintf("%09d", a.Range.Start.Column), a.RuleID, string(a.Severity), a.Message}
	valuesB := []string{b.Range.Start.File, fmt.Sprintf("%09d", b.Range.Start.Line), fmt.Sprintf("%09d", b.Range.Start.Column), b.RuleID, string(b.Severity), b.Message}
	return strings.Compare(strings.Join(valuesA, "\x00"), strings.Join(valuesB, "\x00"))
}
