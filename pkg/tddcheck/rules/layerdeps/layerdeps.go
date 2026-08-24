package layerdeps

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

const RuleID = "layerdeps"

type Violation struct {
	File       string
	Line       int
	Column     int
	ImportPath string
	Message    string
}

func Register(engine *rulekit.Engine) {
	engine.Register(RuleID, rulekit.FileScope, checkFile)
}

func Check(ctx context.Context, root string, config rulekit.Config) ([]Violation, error) {
	snapshot, err := rulekit.Load(ctx, root, config)
	if err != nil {
		return nil, err
	}
	return violationsInSnapshot(snapshot), nil
}

func checkFile(_ context.Context, snapshot *rulekit.Snapshot, file rulekit.GoFile) ([]rulekit.Diagnostic, error) {
	values := violationsInFile(snapshot, file)
	diagnostics := make([]rulekit.Diagnostic, 0, len(values))
	for _, value := range values {
		position := rulekit.Position{File: value.File, Line: value.Line, Column: value.Column}
		diagnostics = append(diagnostics, rulekit.NewDiagnostic(
			RuleID,
			rulekit.SeverityError,
			value.Message+": "+value.ImportPath,
			position,
			position,
		))
	}
	return diagnostics, nil
}

func violationsInSnapshot(snapshot *rulekit.Snapshot) []Violation {
	var violations []Violation
	for _, file := range snapshot.Files {
		if rulekit.FreeFile(file.Base) {
			continue
		}
		violations = append(violations, violationsInFile(snapshot, file)...)
	}
	return violations
}

func violationsInFile(snapshot *rulekit.Snapshot, file rulekit.GoFile) []Violation {
	if rulekit.FreeFile(file.Base) {
		return nil
	}
	profile := snapshot.Profile
	sourceLayer, sourceRel, ok := sourceLayer(file, profile)
	if !ok {
		return nil
	}

	var violations []Violation
	for _, imported := range file.Imports {
		targetLayer, targetRel, ok := importLayer(snapshot, imported.PackagePath, profile)
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
			Column:     imported.Column,
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

func importLayer(snapshot *rulekit.Snapshot, importPath string, profile rulekit.Profile) (string, string, bool) {
	target, ok := snapshot.Package(importPath)
	targetRel := ""
	if ok && target.Dir != "" {
		rel, err := filepath.Rel(snapshot.Root, target.Dir)
		if err == nil && rel != ".." && !strings.HasPrefix(filepath.ToSlash(rel), "../") {
			targetRel = filepath.ToSlash(rel)
		}
	}
	if targetRel == "" {
		rootRel, err := filepath.Rel(snapshot.ProjectRoot, snapshot.Root)
		if err != nil {
			return "", "", false
		}
		rootImport := snapshot.ModulePath
		if rootRel != "." {
			rootImport += "/" + filepath.ToSlash(rootRel)
		}
		if importPath == rootImport {
			targetRel = "."
		} else if value, found := strings.CutPrefix(importPath, rootImport+"/"); found {
			targetRel = value
		} else {
			return "", "", false
		}
	}
	targetLayer := rulekit.LayerForRelPath(targetRel, profile.DependencyLayers)
	if targetLayer == "" {
		return "", "", false
	}
	return targetLayer, targetRel, true
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
