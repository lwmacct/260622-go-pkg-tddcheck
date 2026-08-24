package filelayout

import (
	"fmt"
	"sort"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func serviceSubjectViolations(context *rulekit.Snapshot) []Violation {
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
		if file.IsTest {
			continue
		}
		if file.Layer != "service" {
			continue
		}
		name, ok := parseFileName(file.Base, rulekit.FileNameModeQualifiedKind)
		if !ok || name.namespace != "" || name.kind == "free" {
			continue
		}
		item := subjects[name.subject]
		if item == nil {
			item = &subject{}
			subjects[name.subject] = item
		}
		item.files = append(item.files, file)
		if name.kind == "service" {
			item.hasService = true
		}
	}

	var subjectNames []string
	for subjectName, item := range subjects {
		if !item.hasService {
			subjectNames = append(subjectNames, subjectName)
		}
	}
	sort.Strings(subjectNames)
	var violations []Violation
	for _, subjectName := range subjectNames {
		files := subjects[subjectName].files
		sort.Slice(files, func(a, b int) bool {
			return files[a].AbsPath < files[b].AbsPath
		})
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(files[0].AbsPath),
			Line:    1,
			Message: fmt.Sprintf("service subject %q must declare %s.service.go with New%sService", subjectName, subjectName, upperCamelName(subjectName)),
		})
	}
	return violations
}
