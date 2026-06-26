package filelayout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

const RuleID = "filelayout"

type Rules struct {
	root   string
	config rulekit.Config
}

type Violation struct {
	File    string
	Line    int
	Message string
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
			Message: value.Message,
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
		lines = append(lines, fmt.Sprintf("%s:%d: %s", violation.File, violation.Line, violation.Message))
	}
	t.Fatalf("invalid file layout:\n  - %s", strings.Join(lines, "\n  - "))
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
		if file.Layer == "" {
			continue
		}
		violations = append(violations, violationsInFile(context.Profile, file)...)
	}
	violations = append(violations, serviceSubjectViolations(context)...)
	violations = append(violations, appcmdTransportViolations(context)...)
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
