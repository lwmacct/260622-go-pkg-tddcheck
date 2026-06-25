package rulekit

import "slices"

type Config struct {
	LayerDirs           []string
	DependencyLayerDirs []string
	SkipDirs            []string
	LayerRules          []LayerDependencyRule

	LayerFileNameModes   map[string]string
	LayerFileKinds       map[string][]string
	ArchitectureScopes   map[string][]string
	EscapedScopeSuffixes []string
	ForbiddenWeakScopes  []string
}

type LayerDependencyRule struct {
	SourceLayer             string
	SourceRelPrefix         string
	ExceptSourceRelPrefixes []string
	TargetLayer             string
	TargetRelPrefix         string
	ExceptTargetRelPrefixes []string
	Message                 string
}

const (
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
		LayerFileKinds: map[string][]string{
			"handler":    {"context", "dto", "endpoint", "handler", "mapper", "middleware", "support", "utils"},
			"service":    {"commands", "mapper", "provider", "service", "support"},
			"repository": {"repository", "schema", "store", "support"},
		},
		ArchitectureScopes: map[string][]string{
			"handler":    {"x_http", "x_shared"},
			"service":    {"x_shared"},
			"repository": {"x_store", "x_shared"},
		},
		EscapedScopeSuffixes: []string{
			"commands",
			"constants",
			"context",
			"create",
			"delete",
			"dto",
			"errors",
			"handler",
			"list",
			"mapper",
			"model",
			"patch",
			"provider",
			"repository",
			"schema",
			"service",
			"store",
			"support",
			"update",
			"upsert",
			"utils",
			"validation",
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

func StringIn(value string, values []string) bool {
	return slices.Contains(values, value)
}
