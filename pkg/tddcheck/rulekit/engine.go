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

// RenameFix describes an exact file rename.
type RenameFix struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// SuggestedFix describes a machine-readable correction for a diagnostic.
type SuggestedFix struct {
	Message string     `json:"message"`
	Rename  *RenameFix `json:"rename,omitempty"`
}

// Diagnostic is one stable, source-located rule result.
type Diagnostic struct {
	RuleID       string        `json:"ruleId"`
	Code         string        `json:"code"`
	Severity     Severity      `json:"severity"`
	Message      string        `json:"message"`
	Range        Range         `json:"range"`
	SuggestedFix *SuggestedFix `json:"suggestedFix,omitempty"`
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
	label := d.Code
	if label == "" {
		label = d.RuleID
	}
	return fmt.Sprintf("%s [%s] %s", location, label, d.Message)
}

// NewDiagnostic constructs a normalized source diagnostic.
func NewDiagnostic(ruleID string, code string, severity Severity, message string, start Position, end Position) Diagnostic {
	if severity == "" {
		severity = SeverityError
	}
	if end.File == "" {
		end = start
	}
	return Diagnostic{
		RuleID:   ruleID,
		Code:     code,
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
					if values[index].Code == "" {
						values[index].Code = values[index].RuleID
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
	valuesA := []string{a.Range.Start.File, fmt.Sprintf("%09d", a.Range.Start.Line), fmt.Sprintf("%09d", a.Range.Start.Column), a.RuleID, a.Code, string(a.Severity), a.Message}
	valuesB := []string{b.Range.Start.File, fmt.Sprintf("%09d", b.Range.Start.Line), fmt.Sprintf("%09d", b.Range.Start.Column), b.RuleID, b.Code, string(b.Severity), b.Message}
	return strings.Compare(strings.Join(valuesA, "\x00"), strings.Join(valuesB, "\x00"))
}
