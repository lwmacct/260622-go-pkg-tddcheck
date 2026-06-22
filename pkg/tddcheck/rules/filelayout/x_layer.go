package filelayout

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func layerForFile(root string, file string, config rulekit.Config) (string, bool) {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return "", false
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if slices.Contains(config.LayerDirs, part) {
			return part, true
		}
	}
	return "", false
}

func allowedKind(layer string, kind string) bool {
	switch layer {
	case "handler":
		return oneOf(kind, "dto", "handler", "mapper", "utils")
	case "service":
		return oneOf(kind, "commands", "constants", "errors", "mapper", "models", "service", "utils", "validation", "writes")
	case "repository":
		return oneOf(kind, "constants", "database", "errors", "model", "models", "repository", "schema", "store", "utils")
	default:
		return false
	}
}

func allowedArchitectureKind(layer string, scope string, kind string) bool {
	switch layer {
	case "handler":
		switch scope {
		case "x_api", "x_frontend", "x_router", "x_shared":
			return oneOf(kind, "handler", "utils")
		default:
			return false
		}
	case "service":
		switch scope {
		case "x_batch":
			return kind == "service"
		case "x_id":
			return kind == "validation"
		case "x_shared":
			return oneOf(kind, "errors", "mapper", "models", "utils", "validation", "writes")
		default:
			return false
		}
	case "repository":
		switch scope {
		case "x_database":
			return kind == "repository"
		case "x_schema":
			return oneOf(kind, "repository", "utils")
		case "x_store":
			return kind == "repository"
		case "x_shared":
			return oneOf(kind, "constants", "errors", "models", "utils")
		default:
			return false
		}
	default:
		return false
	}
}
