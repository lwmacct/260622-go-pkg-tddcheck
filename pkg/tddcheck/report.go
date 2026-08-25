package tddcheck

import (
	"fmt"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

type Diagnostic = rulekit.Diagnostic
type SuggestedFix = rulekit.SuggestedFix
type RenameFix = rulekit.RenameFix
type Severity = rulekit.Severity
type Position = rulekit.Position
type Range = rulekit.Range
type LoadError = rulekit.LoadError
type LoadErrorKind = rulekit.LoadErrorKind
type FileIdentity = rulekit.FileIdentity

const (
	SeverityError   = rulekit.SeverityError
	SeverityWarning = rulekit.SeverityWarning
	SeverityInfo    = rulekit.SeverityInfo
	LoadErrorList   = rulekit.LoadErrorList
	LoadErrorParse  = rulekit.LoadErrorParse
	LoadErrorType   = rulekit.LoadErrorType
)

func (a Analysis) Passed() bool {
	if len(a.LoadErrors) > 0 {
		return false
	}
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
	lines := make([]string, 0, len(a.Diagnostics)+len(a.LoadErrors)+1)
	lines = append(lines, "tddcheck: failed")
	for _, loadError := range a.LoadErrors {
		position := loadError.PackagePath
		if loadError.Position != "" {
			position += ":" + loadError.Position
		}
		lines = append(lines, fmt.Sprintf("%s [package/%s] %s", position, loadError.Kind, loadError.Message))
	}
	for _, diagnostic := range a.Diagnostics {
		lines = append(lines, diagnostic.String())
	}
	return strings.Join(lines, "\n")
}
