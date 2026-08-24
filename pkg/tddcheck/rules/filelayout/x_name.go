package filelayout

import (
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

type fileName struct {
	subject   string
	namespace string
	kind      string
}

func parseFileName(base string, mode string) (fileName, bool) {
	if !strings.HasSuffix(base, ".go") {
		return fileName{}, false
	}
	name := strings.TrimSuffix(base, ".go")
	if mode == rulekit.FileNameModePackageKind {
		if name == "" || strings.Contains(name, "..") {
			return fileName{}, false
		}
		return fileName{kind: name}, true
	}
	parts := strings.Split(name, ".")
	switch {
	case len(parts) == 2 && parts[0] != "" && parts[0] != "x" && validKind(parts[1]):
		return fileName{subject: parts[0], kind: parts[1]}, true
	case len(parts) == 3 && parts[0] == "x" && parts[1] != "" && validKind(parts[2]):
		return fileName{namespace: parts[1], kind: parts[2]}, true
	default:
		return fileName{}, false
	}
}

func validKind(value string) bool {
	return value != "" && !strings.Contains(value, "_")
}

func (n fileName) qualifier() string {
	if n.namespace != "" {
		return n.namespace
	}
	return n.subject
}
