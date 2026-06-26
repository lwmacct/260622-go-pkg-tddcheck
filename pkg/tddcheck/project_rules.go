package tddcheck

import (
	"fmt"
	"strings"
	"time"
)

type Violation struct {
	Rule    string
	File    string
	Line    int
	Message string
}

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

type Result struct {
	Passed     bool
	Err        error
	Violations []Violation
	Duration   time.Duration
}

func (a Analysis) Passed() bool {
	return len(a.Violations) == 0
}

func (a Analysis) Result() Result {
	return Result{
		Passed:     a.Passed(),
		Violations: a.Violations,
		Duration:   a.Duration,
	}
}

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

func (r Result) Text() string {
	if r.Err != nil {
		return "tddcheck: " + r.Err.Error()
	}
	return Analysis{Violations: r.Violations}.Text()
}

func resultError(err error, violations []Violation, start time.Time) Result {
	return Result{Passed: false, Err: err, Violations: violations, Duration: time.Since(start)}
}
