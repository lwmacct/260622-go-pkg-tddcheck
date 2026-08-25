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
	// LayerPackageNames maps strict layer directories to their expected Go
	// package names. Entries also make the layer directory an exact child of
	// the analyzed root; nested packages are reported as architecture errors.
	// Nil inherits the default handler/service/repository mapping.
	LayerPackageNames map[string]string `json:"layerPackageNames,omitempty"`
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
	// appear in exported service contracts. Nil inherits defaults;
	// an explicit empty slice disables this boundary check.
	PublicTypeBoundarySuffixes []string `json:"publicTypeBoundarySuffixes,omitempty"`
	// MaxSupportDeclarationLines limits the total lines occupied by type, const,
	// and var declarations in a support file. Zero disables the limit.
	MaxSupportDeclarationLines int `json:"maxSupportDeclarationLines,omitempty"`
	// StoreMethodActions lists recognized exported Store method action prefixes.
	// Unknown actions and action-specific result shapes are style warnings.
	// Nil inherits defaults; an explicit empty slice disables action-style checks.
	StoreMethodActions []string `json:"storeMethodActions,omitempty"`
	// WarnSubjectPrivateAccess reports private declarations used across business
	// subjects in the same layer package.
	WarnSubjectPrivateAccess bool `json:"warnSubjectPrivateAccess,omitempty"`
	// WarnUnclassifiedFiles reports Go files outside configured layout layers.
	WarnUnclassifiedFiles bool `json:"warnUnclassifiedFiles,omitempty"`
	// MaxSharedDeclarationLines limits the total AST line span of declarations in
	// x.shared files. Zero disables the warning.
	MaxSharedDeclarationLines int `json:"maxSharedDeclarationLines,omitempty"`
	// SubjectOwnershipModes controls subject-prefix checks by layer and kind.
	// Values are error, warning, or off. Unconfigured entries remain errors.
	SubjectOwnershipModes map[string]map[string]string `json:"subjectOwnershipModes,omitempty"`
	// Initialisms lists snake-case subject components that are exported in all
	// caps, for example api -> API and http -> HTTP. Nil inherits defaults.
	Initialisms []string `json:"initialisms,omitempty"`
	// InitialismOverrides maps exceptional components to their exact exported
	// spelling, for example oauth -> OAuth. Nil inherits defaults.
	InitialismOverrides map[string]string `json:"initialismOverrides,omitempty"`
	// FailOnWarnings makes Analysis.Passed report warnings as failures.
	FailOnWarnings bool `json:"failOnWarnings,omitempty"`
}

