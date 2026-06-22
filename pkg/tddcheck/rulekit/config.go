package rulekit

import "slices"

type Config struct {
	LayerDirs  []string
	SkipDirs   []string
	LayerRules []LayerDependencyRule

	AllowedSharedScopes []string
	ForbiddenWeakScopes []string
}

type LayerDependencyRule struct {
	SourceLayer     string
	TargetLayer     string
	TargetRelPrefix string
	Message         string
}

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
		AllowedSharedScopes: []string{
			"api",
			"batch",
			"database",
			"frontend",
			"ids",
			"repository",
			"router",
			"schema",
			"shared",
			"store",
			"write",
		},
		ForbiddenWeakScopes: []string{"common", "default", "helper", "helpers", "misc", "util", "utils"},
	}
}

func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()
	if c.LayerDirs == nil {
		c.LayerDirs = defaults.LayerDirs
	}
	if c.SkipDirs == nil {
		c.SkipDirs = defaults.SkipDirs
	}
	if c.LayerRules == nil {
		c.LayerRules = defaults.LayerRules
	}
	if c.AllowedSharedScopes == nil {
		c.AllowedSharedScopes = defaults.AllowedSharedScopes
	}
	if c.ForbiddenWeakScopes == nil {
		c.ForbiddenWeakScopes = defaults.ForbiddenWeakScopes
	}
	return c
}

func StringIn(value string, values []string) bool {
	return slices.Contains(values, value)
}
