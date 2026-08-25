package filelayout

import (
	"fmt"
	"sort"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func subjectAnchorViolations(context *rulekit.Snapshot) []Violation {
	type subject struct {
		files     []rulekit.GoFile
		hasAnchor bool
	}
	type subjectKey struct {
		layer   string
		subject string
	}
	subjects := map[subjectKey]*subject{}
	for _, file := range context.Files {
		if file.IsTest {
			continue
		}
		layer, ok := context.Profile.Layer(file.Layer)
		if !ok || layer.SubjectAnchorKind == "" {
			continue
		}
		name, ok := file.Identity, file.IdentityOK
		if !ok || name.Namespace != "" || name.Kind == "free" {
			continue
		}
		key := subjectKey{layer: file.Layer, subject: name.Subject}
		item := subjects[key]
		if item == nil {
			item = &subject{}
			subjects[key] = item
		}
		item.files = append(item.files, file)
		if name.Kind == layer.SubjectAnchorKind {
			item.hasAnchor = true
		}
	}

	var missing []subjectKey
	for key, item := range subjects {
		if !item.hasAnchor {
			missing = append(missing, key)
		}
	}
	sort.Slice(missing, func(a, b int) bool {
		if missing[a].layer != missing[b].layer {
			return missing[a].layer < missing[b].layer
		}
		return missing[a].subject < missing[b].subject
	})
	var violations []Violation
	for _, key := range missing {
		files := subjects[key].files
		sort.Slice(files, func(a, b int) bool {
			return files[a].AbsPath < files[b].AbsPath
		})
		layer, _ := context.Profile.Layer(key.layer)
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(files[0].AbsPath),
			Line:    1,
			Code:    RuleID + "/missing-subject-anchor",
			Message: fmt.Sprintf("%s subject %q must declare %s.%s.go as its anchor", key.layer, key.subject, key.subject, layer.SubjectAnchorKind),
		})
	}
	return violations
}
