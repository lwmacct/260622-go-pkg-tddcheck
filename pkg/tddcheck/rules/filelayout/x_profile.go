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
			"service":    {"commands", "constants", "errors", "mapper", "model", "service", "utils", "validation"},
			"repository": {"constants", "errors", "model", "repository", "schema", "store", "utils"},
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
				"x_id":     {"validation"},
				"x_shared": {"errors", "mapper", "model", "utils", "validation"},
			},
			"repository": {
				"x_database": {"repository"},
				"x_schema":   {"repository", "utils"},
				"x_store":    {"repository"},
				"x_shared":   {"constants", "errors", "model", "utils"},
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
