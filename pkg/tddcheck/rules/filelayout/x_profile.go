package filelayout

import "github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"

const architectureScopePrefix = "x_"

type layoutProfile struct {
	kindsByLayer         map[string][]string
	architectureScopes   map[string][]string
	escapedScopeSuffixes []string
}

func hsrStrictProfile() layoutProfile {
	return profileFromConfig(rulekit.DefaultConfig())
}

func profileFromConfig(config rulekit.Config) layoutProfile {
	config = config.WithDefaults()
	return layoutProfile{
		kindsByLayer:         config.LayerFileKinds,
		architectureScopes:   config.ArchitectureScopes,
		escapedScopeSuffixes: config.EscapedScopeSuffixes,
	}
}

func (p layoutProfile) kindAllowed(layer string, kind string) bool {
	return oneOf(kind, p.kindsByLayer[layer]...)
}

func (p layoutProfile) architectureScopeReserved(scope string) bool {
	for _, scopes := range p.architectureScopes {
		if oneOf(architectureScopePrefix+scope, scopes...) {
			return true
		}
	}
	return false
}

func (p layoutProfile) architectureScopeAllowed(layer string, scope string) bool {
	return oneOf(scope, p.architectureScopes[layer]...)
}