type Profile struct {
	Layers                     []LayerProfile
	DependencyLayers           []string
	SkipDirs                   []string
	LayerRules                 []LayerDependencyRule
	LayerPackageNames          map[string]string
	EscapedSubjectSuffixes     []string
	ForbiddenWeakSubjects      []string
	PublicTypeBoundarySuffixes []string
	MaxSupportDeclarationLines int
	StoreMethodActions         []string
	WarnSubjectPrivateAccess   bool
	WarnUnclassifiedFiles      bool
	MaxSharedDeclarationLines  int
	SubjectOwnershipModes      map[string]map[string]string
	Initialisms                map[string]string
	InitialismOverrides        map[string]string
	FailOnWarnings             bool

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
		LayerPackageNames: map[string]string{
			"handler":    "handler",
			"service":    "service",
			"repository": "repository",
		},
		SkipDirs: []string{".git", ".hg", ".svn", "vendor", "node_modules", "dist", "build"},
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
				"handler": "handler", "middleware": "middleware",
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
		StoreMethodActions:         []string{"List", "Fetch", "Count", "Exists", "Create", "Update", "Delete", "Upsert", "Add", "Remove", "Replace"},
		Initialisms: []string{
			"api", "http", "id", "ip", "json", "llm", "rbac", "sql", "ssh", "tls", "url", "uuid", "ws",
		},
		InitialismOverrides: map[string]string{"oauth": "OAuth"},
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
	if c.LayerPackageNames == nil {
		c.LayerPackageNames = defaults.LayerPackageNames
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
	if c.StoreMethodActions == nil {
		c.StoreMethodActions = defaults.StoreMethodActions
	}
	if c.SubjectOwnershipModes == nil {
		c.SubjectOwnershipModes = defaults.SubjectOwnershipModes
	}
	if c.Initialisms == nil {
		c.Initialisms = defaults.Initialisms
	}
	if c.InitialismOverrides == nil {
		c.InitialismOverrides = defaults.InitialismOverrides
	}
	return c
}

// Compile applies defaults, validates cross-field references, and deep-clones
// caller-owned collections so an analysis cannot observe later mutations.
func (c Config) Compile() (Config, error) {
	inheritedLayerRules := c.LayerRules == nil
	inheritedSubjectAnchors := c.LayerSubjectAnchorKinds == nil
	inheritedLayerPackageNames := c.LayerPackageNames == nil
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
	if inheritedLayerPackageNames {
		layers := sliceSet(c.LayerDirs)
		maps.DeleteFunc(c.LayerPackageNames, func(layer string, _ string) bool {
			return !layers[layer]
		})
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	c.BuildFlags = slices.Clone(c.BuildFlags)
	c.LayerDirs = slices.Clone(c.LayerDirs)
	c.DependencyLayerDirs = slices.Clone(c.DependencyLayerDirs)
	c.LayerPackageNames = maps.Clone(c.LayerPackageNames)
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
	c.StoreMethodActions = slices.Clone(c.StoreMethodActions)
	c.SubjectOwnershipModes = cloneStringMaps(c.SubjectOwnershipModes)
	c.Initialisms = slices.Clone(c.Initialisms)
	c.InitialismOverrides = maps.Clone(c.InitialismOverrides)
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
	if c.MaxSharedDeclarationLines < 0 {
		return fmt.Errorf("max shared declaration lines must not be negative")
	}
	for layer, kinds := range c.SubjectOwnershipModes {
		if !sliceSet(c.LayerDirs)[layer] {
			return fmt.Errorf("subject ownership references unknown layer %q", layer)
		}
		for kind, mode := range kinds {
			if !validFileAtom(kind) {
				return fmt.Errorf("subject ownership has invalid file kind %q", kind)
			}
			switch mode {
			case "error", "warning", "off":
			default:
				return fmt.Errorf("subject ownership for %s.%s has invalid mode %q", layer, kind, mode)
			}
		}
	}
	for _, key := range c.Initialisms {
		if !validFileAtom(key) {
			return fmt.Errorf("initialism %q must be a lowercase filename atom", key)
		}
	}
	for key, value := range c.InitialismOverrides {
		if !validFileAtom(key) {
			return fmt.Errorf("initialism override %q must be a lowercase filename atom", key)
		}
		if !startsWithUpperASCII(value) {
			return fmt.Errorf("initialism override %q must start with an uppercase letter", value)
		}
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
	for layer, packageName := range c.LayerPackageNames {
		if !dependencyLayers[layer] && !sliceSet(c.LayerDirs)[layer] {
			return fmt.Errorf("layer package names reference unknown layer %q", layer)
		}
		if !validPackageName(packageName) {
			return fmt.Errorf("layer %q has invalid package name %q", layer, packageName)
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
	for _, action := range c.StoreMethodActions {
		if !validExportedIdentifier(action) {
			return fmt.Errorf("store method action %q must be an exported identifier", action)
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

func validPackageName(value string) bool {
	if value == "" || (value[0] < 'a' || value[0] > 'z') {
		return false
	}
	for _, char := range value[1:] {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

func validExportedIdentifier(value string) bool {
	if value == "" || value[0] < 'A' || value[0] > 'Z' {
		return false
	}
	for _, char := range value[1:] {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '_' {
			return false
		}
	}
	return true
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

func effectiveInitialisms(values []string, overrides map[string]string) map[string]string {
	result := make(map[string]string, len(values)+len(overrides))
	for _, value := range values {
		result[strings.ToLower(value)] = strings.ToUpper(value)
	}
	for key, value := range overrides {
		result[strings.ToLower(key)] = value
	}
	return result
}

func startsWithUpperASCII(value string) bool {
	return value != "" && value[0] >= 'A' && value[0] <= 'Z'
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
		LayerPackageNames:          maps.Clone(c.LayerPackageNames),
		EscapedSubjectSuffixes:     c.EscapedSubjectSuffixes,
		ForbiddenWeakSubjects:      c.ForbiddenWeakSubjects,
		PublicTypeBoundarySuffixes: slices.Clone(c.PublicTypeBoundarySuffixes),
		MaxSupportDeclarationLines: c.MaxSupportDeclarationLines,
		StoreMethodActions:         slices.Clone(c.StoreMethodActions),
		WarnSubjectPrivateAccess:   c.WarnSubjectPrivateAccess,
		WarnUnclassifiedFiles:      c.WarnUnclassifiedFiles,
		MaxSharedDeclarationLines:  c.MaxSharedDeclarationLines,
		SubjectOwnershipModes:      cloneStringMaps(c.SubjectOwnershipModes),
		Initialisms:                effectiveInitialisms(c.Initialisms, c.InitialismOverrides),
		InitialismOverrides:        maps.Clone(c.InitialismOverrides),
		FailOnWarnings:             c.FailOnWarnings,
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

func (p Profile) PackageName(layer string) (string, bool) {
	name, ok := p.LayerPackageNames[layer]
	return name, ok
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

func (p Profile) SubjectOwnershipMode(layerName string, kind string) string {
	if mode := p.SubjectOwnershipModes[layerName][kind]; mode != "" {
		return mode
	}
	return "error"
}

func StringIn(value string, values []string) bool {
	return slices.Contains(values, value)
}
