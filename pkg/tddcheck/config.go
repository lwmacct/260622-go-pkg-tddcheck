package tddcheck

import "github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"

// Config controls project scanning, file-layout rules, and layer dependencies.
// Nil slice and map fields inherit their values from [DefaultConfig].
type Config = rulekit.Config

// LayerDependencyRule forbids imports from a matching source layer and path to
// a matching target layer and path.
type LayerDependencyRule = rulekit.LayerDependencyRule

const (
	// FileNameModeScopeKind requires filenames in {scope}.{kind}.go form.
	FileNameModeScopeKind = rulekit.FileNameModeScopeKind
	// FileNameModePackageKind requires filenames in {kind}.go form.
	FileNameModePackageKind = rulekit.FileNameModePackageKind
)

// DefaultConfig returns the handler, service, and repository architecture
// configuration supplied by tddcheck.
func DefaultConfig() Config {
	return rulekit.DefaultConfig()
}
