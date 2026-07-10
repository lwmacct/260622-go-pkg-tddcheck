package tddcheck

import (
	"fmt"
	"strings"
	"time"
)

// Violation describes one failed architecture rule.
type Violation struct {
	Rule    string
	File    string
	Line    int
	Message string
}

// String formats the violation as a source location, rule ID, and message.
func (v Violation) String() string {
	location := v.File
	if v.Line > 0 {
		location = fmt.Sprintf("%s:%d", v.File, v.Line)
	}
	if location == "" {
		location = "-"
	}
	return fmt.Sprintf("%s [%s] %s", location, v.Rule, v.Message)
}

// Result is the compact check result used by text-oriented integrations.
type Result struct {
	// Passed reports whether the check completed without violations.
	Passed bool
	// Err is an operational failure that prevented a completed check.
	Err error
	// Violations contains failed architecture rules.
	Violations []Violation
	// Duration is the elapsed check time.
	Duration time.Duration
}

// Passed reports whether the analysis contains no rule violations.
func (a Analysis) Passed() bool {
	return len(a.Violations) == 0
}

// Result returns a compact representation of the completed analysis.
func (a Analysis) Result() Result {
	return Result{
		Passed:     a.Passed(),
		Violations: a.Violations,
		Duration:   a.Duration,
	}
}

// ProjectIndex returns the analysis index with root and module metadata filled
// from the containing Analysis when necessary.
func (a Analysis) ProjectIndex() Index {
	index := a.Index
	if index.Root == "" {
		index.Root = a.Root
	}
	if index.ModulePath == "" {
		index.ModulePath = a.ModulePath
	}
	if index.projectRoot == "" {
		index.projectRoot = a.projectRoot
	}
	return index
}

// Text formats the check status and all violations for terminal output.
func (a Analysis) Text() string {
	if a.Passed() {
		return "tddcheck: passed"
	}

	lines := make([]string, 0, len(a.Violations)+1)
	lines = append(lines, "tddcheck: failed")
	for _, violation := range a.Violations {
		lines = append(lines, violation.String())
	}
	return strings.Join(lines, "\n")
}

// Text formats an operational error or the check status and violations.
func (r Result) Text() string {
	if r.Err != nil {
		return "tddcheck: " + r.Err.Error()
	}
	return Analysis{Violations: r.Violations}.Text()
}
