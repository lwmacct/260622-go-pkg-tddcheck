package filelayout

import (
	"context"
	"fmt"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

const RuleID = "filelayout"

type Violation struct {
	File    string
	Line    int
	Column  int
	Message string
}

func Register(engine *rulekit.Engine) {
	engine.Register(RuleID, rulekit.FileScope, checkFile)
	engine.Register(RuleID, rulekit.SnapshotScope, checkSnapshot)
}

func Check(ctx context.Context, root string, config rulekit.Config) ([]Violation, error) {
	config, err := config.Compile()
	if err != nil {
		return nil, err
	}
	if err := config.ValidateFileLayout(); err != nil {
		return nil, err
	}
	snapshot, err := rulekit.Load(ctx, root, config)
	if err != nil {
		return nil, err
	}
	return violationsInSnapshot(snapshot), nil
}

func checkFile(_ context.Context, snapshot *rulekit.Snapshot, file rulekit.GoFile) ([]rulekit.Diagnostic, error) {
	if file.IsTest || rulekit.FreeFile(file.Base) || file.Layer == "" {
		return nil, nil
	}
	return diagnostics(violationsInFile(snapshot.Profile, file)), nil
}

func checkSnapshot(_ context.Context, _ *rulekit.Snapshot, snapshot *rulekit.Snapshot) ([]rulekit.Diagnostic, error) {
	values := serviceSubjectViolations(snapshot)
	values = append(values, appcmdTransportViolations(snapshot)...)
	return diagnostics(values), nil
}

func diagnostics(values []Violation) []rulekit.Diagnostic {
	diagnostics := make([]rulekit.Diagnostic, 0, len(values))
	for _, value := range values {
		position := rulekit.Position{File: value.File, Line: value.Line, Column: value.Column}
		diagnostics = append(diagnostics, rulekit.NewDiagnostic(RuleID, rulekit.SeverityError, value.Message, position, position))
	}
	return diagnostics
}

func violationsInSnapshot(snapshot *rulekit.Snapshot) []Violation {
	var violations []Violation
	for _, file := range snapshot.Files {
		if file.IsTest || rulekit.FreeFile(file.Base) {
			continue
		}
		if file.Layer == "" {
			continue
		}
		violations = append(violations, violationsInFile(snapshot.Profile, file)...)
	}
	violations = append(violations, serviceSubjectViolations(snapshot)...)
	violations = append(violations, appcmdTransportViolations(snapshot)...)
	return violations
}

func violationsInFile(profile rulekit.Profile, file rulekit.GoFile) []Violation {
	if rulekit.FreeFile(file.Base) {
		return nil
	}
	layer := file.Layer
	layerProfile, ok := profile.Layer(layer)
	if !ok {
		return nil
	}
	mode := layerProfile.FileNameMode
	parsed, ok := parseFileName(file.Base, mode)
	if !ok {
		pattern := "{scope}.{type}.go"
		if mode == rulekit.FileNameModePackageKind {
			pattern = "{type}.go"
		}
		return []Violation{{
			File:    rulekit.DisplayFilename(file.AbsPath),
			Line:    1,
			Message: fmt.Sprintf("%s file must use %s naming", layer, pattern),
		}}
	}

	context := layoutFile{
		profile: profile,
		mode:    mode,
		name:    parsed,
		file:    file,
	}
	return layoutFileViolations(context)
}

type layoutFile struct {
	profile rulekit.Profile
	mode    string
	name    fileName
	file    rulekit.GoFile
}

func layoutFileViolations(context layoutFile) []Violation {
	var violations []Violation
	if context.scopeKindMode() && rulekit.StringIn(context.name.scope, context.profile.ForbiddenWeakScopes) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Message: fmt.Sprintf("scope %q is too weak; use a subject name or approved shared scope", context.name.scope),
		})
	}
	if context.scopeKindMode() && !context.architectureScope() {
		if escapedKind, ok := escapedKindScope(context.profile.EscapedScopeSuffixes, context.name.scope); ok {
			violations = append(violations, Violation{
				File:    rulekit.DisplayFilename(context.file.AbsPath),
				Line:    1,
				Message: fmt.Sprintf("scope %q must not encode file type %q; use the subject scope and a single type suffix", context.name.scope, escapedKind),
			})
		}
	}
	if context.scopeKindMode() && !context.architectureScope() && context.profile.ArchitectureScopeReserved(context.name.scope) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q must use x_%s prefix", context.name.scope, context.name.scope),
		})
	}
	if context.scopeKindMode() && context.architectureScope() && !context.profile.ArchitectureScopeReserved(strings.TrimPrefix(context.name.scope, architectureScopePrefix)) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q is not reserved", context.name.scope),
		})
	}
	if !context.profile.KindAllowed(context.file.Layer, context.name.kind) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Message: fmt.Sprintf("%s file type %q is not allowed", context.file.Layer, context.name.kind),
		})
	}
	if context.scopeKindMode() && context.architectureScope() && !context.profile.ArchitectureScopeAllowed(context.file.Layer, context.name.scope) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q is not allowed in %s", context.name.scope, context.file.Layer),
		})
	}
	if context.scopeKindMode() {
		violations = append(violations, inferredScopeViolations(context.name, context.file)...)
	}

	violations = append(violations, declarationViolations(context.name, context.file)...)
	return violations
}

func (f layoutFile) scopeKindMode() bool {
	return f.mode == rulekit.FileNameModeScopeKind
}

func (f layoutFile) architectureScope() bool {
	return strings.HasPrefix(f.name.scope, architectureScopePrefix)
}
