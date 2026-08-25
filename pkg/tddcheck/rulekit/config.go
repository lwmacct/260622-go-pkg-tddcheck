package rulekit

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Config defines the directories and conventions used by architecture rules.
// Each nil slice or map is independently populated by [Config.WithDefaults].
type Config struct {
	// IncludeTests loads package test variants and *_test.go files.
	IncludeTests bool `json:"includeTests,omitempty"`
	// BuildFlags are forwarded to the Go package loader, for example -tags=integration.
	BuildFlags []string `json:"buildFlags,omitempty"`
	// StrictPackages turns package-list and type-checking errors into analysis errors.
	StrictPackages bool `json:"strictPackages,omitempty"`

	// LayerDirs lists directory names checked by file-layout rules.
	LayerDirs []string `json:"layerDirs,omitempty"`
	// DependencyLayerDirs lists directory names recognized by dependency rules.
	// A nil value inherits LayerDirs.
	DependencyLayerDirs []string `json:"dependencyLayerDirs,omitempty"`
	// SkipDirs lists directory names excluded from source scanning.
	SkipDirs []string `json:"skipDirs,omitempty"`
	// LayerRules lists forbidden import relationships.
	LayerRules []LayerDependencyRule `json:"layerRules,omitempty"`

	// LayerFileNameModes maps a layer to a FileNameMode constant.
	LayerFileNameModes map[string]string `json:"layerFileNameModes,omitempty"`
	// LayerKindPolicies maps each allowed filename kind to an explicit
	// declaration policy ID within its layer.
	LayerKindPolicies map[string]map[string]string `json:"layerKindPolicies,omitempty"`
	// LayerSubjectAnchorKinds requires every business subject in a qualified-kind
	// layer to declare the configured anchor kind.
	LayerSubjectAnchorKinds map[string]string `json:"layerSubjectAnchorKinds,omitempty"`
	// ArchitectureNamespaces maps a layer to its allowed architecture namespaces.
	ArchitectureNamespaces map[string][]string `json:"architectureNamespaces,omitempty"`
	// EscapedSubjectSuffixes lists kinds and actions that business subjects must not
	// encode as suffixes.
	EscapedSubjectSuffixes []string `json:"escapedSubjectSuffixes,omitempty"`
	// ForbiddenWeakSubjects lists ambiguous business subject names.
	ForbiddenWeakSubjects []string `json:"forbiddenWeakSubjects,omitempty"`
	// PublicTypeBoundarySuffixes lists repository type suffixes that must not
	// appear in exported service method signatures. Nil inherits defaults;
	// an explicit empty slice disables this boundary check.
	PublicTypeBoundarySuffixes []string `json:"publicTypeBoundarySuffixes,omitempty"`
	// MaxSupportDeclarationLines limits the total lines occupied by type, const,
	// and var declarations in a support file. Zero disables the limit.
	MaxSupportDeclarationLines int `json:"maxSupportDeclarationLines,omitempty"`
}

type Profile struct {
	Layers                     []LayerProfile
	DependencyLayers           []string
	SkipDirs                   []string
	LayerRules                 []LayerDependencyRule
	EscapedSubjectSuffixes     []string
	ForbiddenWeakSubjects      []string
	PublicTypeBoundarySuffixes []string
	MaxSupportDeclarationLines int

	layersByName             map[string]LayerProfile
	dependencyLayerSet       map[string]bool
	kindPolicyByLayer        map[string]map[string]string
	architectureNamespaceSet map[string]map[string]bool
}

type LayerProfile struct {
	Name                   string
	FileNameMode           string
	KindPolicies           map[string]string
	SubjectAnchorKind      string
	ArchitectureNamespaces []string
}

// LayerDependencyRule describes a forbidden import relationship. Source path
// and target prefixes are both relative to the analyzed root.
type LayerDependencyRule struct {
	// SourceLayer is the layer containing the importing file.
	SourceLayer string `json:"sourceLayer"`
	// SourceRelPrefix optionally restricts importing directories relative to the
	// analyzed root.
	SourceRelPrefix string `json:"sourceRelPrefix,omitempty"`
	// ExceptSourceRelPrefixes exempts importing directories relative to the
	// analyzed root.
	ExceptSourceRelPrefixes []string `json:"exceptSourceRelPrefixes,omitempty"`
	// TargetLayer is the layer containing the imported package.
	TargetLayer string `json:"targetLayer"`
	// TargetRelPrefix optionally restricts imported packages relative to the
	// analyzed root.
	TargetRelPrefix string `json:"targetRelPrefix,omitempty"`
	// ExceptTargetRelPrefixes exempts imported packages relative to the analyzed
	// root.
	ExceptTargetRelPrefixes []string `json:"exceptTargetRelPrefixes,omitempty"`
	// Message overrides the default violation message when non-empty.
	Message string `json:"message,omitempty"`
}

