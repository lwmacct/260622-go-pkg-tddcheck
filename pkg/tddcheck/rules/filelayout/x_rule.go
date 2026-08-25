package filelayout

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

const RuleID = "filelayout"

type Violation struct {
	File     string
	Line     int
	Column   int
	Code     string
	Severity rulekit.Severity
	Message  string
	Fix      *rulekit.SuggestedFix
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
	if err := ValidateProfile(config.Profile()); err != nil {
		return nil, err
	}
	snapshot, err := rulekit.Load(ctx, root, config)
	if err != nil {
		return nil, err
	}
	return violationsInSnapshot(snapshot), nil
}

func checkFile(_ context.Context, snapshot *rulekit.Snapshot, file rulekit.GoFile) ([]rulekit.Diagnostic, error) {
	if file.IsTest || file.Layer == "" {
		return nil, nil
	}
	return diagnostics(violationsInFile(snapshot.Profile, file)), nil
}

func checkSnapshot(_ context.Context, _ *rulekit.Snapshot, snapshot *rulekit.Snapshot) ([]rulekit.Diagnostic, error) {
	values := subjectAnchorViolations(snapshot)
	values = append(values, identityCollisionViolations(snapshot)...)
	values = append(values, publicTypeBoundaryViolations(snapshot)...)
	return diagnostics(values), nil
}

func diagnostics(values []Violation) []rulekit.Diagnostic {
	diagnostics := make([]rulekit.Diagnostic, 0, len(values))
	for _, value := range values {
		position := rulekit.Position{File: value.File, Line: value.Line, Column: value.Column}
		code := value.Code
		if code == "" {
			code = RuleID + "/policy"
		}
		severity := value.Severity
		if severity == "" {
			severity = rulekit.SeverityError
		}
		diagnostic := rulekit.NewDiagnostic(RuleID, code, severity, value.Message, position, position)
		diagnostic.SuggestedFix = value.Fix
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics
}

func violationsInSnapshot(snapshot *rulekit.Snapshot) []Violation {
	var violations []Violation
	for _, file := range snapshot.Files {
		if file.IsTest {
			continue
		}
		if file.Layer == "" {
			continue
		}
		violations = append(violations, violationsInFile(snapshot.Profile, file)...)
	}
	violations = append(violations, subjectAnchorViolations(snapshot)...)
	violations = append(violations, identityCollisionViolations(snapshot)...)
	violations = append(violations, publicTypeBoundaryViolations(snapshot)...)
	return violations
}

func violationsInFile(profile rulekit.Profile, file rulekit.GoFile) []Violation {
	layer := file.Layer
	layerProfile, ok := profile.Layer(layer)
	if !ok {
		return nil
	}
	mode := layerProfile.FileNameMode
	parsed, ok := file.Identity, file.IdentityOK
	if !ok {
		pattern := "{subject}.{kind}.go or x.{namespace}.{kind}.go"
		if mode == rulekit.FileNameModePackageKind {
			pattern = "{kind}.go"
		}
		return []Violation{{
			File:    rulekit.DisplayFilename(file.AbsPath),
			Line:    1,
			Code:    RuleID + "/invalid-filename",
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
	policyID, kindAllowed := context.profile.KindPolicy(context.file.Layer, context.name.Kind)
	if !kindAllowed {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Code:    RuleID + "/kind-not-allowed",
			Message: fmt.Sprintf("%s file kind %q is not allowed", context.file.Layer, context.name.Kind),
		})
	}
	// free is an explicit escape hatch for declaration content and identity
	// qualifiers. The filename still must be syntactically valid and its kind
	// must be enabled by the layer profile.
	if context.name.Kind == "free" {
		violations = append(violations, Violation{
			File:     rulekit.DisplayFilename(context.file.AbsPath),
			Line:     1,
			Code:     RuleID + "/free-file",
			Severity: rulekit.SeverityWarning,
			Message:  "free files bypass declaration and subject ownership checks; classify this file when possible",
		})
		return violations
	}
	if context.qualifiedKindMode() && !context.architectureFile() && rulekit.StringIn(context.name.Subject, context.profile.ForbiddenWeakSubjects) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Code:    RuleID + "/weak-subject",
			Message: fmt.Sprintf("subject %q is too weak; use a specific business subject", context.name.Subject),
		})
	}
	if context.qualifiedKindMode() && !context.architectureFile() {
		if escapedKind, ok := escapedKindSubject(context.profile.EscapedSubjectSuffixes, context.name.Subject); ok {
			violations = append(violations, Violation{
				File:    rulekit.DisplayFilename(context.file.AbsPath),
				Line:    1,
				Code:    RuleID + "/kind-in-subject",
				Message: fmt.Sprintf("subject %q must not encode file kind %q; use the business subject and a single kind suffix", context.name.Subject, escapedKind),
			})
		}
	}
	if context.qualifiedKindMode() && !context.architectureFile() {
		if namespace, legacy := strings.CutPrefix(context.name.Subject, "x_"); legacy {
			oldFile := rulekit.DisplayFilename(context.file.AbsPath)
			newFile := path.Join(path.Dir(oldFile), "x."+namespace+"."+context.name.Kind+".go")
			violations = append(violations, Violation{
				File:    oldFile,
				Line:    1,
				Code:    RuleID + "/legacy-namespace",
				Message: fmt.Sprintf("architecture namespace %q must use x.%s.{kind}.go naming", namespace, namespace),
				Fix: &rulekit.SuggestedFix{
					Message: "rename legacy architecture file",
					Rename:  &rulekit.RenameFix{From: oldFile, To: newFile},
				},
			})
		}
	}
	if context.qualifiedKindMode() && context.architectureFile() && !context.profile.ArchitectureNamespaceAllowed(context.file.Layer, context.name.Namespace) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Code:    RuleID + "/namespace-not-allowed",
			Message: fmt.Sprintf("architecture namespace %q is not allowed in %s", context.name.Namespace, context.file.Layer),
		})
	}
	var subjectIdentityViolations []Violation
	if context.qualifiedKindMode() && context.name.Kind != "free" {
		subjectIdentityViolations = inferredSubjectViolations(context.name, context.file)
		violations = append(violations, subjectIdentityViolations...)
	}
	if context.qualifiedKindMode() && !context.architectureFile() && len(subjectIdentityViolations) == 0 && !strings.HasPrefix(context.name.Subject, "x_") {
		violations = append(violations, subjectPrefixViolations(context.name, context.file)...)
	}

	violations = append(violations, declarationViolations(context.profile, context.name, context.file, policyID)...)
	return violations
}

func (f layoutFile) qualifiedKindMode() bool {
	return f.mode == rulekit.FileNameModeQualifiedKind
}

func (f layoutFile) architectureFile() bool {
	return f.name.Architecture()
}
