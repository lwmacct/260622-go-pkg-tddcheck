package filelayout

import "strings"

type fileName struct {
	scope string
	kind  string
}

func parseFileName(base string) (fileName, bool) {
	if !strings.HasSuffix(base, ".go") {
		return fileName{}, false
	}
	name := strings.TrimSuffix(base, ".go")
	parts := strings.Split(name, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fileName{}, false
	}
	if strings.Contains(parts[1], "_") {
		return fileName{}, false
	}
	return fileName{scope: parts[0], kind: parts[1]}, true
}
