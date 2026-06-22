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
