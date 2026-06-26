package filelayout

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func serviceSubjectViolations(context *rulekit.Context) []Violation {
	serviceLayer, ok := context.Profile.Layer("service")
	if ok && serviceLayer.FileNameMode == rulekit.FileNameModePackageKind {
		return nil
	}
	type subject struct {
		files      []rulekit.GoFile
		hasService bool
	}
	subjects := map[string]*subject{}
	for _, file := range context.Files {
		if rulekit.FreeFile(file.Base) {
			continue
		}
		if file.Layer != "service" {
			continue
		}
		name, ok := parseFileName(file.Base, rulekit.FileNameModeScopeKind)
		if !ok || strings.HasPrefix(name.scope, architectureScopePrefix) {
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
		sort.Slice(files, func(a, b int) bool {
			return files[a].AbsPath < files[b].AbsPath
		})
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(files[0].AbsPath),
			Line:    1,
			Message: fmt.Sprintf("service subject %q must declare %s.service.go with New%sService", scope, scope, upperCamelName(scope)),
		})
	}
	return violations
}
