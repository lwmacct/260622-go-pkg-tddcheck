package filelayout

const architectureScopePrefix = "x_"

type layoutProfile struct {
	kindsByLayer             map[string][]string
	architectureKindsByLayer map[string]map[string][]string
	escapedScopeSuffixes     []string
}

func hsrStrictProfile() layoutProfile {
	return layoutProfile{
		kindsByLayer: map[string][]string{
			"handler":    {"dto", "handler", "mapper", "utils"},
			"service":    {"commands", "mapper", "service", "support"},
			"repository": {"repository", "schema", "store", "support"},
		},
		architectureKindsByLayer: map[string]map[string][]string{
			"handler": {
				"x_api":      {"handler", "utils"},
				"x_frontend": {"handler", "utils"},
				"x_router":   {"handler", "utils"},
				"x_shared":   {"handler", "utils"},
			},
			"service": {
				"x_batch":  {"service"},
				"x_id":     {"support"},
				"x_shared": {"mapper", "support"},
			},
			"repository": {
				"x_database": {"repository"},
				"x_schema":   {"repository", "support"},
				"x_store":    {"repository"},
				"x_shared":   {"support"},
			},
		},
		escapedScopeSuffixes: []string{
			"commands",
			"constants",
			"create",
			"delete",
			"dto",
			"errors",
			"handler",
			"list",
			"mapper",
			"model",
			"patch",
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
