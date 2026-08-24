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
	// LayerFileKinds maps a layer to its allowed filename kinds.
	LayerFileKinds map[string][]string `json:"layerFileKinds,omitempty"`
	// ArchitectureScopes maps a layer to its allowed x_ scopes.
	ArchitectureScopes map[string][]string `json:"architectureScopes,omitempty"`
	// EscapedScopeSuffixes lists kinds and actions that business scopes must not
	// encode as suffixes.
	EscapedScopeSuffixes []string `json:"escapedScopeSuffixes,omitempty"`
	// ForbiddenWeakScopes lists ambiguous business scope names.
	ForbiddenWeakScopes []string `json:"forbiddenWeakScopes,omitempty"`
}

type Profile struct {
	Layers               []LayerProfile
	DependencyLayers     []string
	SkipDirs             []string
	LayerRules           []LayerDependencyRule
	EscapedScopeSuffixes []string
	ForbiddenWeakScopes  []string

	layersByName         map[string]LayerProfile
	dependencyLayerSet   map[string]bool
	allowedKindSet       map[string]map[string]bool
	architectureScopeSet map[string]map[string]bool
	reservedScopeNameSet map[string]bool
}

type LayerProfile struct {
	Name               string
	FileNameMode       string
	FileKinds          []string
	ArchitectureScopes []string
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
	ArchitectureScopePrefix = "x_"
	FileNameModeScopeKind   = "scope_kind"
	FileNameModePackageKind = "package_kind"
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
			"handler":    FileNameModeScopeKind,
			"service":    FileNameModeScopeKind,
			"repository": FileNameModeScopeKind,
		},
		// Keep shared/cross-layer entries before layer-specific entries.
		LayerFileKinds: map[string][]string{
			"handler":    {"support", "mapper", "context", "dto", "endpoint", "handler", "middleware", "utils"},
			"service":    {"support", "mapper", "commands", "provider", "service"},
			"repository": {"support", "repository", "schema", "store"},
		},
		ArchitectureScopes: map[string][]string{
			"handler":    {"x_shared", "x_http"},
			"service":    {"x_shared"},
			"repository": {"x_shared", "x_store"},
		},
		EscapedScopeSuffixes: []string{
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
		ForbiddenWeakScopes: []string{"common", "default", "helper", "helpers", "misc", "util", "utils"},
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
	if c.LayerFileKinds == nil {
		c.LayerFileKinds = defaults.LayerFileKinds
	}
	if c.ArchitectureScopes == nil {
		c.ArchitectureScopes = defaults.ArchitectureScopes
	}
	if c.EscapedScopeSuffixes == nil {
		c.EscapedScopeSuffixes = defaults.EscapedScopeSuffixes
	}
	if c.ForbiddenWeakScopes == nil {
		c.ForbiddenWeakScopes = defaults.ForbiddenWeakScopes
	}
	return c
}

// Compile applies defaults, validates cross-field references, and deep-clones
// caller-owned collections so an analysis cannot observe later mutations.
func (c Config) Compile() (Config, error) {
	inheritedLayerRules := c.LayerRules == nil
	c = c.WithDefaults()
	if inheritedLayerRules {
		layers := sliceSet(c.DependencyLayerDirs)
		c.LayerRules = slices.DeleteFunc(c.LayerRules, func(rule LayerDependencyRule) bool {
			return !layers[rule.SourceLayer] || !layers[rule.TargetLayer]
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
	c.LayerFileKinds = cloneStringSlices(c.LayerFileKinds)
	c.ArchitectureScopes = cloneStringSlices(c.ArchitectureScopes)
	c.EscapedScopeSuffixes = slices.Clone(c.EscapedScopeSuffixes)
	c.ForbiddenWeakScopes = slices.Clone(c.ForbiddenWeakScopes)
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
	dependencyLayers := sliceSet(c.DependencyLayerDirs)
	for _, layer := range c.LayerDirs {
		mode := c.LayerFileNameModes[layer]
		if mode != "" && mode != FileNameModeScopeKind && mode != FileNameModePackageKind {
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
		if mode != FileNameModeScopeKind && mode != FileNameModePackageKind {
			return fmt.Errorf("layer %q has invalid filename mode %q", layer, mode)
		}
		if len(c.LayerFileKinds[layer]) == 0 {
			return fmt.Errorf("layer %q has no allowed file kinds", layer)
		}
	}
	for layer := range c.LayerFileNameModes {
		if !layers[layer] {
			return fmt.Errorf("filename mode references unknown layer %q", layer)
		}
	}
	for layer := range c.LayerFileKinds {
		if !layers[layer] {
			return fmt.Errorf("file kinds reference unknown layer %q", layer)
		}
	}
	for layer := range c.ArchitectureScopes {
		if !layers[layer] {
			return fmt.Errorf("architecture scopes reference unknown layer %q", layer)
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

func (c Config) Profile() Profile {
	c = c.WithDefaults()
	layers := make([]LayerProfile, 0, len(c.LayerDirs))
	layersByName := make(map[string]LayerProfile, len(c.LayerDirs))
	allowedKinds := make(map[string]map[string]bool, len(c.LayerDirs))
	architectureScopes := make(map[string]map[string]bool, len(c.LayerDirs))
	reservedScopes := map[string]bool{}
	for _, layer := range c.LayerDirs {
		mode := c.LayerFileNameModes[layer]
		if mode == "" {
			mode = FileNameModeScopeKind
		}
		layerProfile := LayerProfile{
			Name:               layer,
			FileNameMode:       mode,
			FileKinds:          c.LayerFileKinds[layer],
			ArchitectureScopes: c.ArchitectureScopes[layer],
		}
		layers = append(layers, layerProfile)
		layersByName[layer] = layerProfile
		allowedKinds[layer] = sliceSet(layerProfile.FileKinds)
		architectureScopes[layer] = sliceSet(layerProfile.ArchitectureScopes)
		for _, scope := range layerProfile.ArchitectureScopes {
			if name, ok := strings.CutPrefix(scope, ArchitectureScopePrefix); ok {
				reservedScopes[name] = true
			}
		}
	}
	return Profile{
		Layers:               layers,
		DependencyLayers:     c.DependencyLayerDirs,
		SkipDirs:             c.SkipDirs,
		LayerRules:           c.LayerRules,
		EscapedScopeSuffixes: c.EscapedScopeSuffixes,
		ForbiddenWeakScopes:  c.ForbiddenWeakScopes,
		layersByName:         layersByName,
		dependencyLayerSet:   sliceSet(c.DependencyLayerDirs),
		allowedKindSet:       allowedKinds,
		architectureScopeSet: architectureScopes,
		reservedScopeNameSet: reservedScopes,
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
	return p.allowedKindSet[layerName][kind]
}

func (p Profile) ArchitectureScopeAllowed(layerName string, scope string) bool {
	return p.architectureScopeSet[layerName][scope]
}

func (p Profile) ArchitectureScopeReserved(scope string) bool {
	return p.reservedScopeNameSet[scope]
}

func StringIn(value string, values []string) bool {
	return slices.Contains(values, value)
}
