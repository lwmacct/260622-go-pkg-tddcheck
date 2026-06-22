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
	return violations, nil
}

func violationsInFile(config rulekit.Config, layer string, filename string) ([]Violation, error) {
	profile := hsrStrictProfile()
	base := filepath.Base(filename)
	parsed, ok := parseFileName(base)
	if !ok {
		return []Violation{{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("%s file must use {scope}.{type}.go naming", layer),
		}}, nil
	}

	var violations []Violation
	if rulekit.StringIn(parsed.scope, config.ForbiddenWeakScopes) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("scope %q is too weak; use a resource name or approved shared scope", parsed.scope),
		})
	}
	if !strings.HasPrefix(parsed.scope, architectureScopePrefix) {
		if escapedKind, ok := escapedKindScope(profile, parsed.scope); ok {
			violations = append(violations, Violation{
				File:    rulekit.DisplayFilename(filename),
				Line:    1,
				Message: fmt.Sprintf("scope %q must not encode file type %q; use the resource scope and a single type suffix", parsed.scope, escapedKind),
			})
		}
	}
	if !strings.HasPrefix(parsed.scope, architectureScopePrefix) && profile.architectureScopeReserved(parsed.scope) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q must use x_%s prefix", parsed.scope, parsed.scope),
		})
	}
	if strings.HasPrefix(parsed.scope, architectureScopePrefix) && !profile.architectureScopeReserved(strings.TrimPrefix(parsed.scope, architectureScopePrefix)) {
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
	if strings.HasPrefix(parsed.scope, architectureScopePrefix) && !profile.architectureKindAllowed(layer, parsed.scope, parsed.kind) {
		violations = append(violations, Violation{
			File:    rulekit.DisplayFilename(filename),
			Line:    1,
			Message: fmt.Sprintf("architecture scope %q must not use %s.%s.go in %s", parsed.scope, parsed.scope, parsed.kind, layer),
		})
	}
	scopeViolations, err := inferredScopeViolations(parsed, filename)
	if err != nil {
		return nil, err
	}
	violations = append(violations, scopeViolations...)

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
	case "commands":
		return commandsViolations(fileSet, filename, parsedFile), nil
	case "constants":
		return constantsViolations(fileSet, filename, parsedFile), nil
	case "validation":
		return validationViolations(fileSet, filename, parsedFile), nil
	case "errors":
		return errorsViolations(fileSet, filename, parsedFile), nil
	case "utils":
		return utilsViolations(fileSet, filename, parsedFile), nil
	case "service":
		if layer == "service" {
			if name.scope == "x_batch" {
				return nil, nil
			}
			return serviceViolations(fileSet, filename, name, parsedFile), nil
		}
	case "store":
		if layer == "repository" {
			return storeViolations(fileSet, filename, parsedFile), nil
		}
	case "schema":
		if layer == "repository" {
			return schemaViolations(fileSet, filename, parsedFile), nil
		}
	case "repository":
		if layer == "repository" {
			return repositoryViolations(fileSet, filename, name, parsedFile), nil
		}
	case "model":
		if layer == "repository" && name.kind == "model" {
			return repositoryModelViolations(fileSet, filename, parsedFile), nil
		}
		return typeOnlyViolations(fileSet, filename, parsedFile, name.kind+" files must only declare types"), nil
	}
	return nil, nil
}
