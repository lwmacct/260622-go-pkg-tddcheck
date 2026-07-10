package rulekit

import "slices"

// Config defines the directories and conventions used by architecture rules.
// Each nil slice or map is independently populated by [Config.WithDefaults].
type Config struct {
	// LayerDirs lists directory names checked by file-layout rules.
	LayerDirs []string
	// DependencyLayerDirs lists directory names recognized by dependency rules.
	// A nil value inherits LayerDirs.
	DependencyLayerDirs []string
	// SkipDirs lists directory names excluded from source scanning.
	SkipDirs []string
	// LayerRules lists forbidden import relationships.
	LayerRules []LayerDependencyRule

	// LayerFileNameModes maps a layer to a FileNameMode constant.
	LayerFileNameModes map[string]string
	// LayerFileKinds maps a layer to its allowed filename kinds.
	LayerFileKinds map[string][]string
	// ArchitectureScopes maps a layer to its allowed x_ scopes.
	ArchitectureScopes map[string][]string
	// EscapedScopeSuffixes lists kinds and actions that business scopes must not
	// encode as suffixes.
	EscapedScopeSuffixes []string
	// ForbiddenWeakScopes lists ambiguous business scope names.
	ForbiddenWeakScopes []string
}

type Profile struct {
	Layers               []LayerProfile
	DependencyLayers     []string
	SkipDirs             []string
	LayerRules           []LayerDependencyRule
	EscapedScopeSuffixes []string
	ForbiddenWeakScopes  []string
}

type LayerProfile struct {
	Name               string
	FileNameMode       string
	FileKinds          []string
	ArchitectureScopes []string
}

// LayerDependencyRule describes a forbidden import relationship. Source path
// prefixes are relative to the analyzed root; target path prefixes are relative
// to the module's internal directory.
type LayerDependencyRule struct {
	// SourceLayer is the layer containing the importing file.
	SourceLayer string
	// SourceRelPrefix optionally restricts importing directories relative to the
	// analyzed root.
	SourceRelPrefix string
	// ExceptSourceRelPrefixes exempts importing directories relative to the
	// analyzed root.
	ExceptSourceRelPrefixes []string
	// TargetLayer is the layer containing the imported package.
	TargetLayer string
	// TargetRelPrefix optionally restricts imported packages relative to the
	// module's internal directory.
	TargetRelPrefix string
	// ExceptTargetRelPrefixes exempts imported packages relative to the module's
	// internal directory.
	ExceptTargetRelPrefixes []string
	// Message overrides the default violation message when non-empty.
	Message string
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

func (c Config) Profile() Profile {
	c = c.WithDefaults()
	layers := make([]LayerProfile, 0, len(c.LayerDirs))
	for _, layer := range c.LayerDirs {
		mode := c.LayerFileNameModes[layer]
		if mode == "" {
			mode = FileNameModeScopeKind
		}
		layers = append(layers, LayerProfile{
			Name:               layer,
			FileNameMode:       mode,
			FileKinds:          c.LayerFileKinds[layer],
			ArchitectureScopes: c.ArchitectureScopes[layer],
		})
	}
	return Profile{
		Layers:               layers,
		DependencyLayers:     c.DependencyLayerDirs,
		SkipDirs:             c.SkipDirs,
		LayerRules:           c.LayerRules,
		EscapedScopeSuffixes: c.EscapedScopeSuffixes,
		ForbiddenWeakScopes:  c.ForbiddenWeakScopes,
	}
}

func (p Profile) Layer(name string) (LayerProfile, bool) {
	for _, layer := range p.Layers {
		if layer.Name == name {
			return layer, true
		}
	}
	return LayerProfile{}, false
}

func (p Profile) LayerNames() []string {
	names := make([]string, 0, len(p.Layers))
	for _, layer := range p.Layers {
		names = append(names, layer.Name)
	}
	return names
}

func (p Profile) DependencyLayer(name string) bool {
	return StringIn(name, p.DependencyLayers)
}

func (p Profile) KindAllowed(layerName string, kind string) bool {
	layer, ok := p.Layer(layerName)
	return ok && StringIn(kind, layer.FileKinds)
}

func (p Profile) ArchitectureScopeAllowed(layerName string, scope string) bool {
	layer, ok := p.Layer(layerName)
	return ok && StringIn(scope, layer.ArchitectureScopes)
}

func (p Profile) ArchitectureScopeReserved(scope string) bool {
	for _, layer := range p.Layers {
		if StringIn(ArchitectureScopePrefix+scope, layer.ArchitectureScopes) {
			return true
		}
	}
	return false
}

func StringIn(value string, values []string) bool {
	return slices.Contains(values, value)
}
