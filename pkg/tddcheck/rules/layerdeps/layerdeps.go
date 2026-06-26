package layerdeps

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

const RuleID = "layerdeps"

type Rules struct {
	root   string
	config rulekit.Config
}

type Violation struct {
	File       string
	Line       int
	ImportPath string
	Message    string
}

func New(root string, options ...rulekit.Option) Rules {
	values := rulekit.NewRuleOptions(root, options...)
	return Rules{root: values.Root, config: values.Config}
}

func (r Rules) ID() string {
	return RuleID
}

func (r Rules) Check(context *rulekit.Context) ([]rulekit.Diagnostic, error) {
	values := violationsInContext(context)
	diagnostics := make([]rulekit.Diagnostic, 0, len(values))
	for _, value := range values {
		diagnostics = append(diagnostics, rulekit.Diagnostic{
			Rule:    RuleID,
			File:    value.File,
			Line:    value.Line,
			Message: value.Message + ": " + value.ImportPath,
		})
	}
	return diagnostics, nil
}

func (r Rules) Assert(t *testing.T) {
	t.Helper()

	violations, err := r.Violations()
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) == 0 {
		return
	}

	lines := make([]string, 0, len(violations))
	for _, violation := range violations {
		lines = append(lines, fmt.Sprintf(
			"%s:%d: %s: %s",
			violation.File,
			violation.Line,
			violation.Message,
			violation.ImportPath,
		))
	}
	t.Fatalf("invalid layer dependencies:\n  - %s", strings.Join(lines, "\n  - "))
}

func (r Rules) Violations() ([]Violation, error) {
	context, err := rulekit.NewContext(r.root, RuleID, r.config)
	if err != nil {
		return nil, err
	}
	return violationsInContext(context), nil
}

func violationsInContext(context *rulekit.Context) []Violation {
	var violations []Violation
	for _, file := range context.Files {
		if rulekit.FreeFile(file.Base) {
			continue
		}
		violations = append(violations, violationsInFile(context, file)...)
	}
	return violations
}

func violationsInFile(context *rulekit.Context, file rulekit.GoFile) []Violation {
	if rulekit.FreeFile(file.Base) {
		return nil
	}
	profile := context.Profile
	sourceLayer, sourceRel, ok := sourceLayer(file, profile)
	if !ok {
		return nil
	}

	var violations []Violation
	for _, imported := range file.Imports {
		targetLayer, targetRel, ok := importLayer(context.ModulePath, imported.Path, profile)
		if !ok {
			continue
		}
		message, invalid := invalidDependency(profile, sourceLayer, sourceRel, targetLayer, targetRel)
		if !invalid {
			continue
		}
		violations = append(violations, Violation{
			File:       rulekit.DisplayFilename(file.AbsPath),
			Line:       imported.Line,
			ImportPath: imported.Path,
			Message:    message,
		})
	}
	return violations
}

func sourceLayer(file rulekit.GoFile, profile rulekit.Profile) (string, string, bool) {
	for _, part := range strings.Split(file.RelPath, "/") {
		if profile.DependencyLayer(part) {
			return part, file.Dir, true
		}
	}
	return "", "", false
}

func importLayer(modulePath string, importPath string, profile rulekit.Profile) (string, string, bool) {
	prefix := modulePath + "/internal/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", "", false
	}
	rel := strings.TrimPrefix(importPath, prefix)
	if profile.DependencyLayer(rel) {
		return rel, rel, true
	}
	layer, _, ok := strings.Cut(rel, "/")
	if !ok || !profile.DependencyLayer(layer) {
		return "", "", false
	}
	return layer, rel, true
}

func invalidDependency(profile rulekit.Profile, source string, sourceRel string, target string, targetRel string) (string, bool) {
	for _, rule := range profile.LayerRules {
		if source != rule.SourceLayer || target != rule.TargetLayer {
			continue
		}
		if rule.SourceRelPrefix != "" && !strings.HasPrefix(sourceRel, rule.SourceRelPrefix) {
			continue
		}
		if rule.TargetRelPrefix != "" && !strings.HasPrefix(targetRel, rule.TargetRelPrefix) {
			continue
		}
		if sourceExcepted(rule, sourceRel) {
			continue
		}
		if dependencyExcepted(rule, targetRel) {
			continue
		}
		message := rule.Message
		if message == "" {
			message = source + " must not import " + target
		}
		return message, true
	}
	return "", false
}

func sourceExcepted(rule rulekit.LayerDependencyRule, sourceRel string) bool {
	for _, prefix := range rule.ExceptSourceRelPrefixes {
		if strings.HasPrefix(sourceRel, prefix) {
			return true
		}
	}
	return false
}

func dependencyExcepted(rule rulekit.LayerDependencyRule, targetRel string) bool {
	for _, prefix := range rule.ExceptTargetRelPrefixes {
		if strings.HasPrefix(targetRel, prefix) {
			return true
		}
	}
	return false
}
