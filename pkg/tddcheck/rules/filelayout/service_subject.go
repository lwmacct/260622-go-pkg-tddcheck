package filelayout

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func serviceSubjectViolations(root string, files []string, config rulekit.Config) []Violation {
	if config.LayerFileNameModes["service"] == rulekit.FileNameModePackageKind {
		return nil
	}
	type subject struct {
		files      []string
		hasService bool
	}
	subjects := map[string]*subject{}
	for _, file := range files {
		if isFreeFile(file) {
			continue
		}
		layer, ok := layerForFile(root, file, config)
		if !ok || layer != "service" {
			continue
		}
		name, ok := parseFileName(filepath.Base(file), rulekit.FileNameModeScopeKind)
		if !ok || strings.HasPrefix(name.scope, architectureScopePrefix) || name.scope == "x_batch" {
			continue
		}
		item := subjects[name.scope]
		if item == nil {
			item = &subject{}
			subjects[name.scope] = item
		}
		item.files = append(item.files, file)
		if name.kind == "service" {
			item.hasService = true
		}
	}

	var scopes []string
	for scope, item := range subjects {
		if !item.hasService {
			scopes = append(scopes, scope)
		}
	}
	sort.Strings(scopes)
	var violations []Violation
	for _, scope := range scopes {
		files := subjects[scope].files
		sort.Strings(files)
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(files[0]),
			Line:    1,
			Message: fmt.Sprintf("service subject %q must declare %s.service.go with New%sService", scope, scope, upperCamelName(scope)),
		})
	}
	return violations
}
