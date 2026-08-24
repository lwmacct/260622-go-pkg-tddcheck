package tddcheck

import (
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

type Diagnostic = rulekit.Diagnostic
type Severity = rulekit.Severity
type Position = rulekit.Position
type Range = rulekit.Range
type LoadError = rulekit.LoadError
type LoadErrorKind = rulekit.LoadErrorKind

const (
	SeverityError   = rulekit.SeverityError
	SeverityWarning = rulekit.SeverityWarning
	SeverityInfo    = rulekit.SeverityInfo
	LoadErrorList   = rulekit.LoadErrorList
	LoadErrorParse  = rulekit.LoadErrorParse
	LoadErrorType   = rulekit.LoadErrorType
)

func (a Analysis) Passed() bool {
	for _, diagnostic := range a.Diagnostics {
		if diagnostic.Severity == SeverityError {
			return false
		}
	}
	return true
}

func (a Analysis) ArchitectureIndex() Index {
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
	lines := make([]string, 0, len(a.Diagnostics)+1)
	lines = append(lines, "tddcheck: failed")
	for _, diagnostic := range a.Diagnostics {
		lines = append(lines, diagnostic.String())
	}
	return strings.Join(lines, "\n")
}
