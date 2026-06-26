package tddcheck

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rules/filelayout"
	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rules/layerdeps"
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

func (r Result) Text() string {
	if r.Err != nil {
		return "tddcheck: " + r.Err.Error()
	}
	if len(r.Violations) == 0 {
		return "tddcheck: passed"
	}

	lines := make([]string, 0, len(r.Violations)+1)
	lines = append(lines, "tddcheck: failed")
	for _, violation := range r.Violations {
		lines = append(lines, violation.String())
	}
	return strings.Join(lines, "\n")
}

type ProjectRules struct {
	Root   string
	Config Config
}

func (r ProjectRules) Assert(tb testing.TB) {
	tb.Helper()

	result := r.Check()
	if result.Err != nil {
		tb.Fatal(result.Err)
	}
	if !result.Passed {
		tb.Fatal(result.Text())
	}
}

func (r ProjectRules) Check() Result {
	start := time.Now()
	root := r.Root
	if root == "" {
		root = "internal"
	}
	config := r.Config

	var violations []Violation
	context, err := rulekit.NewContext(root, "project", config)
	if err != nil {
		return resultError(err, violations, start)
	}
	for _, rule := range defaultRules() {
		values, err := rule.Check(context)
		if err != nil {
			return resultError(err, violations, start)
		}
		for _, value := range values {
			violations = append(violations, Violation{
				Rule:    value.Rule,
				File:    value.File,
				Line:    value.Line,
				Message: value.Message,
			})
		}
	}

	slices.SortFunc(violations, func(a, b Violation) int {
		return strings.Compare(a.String(), b.String())
	})
	return Result{
		Passed:     len(violations) == 0,
		Violations: violations,
		Duration:   time.Since(start),
	}
}

func defaultRules() []rulekit.Rule {
	return []rulekit.Rule{
		filelayout.New(""),
		layerdeps.New(""),
	}
}

func resultError(err error, violations []Violation, start time.Time) Result {
	return Result{Passed: false, Err: err, Violations: violations, Duration: time.Since(start)}
}
