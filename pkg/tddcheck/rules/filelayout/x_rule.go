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
	if file.IsTest || file.Layer == "" {
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
		if file.IsTest {
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
	layer := file.Layer
	layerProfile, ok := profile.Layer(layer)
	if !ok {
		return nil
	}
	mode := layerProfile.FileNameMode
	parsed, ok := parseFileName(file.Base, mode)
	if !ok {
		pattern := "{subject}.{kind}.go or x.{namespace}.{kind}.go"
		if mode == rulekit.FileNameModePackageKind {
			pattern = "{kind}.go"
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
	if context.qualifiedKindMode() && !context.architectureFile() && rulekit.StringIn(context.name.subject, context.profile.ForbiddenWeakSubjects) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Message: fmt.Sprintf("subject %q is too weak; use a specific business subject", context.name.subject),
		})
	}
	if context.qualifiedKindMode() && !context.architectureFile() {
		if escapedKind, ok := escapedKindSubject(context.profile.EscapedSubjectSuffixes, context.name.subject); ok {
			violations = append(violations, Violation{
				File:    rulekit.DisplayFilename(context.file.AbsPath),
				Line:    1,
				Message: fmt.Sprintf("subject %q must not encode file kind %q; use the business subject and a single kind suffix", context.name.subject, escapedKind),
			})
		}
	}
	if context.qualifiedKindMode() && !context.architectureFile() {
		namespace := context.name.subject
		reserved := context.profile.ArchitectureNamespaceReserved(namespace)
		if !reserved {
			if legacyNamespace, ok := strings.CutPrefix(namespace, "x_"); ok {
				namespace = legacyNamespace
				reserved = context.profile.ArchitectureNamespaceReserved(namespace)
			}
		}
		if reserved {
			violations = append(violations, Violation{
				File:    rulekit.DisplayFilename(context.file.AbsPath),
				Line:    1,
				Message: fmt.Sprintf("architecture namespace %q must use x.%s.{kind}.go naming", namespace, namespace),
			})
		}
	}
	if context.qualifiedKindMode() && context.architectureFile() && !context.profile.ArchitectureNamespaceReserved(context.name.namespace) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Message: fmt.Sprintf("architecture namespace %q is not reserved", context.name.namespace),
		})
	}
	if !context.profile.KindAllowed(context.file.Layer, context.name.kind) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Message: fmt.Sprintf("%s file kind %q is not allowed", context.file.Layer, context.name.kind),
		})
	}
	if context.qualifiedKindMode() && context.architectureFile() && !context.profile.ArchitectureNamespaceAllowed(context.file.Layer, context.name.namespace) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(context.file.AbsPath),
			Line:    1,
			Message: fmt.Sprintf("architecture namespace %q is not allowed in %s", context.name.namespace, context.file.Layer),
		})
	}
	if context.qualifiedKindMode() {
		violations = append(violations, inferredSubjectViolations(context.name, context.file)...)
	}

	violations = append(violations, declarationViolations(context.name, context.file)...)
	return violations
}

func (f layoutFile) qualifiedKindMode() bool {
	return f.mode == rulekit.FileNameModeQualifiedKind
}

func (f layoutFile) architectureFile() bool {
	return f.name.namespace != ""
}