const (
	FileNameModeQualifiedKind = "qualified_kind"
	FileNameModePackageKind   = "package_kind"
)

func DefaultConfig() Config {
	return Config{
		LayerDirs: []string{"handler", "service", "repository"},
		SkipDirs:  []string{".git", ".hg", ".svn", "vendor", "node_modules", "dist", "build"},
		LayerRules: []LayerDependencyRule{
			{SourceLayer: "handler", TargetLayer: "repository", Message: "handler must not import repository"},
			{SourceLayer: "service", TargetLayer: "handler", Message: "service must not import handler"},
			{SourceLayer: "repository", TargetLayer: "handler", Message: "repository must not import handler"},
			{SourceLayer: "repository", TargetLayer: "service", Message: "repository must not import service"},
		},
		LayerFileNameModes: map[string]string{
			"handler":    FileNameModeQualifiedKind,
			"service":    FileNameModeQualifiedKind,
			"repository": FileNameModeQualifiedKind,
		},
		// Keep shared/cross-layer entries before layer-specific entries.
		LayerKindPolicies: map[string]map[string]string{
			"handler": {
				"free": "free", "support": "support", "types": "types", "mapper": "mapper",
				"context": "context", "dto": "dto", "endpoint": "endpoint",
				"handler": "handler", "middleware": "middleware", "utils": "utils",
			},
			"service": {
				"free": "free", "support": "support", "types": "types", "mapper": "mapper",
				"commands": "commands", "provider": "provider", "service": "service",
			},
			"repository": {
				"free": "free", "support": "support", "types": "types", "repository": "repository",
				"schema": "schema", "store": "store",
			},
		},
		LayerSubjectAnchorKinds: map[string]string{
			"service": "service",
		},
		ArchitectureNamespaces: map[string][]string{
			"handler":    {"shared", "http"},
			"service":    {"shared"},
			"repository": {"shared", "store"},
		},
		EscapedSubjectSuffixes: []string{
			"support",
			"mapper",
			"service",
			"repository",
			"store",
			"handler",
			"dto",
			"context",
			"provider",
			"schema",
			"utils",
			"commands",
			"types",
			"constants",
			"errors",
			"validation",
			"create",
			"delete",
			"list",
			"patch",
			"update",
			"upsert",
		},
		ForbiddenWeakSubjects:      []string{"common", "default", "helper", "helpers", "misc", "util", "utils"},
		PublicTypeBoundarySuffixes: []string{"Model", "Row", "Patch", "Create", "Filter"},
	}
}

func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()
	if c.LayerDirs == nil {
		c.LayerDirs = defaults.LayerDirs
	}
	if c.DependencyLayerDirs == nil {
		c.DependencyLayerDirs = c.LayerDirs
	}
	if c.SkipDirs == nil {
		c.SkipDirs = defaults.SkipDirs
	}
	if c.LayerRules == nil {
		c.LayerRules = defaults.LayerRules
	}
	if c.LayerFileNameModes == nil {
		c.LayerFileNameModes = defaults.LayerFileNameModes
	}
	if c.LayerKindPolicies == nil {
		c.LayerKindPolicies = defaults.LayerKindPolicies
	}
	if c.LayerSubjectAnchorKinds == nil {
		c.LayerSubjectAnchorKinds = defaults.LayerSubjectAnchorKinds
	}
	if c.ArchitectureNamespaces == nil {
		c.ArchitectureNamespaces = defaults.ArchitectureNamespaces
	}
	if c.EscapedSubjectSuffixes == nil {
		c.EscapedSubjectSuffixes = defaults.EscapedSubjectSuffixes
	}
	if c.ForbiddenWeakSubjects == nil {
		c.ForbiddenWeakSubjects = defaults.ForbiddenWeakSubjects
	}
	if c.PublicTypeBoundarySuffixes == nil {
		c.PublicTypeBoundarySuffixes = defaults.PublicTypeBoundarySuffixes
	}
	return c
}

