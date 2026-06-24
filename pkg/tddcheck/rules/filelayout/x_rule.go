package filelayout

import (
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
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
	root, err := rulekit.ResolveRuleRoot(r.root, RuleID)
	if err != nil {
		return nil, err
	}
	files, err := rulekit.ModuleFiles(root, RuleID, r.config, func(name string) bool {
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	})
	if err != nil {
		return nil, err
	}

	config := r.config.WithDefaults()
	var violations []Violation
	for _, file := range files {
		if isFreeFile(file) {
			continue
		}
		layer, ok := layerForFile(root, file, config)
		if !ok {
			continue
		}
		fileViolations, err := violationsInFile(config, layer, file)
		if err != nil {
			return nil, err
		}
		violations = append(violations, fileViolations...)
	}
	violations = append(violations, serviceSubjectViolations(root, files, config)...)
	appcmdViolations, err := appcmdTransportViolations(root, files, config)
	if err != nil {
		return nil, err
	}
	violations = append(violations, appcmdViolations...)
	return violations, nil
}

func violationsInFile(config rulekit.Config, layer string, filename string) ([]Violation, error) {
	if isFreeFile(filename) {
		return nil, nil
	}
	profile := profileFromConfig(config)
	mode := config.LayerFileNameModes[layer]
	if mode == "" {
		mode = rulekit.FileNameModeScopeKind
	}
	base := filepath.Base(filename)
	parsed, ok := parseFileName(base, mode)
	if !ok {
		pattern := "{scope}.{type}.go"
		if mode == rulekit.FileNameModePackageKind {
			pattern = "{type}.go"
		}
		return []Violation{{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("%s file must use %s naming", layer, pattern),
		}}, nil
	}

	var violations []Violation
	if mode == rulekit.FileNameModeScopeKind && rulekit.StringIn(parsed.scope, config.ForbiddenWeakScopes) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("scope %q is too weak; use a subject name or approved shared scope", parsed.scope),
		})
	}
	if mode == rulekit.FileNameModeScopeKind && !strings.HasPrefix(parsed.scope, architectureScopePrefix) {
		if escapedKind, ok := escapedKindScope(profile, parsed.scope); ok {
			violations = append(violations, Violation{
				File:    rulekit.DisplayFilename(filename),
				Line:    1,
				Message: fmt.Sprintf("scope %q must not encode file type %q; use the subject scope and a single type suffix", parsed.scope, escapedKind),
			})
		}
	}
	if mode == rulekit.FileNameModeScopeKind && !strings.HasPrefix(parsed.scope, architectureScopePrefix) && profile.architectureScopeReserved(parsed.scope) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q must use x_%s prefix", parsed.scope, parsed.scope),
		})
	}
	if mode == rulekit.FileNameModeScopeKind && strings.HasPrefix(parsed.scope, architectureScopePrefix) && !profile.architectureScopeReserved(strings.TrimPrefix(parsed.scope, architectureScopePrefix)) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q is not reserved", parsed.scope),
		})
	}
	if !profile.kindAllowed(layer, parsed.kind) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("%s file type %q is not allowed", layer, parsed.kind),
		})
	}
	if mode == rulekit.FileNameModeScopeKind && strings.HasPrefix(parsed.scope, architectureScopePrefix) && !profile.architectureScopeAllowed(layer, parsed.scope) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q is not allowed in %s", parsed.scope, layer),
		})
	}
	if mode == rulekit.FileNameModeScopeKind {
		scopeViolations, err := inferredScopeViolations(parsed, filename)
		if err != nil {
			return nil, err
		}
		violations = append(violations, scopeViolations...)
	}

	declViolations, err := declarationViolations(layer, parsed, filename)
	if err != nil {
		return nil, err
	}
	violations = append(violations, declViolations...)
	return violations, nil
}

func declarationViolations(layer string, name fileName, filename string) ([]Violation, error) {
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, filename, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	switch name.kind {
	case "endpoint":
		if layer == "handler" && name.scope == "x_http" {
			return architectureEndpointViolations(fileSet, filename, parsedFile), nil
		}
	case "dto":
		return dtoViolations(fileSet, filename, parsedFile), nil
	case "handler":
		if layer == "handler" {
			if strings.HasPrefix(name.scope, architectureScopePrefix) {
				return architectureHandlerViolations(fileSet, filename, parsedFile), nil
			}
			return handlerViolations(fileSet, filename, name, parsedFile), nil
		}
	case "mapper":
		return mapperViolations(fileSet, filename, parsedFile), nil
	case "middleware":
		if layer == "handler" && name.scope == "x_http" {
			return architectureMiddlewareViolations(fileSet, filename, parsedFile), nil
		}
	case "commands":
		return commandsViolations(fileSet, filename, parsedFile), nil
	case "provider":
		if layer == "service" {
			return providerViolations(fileSet, filename, name, parsedFile), nil
		}
	case "utils":
		return utilsViolations(fileSet, filename, parsedFile), nil
	case "support":
		if layer == "handler" && name.scope == "x_http" {
			return architectureSupportViolations(fileSet, filename, parsedFile), nil
		}
		return supportViolations(fileSet, filename, layer, name, parsedFile), nil
	case "service":
		if layer == "service" {
			if name.scope == "x_batch" {
				return nil, nil
			}
			violations := servicePersistenceViolations(fileSet, filename, parsedFile)
			violations = append(violations, serviceViolations(fileSet, filename, name, parsedFile)...)
			return violations, nil
		}
	case "store":
		if layer == "repository" {
			return storeViolations(fileSet, filename, name, parsedFile), nil
		}
	case "schema":
		if layer == "repository" {
			return schemaViolations(fileSet, filename, parsedFile), nil
		}
	case "repository":
		if layer == "repository" {
			return repositoryViolations(fileSet, filename, name, parsedFile), nil
		}
	}
	return nil, nil
}

func isFreeFile(filename string) bool {
	return filepath.Base(filename) == "x_free.go"
}
