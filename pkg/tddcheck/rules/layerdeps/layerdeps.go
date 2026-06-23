package layerdeps

import (
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
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
	root, err := rulekit.ResolveRuleRoot(r.root, RuleID)
	if err != nil {
		return nil, err
	}
	modulePath, err := rulekit.ModulePathForRoot(root)
	if err != nil {
		return nil, err
	}
	config := r.config.WithDefaults()
	files, err := rulekit.ModuleFiles(root, RuleID, config, func(name string) bool {
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	})
	if err != nil {
		return nil, err
	}

	var violations []Violation
	for _, file := range files {
		if isFreeFile(file) {
			continue
		}
		fileViolations, err := violationsInFile(root, modulePath, config, file)
		if err != nil {
			return nil, err
		}
		violations = append(violations, fileViolations...)
	}
	return violations, nil
}

func violationsInFile(root string, modulePath string, config rulekit.Config, filename string) ([]Violation, error) {
	if isFreeFile(filename) {
		return nil, nil
	}
	sourceLayer, sourceRel, ok := sourceLayer(root, filename, config)
	if !ok {
		return nil, nil
	}

	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, filename, nil, parser.ImportsOnly|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	var violations []Violation
	for _, importSpec := range parsedFile.Imports {
		importPath := strings.Trim(importSpec.Path.Value, `"`)
		targetLayer, targetRel, ok := importLayer(modulePath, importPath, config)
		if !ok {
			continue
		}
		message, invalid := invalidDependency(config, sourceLayer, sourceRel, targetLayer, targetRel)
		if !invalid {
			continue
		}
		violations = append(violations, Violation{
			File:       rulekit.DisplayFilename(filename),
			Line:       fileSet.Position(importSpec.Pos()).Line,
			ImportPath: importPath,
			Message:    message,
		})
	}
	return violations, nil
}

func sourceLayer(root string, filename string, config rulekit.Config) (string, string, bool) {
	rel, err := filepath.Rel(root, filename)
	if err != nil {
		return "", "", false
	}
	sourceRel := filepath.ToSlash(filepath.Dir(rel))
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if slices.Contains(config.DependencyLayerDirs, part) {
			return part, sourceRel, true
		}
	}
	return "", "", false
}

func importLayer(modulePath string, importPath string, config rulekit.Config) (string, string, bool) {
	prefix := modulePath + "/internal/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", "", false
	}
	rel := strings.TrimPrefix(importPath, prefix)
	if slices.Contains(config.DependencyLayerDirs, rel) {
		return rel, rel, true
	}
	layer, _, ok := strings.Cut(rel, "/")
	if !ok || !slices.Contains(config.DependencyLayerDirs, layer) {
		return "", "", false
	}
	return layer, rel, true
}

func invalidDependency(config rulekit.Config, source string, sourceRel string, target string, targetRel string) (string, bool) {
	for _, rule := range config.LayerRules {
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

func isFreeFile(filename string) bool {
	return filepath.Base(filename) == "x_free.go"
}

func dependencyExcepted(rule rulekit.LayerDependencyRule, targetRel string) bool {
	for _, prefix := range rule.ExceptTargetRelPrefixes {
		if strings.HasPrefix(targetRel, prefix) {
			return true
		}
	}
	return false
}