// Compile applies defaults, validates cross-field references, and deep-clones
// caller-owned collections so an analysis cannot observe later mutations.
func (c Config) Compile() (Config, error) {
	inheritedLayerRules := c.LayerRules == nil
	inheritedSubjectAnchors := c.LayerSubjectAnchorKinds == nil
	c = c.WithDefaults()
	if inheritedLayerRules {
		layers := sliceSet(c.DependencyLayerDirs)
		c.LayerRules = slices.DeleteFunc(c.LayerRules, func(rule LayerDependencyRule) bool {
			return !layers[rule.SourceLayer] || !layers[rule.TargetLayer]
		})
	}
	if inheritedSubjectAnchors {
		layers := sliceSet(c.LayerDirs)
		maps.DeleteFunc(c.LayerSubjectAnchorKinds, func(layer string, _ string) bool {
			return !layers[layer]
		})
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	c.BuildFlags = slices.Clone(c.BuildFlags)
	c.LayerDirs = slices.Clone(c.LayerDirs)
	c.DependencyLayerDirs = slices.Clone(c.DependencyLayerDirs)
	c.SkipDirs = slices.Clone(c.SkipDirs)
	c.LayerRules = slices.Clone(c.LayerRules)
	for index := range c.LayerRules {
		c.LayerRules[index].ExceptSourceRelPrefixes = slices.Clone(c.LayerRules[index].ExceptSourceRelPrefixes)
		c.LayerRules[index].ExceptTargetRelPrefixes = slices.Clone(c.LayerRules[index].ExceptTargetRelPrefixes)
	}
	c.LayerFileNameModes = maps.Clone(c.LayerFileNameModes)
	c.LayerKindPolicies = cloneStringMaps(c.LayerKindPolicies)
	c.LayerSubjectAnchorKinds = maps.Clone(c.LayerSubjectAnchorKinds)
	c.ArchitectureNamespaces = cloneStringSlices(c.ArchitectureNamespaces)
	c.EscapedSubjectSuffixes = slices.Clone(c.EscapedSubjectSuffixes)
	c.ForbiddenWeakSubjects = slices.Clone(c.ForbiddenWeakSubjects)
	c.PublicTypeBoundarySuffixes = slices.Clone(c.PublicTypeBoundarySuffixes)
	return c, nil
}

func (c Config) Validate() error {
	if err := validateNames("layer", c.LayerDirs); err != nil {
		return err
	}
	if err := validateNames("dependency layer", c.DependencyLayerDirs); err != nil {
		return err
	}
	if err := validateNames("skip directory", c.SkipDirs); err != nil {
		return err
	}
	if c.MaxSupportDeclarationLines < 0 {
		return fmt.Errorf("max support declaration lines must not be negative")
	}
	dependencyLayers := sliceSet(c.DependencyLayerDirs)
	for _, layer := range c.LayerDirs {
		mode := c.LayerFileNameModes[layer]
		if mode != "" && mode != FileNameModeQualifiedKind && mode != FileNameModePackageKind {
			return fmt.Errorf("layer %q has invalid filename mode %q", layer, mode)
		}
	}
	for _, rule := range c.LayerRules {
		if !dependencyLayers[rule.SourceLayer] {
			return fmt.Errorf("dependency rule references unknown source layer %q", rule.SourceLayer)
		}
		if !dependencyLayers[rule.TargetLayer] {
			return fmt.Errorf("dependency rule references unknown target layer %q", rule.TargetLayer)
		}
	}
	return nil
}

// ValidateFileLayout checks the profile fields consumed by file-layout rules.
// Dependency-only tools may intentionally omit these fields and use Validate.
func (c Config) ValidateFileLayout() error {
	layers := sliceSet(c.LayerDirs)
	for _, layer := range c.LayerDirs {
		mode := c.LayerFileNameModes[layer]
		if mode != FileNameModeQualifiedKind && mode != FileNameModePackageKind {
			return fmt.Errorf("layer %q has invalid filename mode %q", layer, mode)
		}
		if len(c.LayerKindPolicies[layer]) == 0 {
			return fmt.Errorf("layer %q has no kind policies", layer)
		}
	}
	for layer := range c.LayerFileNameModes {
		if !layers[layer] {
			return fmt.Errorf("filename mode references unknown layer %q", layer)
		}
	}
	for layer, policies := range c.LayerKindPolicies {
		if !layers[layer] {
			return fmt.Errorf("kind policies reference unknown layer %q", layer)
		}
		for kind, policy := range policies {
			if !validFileAtom(kind) {
				return fmt.Errorf("layer %q has invalid file kind %q", layer, kind)
			}
			if policy == "" {
				return fmt.Errorf("layer %q file kind %q has an empty policy", layer, kind)
			}
		}
	}
	for layer, anchorKind := range c.LayerSubjectAnchorKinds {
		if !layers[layer] {
			return fmt.Errorf("subject anchor references unknown layer %q", layer)
		}
		if c.LayerFileNameModes[layer] != FileNameModeQualifiedKind {
			return fmt.Errorf("layer %q cannot define subject anchor %q in package-kind mode", layer, anchorKind)
		}
		if anchorKind == "free" {
			return fmt.Errorf("layer %q cannot use free as its subject anchor", layer)
		}
		if _, ok := c.LayerKindPolicies[layer][anchorKind]; !ok {
			return fmt.Errorf("layer %q subject anchor %q is not an allowed kind", layer, anchorKind)
		}
	}
	for layer := range c.ArchitectureNamespaces {
		if !layers[layer] {
			return fmt.Errorf("architecture namespaces reference unknown layer %q", layer)
		}
		if err := validateFileComponents("architecture namespace", c.ArchitectureNamespaces[layer]); err != nil {
			return fmt.Errorf("layer %q: %w", layer, err)
		}
	}
	for _, suffix := range c.PublicTypeBoundarySuffixes {
		if suffix == "" {
			return fmt.Errorf("public type boundary suffix must not be empty")
		}
	}
	return nil
}

func validateFileComponents(label string, values []string) error {
	if err := validateNames(label, values); err != nil {
		return err
	}
	for _, value := range values {
		if strings.Contains(value, ".") {
			return fmt.Errorf("%s %q must be a single filename component", label, value)
		}
		if strings.HasPrefix(value, "x_") {
			return fmt.Errorf("%s %q must omit the legacy x_ prefix", label, value)
		}
	}
	return nil
}

func validateNames(label string, values []string) error {
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || strings.Contains(value, "/") || strings.Contains(value, `\`) {
			return fmt.Errorf("%s name %q must be a single non-empty path segment", label, value)
		}
		if seen[value] {
			return fmt.Errorf("duplicate %s %q", label, value)
		}
		seen[value] = true
	}
	return nil
}

func sliceSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func cloneStringSlices(values map[string][]string) map[string][]string {
	result := make(map[string][]string, len(values))
	for key, value := range values {
		result[key] = slices.Clone(value)
	}
	return result
}

func cloneStringMaps(values map[string]map[string]string) map[string]map[string]string {
	result := make(map[string]map[string]string, len(values))
	for key, value := range values {
		result[key] = maps.Clone(value)
	}
	return result
}

func (c Config) Profile() Profile {
	c = c.WithDefaults()
	layers := make([]LayerProfile, 0, len(c.LayerDirs))
	layersByName := make(map[string]LayerProfile, len(c.LayerDirs))
	kindPolicies := make(map[string]map[string]string, len(c.LayerDirs))
	architectureNamespaces := make(map[string]map[string]bool, len(c.LayerDirs))
	for _, layer := range c.LayerDirs {
		mode := c.LayerFileNameModes[layer]
		if mode == "" {
			mode = FileNameModeQualifiedKind
		}
		layerProfile := LayerProfile{
			Name:                   layer,
			FileNameMode:           mode,
			KindPolicies:           c.LayerKindPolicies[layer],
			SubjectAnchorKind:      c.LayerSubjectAnchorKinds[layer],
			ArchitectureNamespaces: c.ArchitectureNamespaces[layer],
		}
		layers = append(layers, layerProfile)
		layersByName[layer] = layerProfile
		kindPolicies[layer] = maps.Clone(layerProfile.KindPolicies)
		architectureNamespaces[layer] = sliceSet(layerProfile.ArchitectureNamespaces)
	}
	return Profile{
		Layers:                     layers,
		DependencyLayers:           c.DependencyLayerDirs,
		SkipDirs:                   c.SkipDirs,
		LayerRules:                 c.LayerRules,
		EscapedSubjectSuffixes:     c.EscapedSubjectSuffixes,
		ForbiddenWeakSubjects:      c.ForbiddenWeakSubjects,
		PublicTypeBoundarySuffixes: slices.Clone(c.PublicTypeBoundarySuffixes),
		MaxSupportDeclarationLines: c.MaxSupportDeclarationLines,
		layersByName:               layersByName,
		dependencyLayerSet:         sliceSet(c.DependencyLayerDirs),
		kindPolicyByLayer:          kindPolicies,
		architectureNamespaceSet:   architectureNamespaces,
	}
}

func (p Profile) Layer(name string) (LayerProfile, bool) {
	layer, ok := p.layersByName[name]
	return layer, ok
}

func (p Profile) LayerNames() []string {
	names := make([]string, 0, len(p.Layers))
	for _, layer := range p.Layers {
		names = append(names, layer.Name)
	}
	return names
}

func (p Profile) DependencyLayer(name string) bool {
	return p.dependencyLayerSet[name]
}

func (p Profile) KindAllowed(layerName string, kind string) bool {
	_, ok := p.kindPolicyByLayer[layerName][kind]
	return ok
}

func (p Profile) KindPolicy(layerName string, kind string) (string, bool) {
	policy, ok := p.kindPolicyByLayer[layerName][kind]
	return policy, ok
}

func (p Profile) ArchitectureNamespaceAllowed(layerName string, namespace string) bool {
	return p.architectureNamespaceSet[layerName][namespace]
}

func StringIn(value string, values []string) bool {
	return slices.Contains(values, value)
}
