package filelayout

import "github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"

const architectureScopePrefix = "x_"

type layoutProfile struct {
	kindsByLayer             map[string][]string
	architectureKindsByLayer map[string]map[string][]string
	escapedScopeSuffixes     []string
}

func hsrStrictProfile() layoutProfile {
	return profileFromConfig(rulekit.DefaultConfig())
}

func profileFromConfig(config rulekit.Config) layoutProfile {
	config = config.WithDefaults()
	return layoutProfile{
		kindsByLayer:             config.LayerFileKinds,
		architectureKindsByLayer: config.ArchitectureFileKinds,
		escapedScopeSuffixes:     config.EscapedScopeSuffixes,
	}
}

func (p layoutProfile) kindAllowed(layer string, kind string) bool {
	return oneOf(kind, p.kindsByLayer[layer]...)
}

func (p layoutProfile) architectureKindAllowed(layer string, scope string, kind string) bool {
	return oneOf(kind, p.architectureKindsByLayer[layer][scope]...)
}

func (p layoutProfile) architectureScopeReserved(scope string) bool {
	for _, scopes := range p.architectureKindsByLayer {
		if _, ok := scopes[architectureScopePrefix+scope]; ok {
			return true
		}
	}
	return false
}
