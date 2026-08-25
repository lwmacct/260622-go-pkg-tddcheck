package filelayout

import (
	"fmt"
	"slices"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

type declContext struct {
	name fileName
	file rulekit.GoFile
}

type declPolicy struct {
	layers    []string
	namespace string
	check     func(declContext) []Violation
}

func declarationViolations(name fileName, file rulekit.GoFile, policyID string) []Violation {
	context := declContext{name: name, file: file}
	policy, ok := defaultDeclPolicies[policyID]
	if !ok {
		return []Violation{{
			File:    rulekit.DisplayFilename(file.AbsPath),
			Line:    1,
			Code:    RuleID + "/unknown-policy",
			Message: fmt.Sprintf("file kind %q references unknown declaration policy %q", name.Kind, policyID),
		}}
	}
	if policy.namespace != "" && name.Namespace != policy.namespace {
		return []Violation{{
			File:    rulekit.DisplayFilename(file.AbsPath),
			Line:    1,
			Code:    RuleID + "/policy-location",
			Message: fmt.Sprintf("file kind %q requires architecture namespace %q", name.Kind, policy.namespace),
		}}
	}
	return policy.check(context)
}

func (c declContext) layer() string {
	return c.file.Layer
}

var defaultDeclPolicies = map[string]declPolicy{
	"free":       {check: allowAnyDeclarations},
	"context":    {layers: []string{"handler"}, namespace: "http", check: checkArchitectureContext},
	"endpoint":   {layers: []string{"handler"}, namespace: "http", check: checkArchitectureEndpoint},
	"dto":        {check: checkDTO},
	"handler":    {layers: []string{"handler"}, check: checkHandler},
	"mapper":     {check: checkMapper},
	"middleware": {layers: []string{"handler"}, namespace: "http", check: checkArchitectureMiddleware},
	"commands":   {layers: []string{"service"}, check: checkCommands},
	"provider":   {layers: []string{"service"}, check: checkProvider},
	"utils":      {check: checkUtils},
	"support":    {check: checkSupport},
	"service":    {layers: []string{"service"}, check: checkService},
	"store":      {layers: []string{"repository"}, check: checkStore},
	"schema":     {layers: []string{"repository"}, check: checkSchema},
	"repository": {layers: []string{"repository"}, namespace: "store", check: checkRepository},
}

func ValidateProfile(profile rulekit.Profile) error {
	for _, layer := range profile.Layers {
		for kind, policyID := range layer.KindPolicies {
			policy, ok := defaultDeclPolicies[policyID]
			if !ok {
				return fmt.Errorf("layer %q file kind %q references unknown declaration policy %q", layer.Name, kind, policyID)
			}
			if len(policy.layers) > 0 && !slices.Contains(policy.layers, layer.Name) {
				return fmt.Errorf("declaration policy %q is not available in layer %q", policyID, layer.Name)
			}
		}
	}
	return nil
}

func allowAnyDeclarations(declContext) []Violation {
	return nil
}

func checkArchitectureContext(context declContext) []Violation {
	return architectureContextViolations(context.file.Fset, context.file.AbsPath, context.file.AST)
}

func checkArchitectureEndpoint(context declContext) []Violation {
	return architectureEndpointViolations(context.file.Fset, context.file.AbsPath, context.file.AST)
}

func checkDTO(context declContext) []Violation {
	return dtoViolations(context.file.Fset, context.file.AbsPath, context.file.AST)
}

func checkHandler(context declContext) []Violation {
	if context.name.Namespace != "" {
		return architectureHandlerViolations(context.file.Fset, context.file.AbsPath, context.file.AST)
	}
	return handlerViolations(context.file.Fset, context.file.AbsPath, context.name, context.file.AST)
}

func checkMapper(context declContext) []Violation {
	return mapperViolations(context.file.Fset, context.file.AbsPath, context.file.AST)
}

func checkArchitectureMiddleware(context declContext) []Violation {
	return architectureMiddlewareViolations(context.file.Fset, context.file.AbsPath, context.file.AST)
}

func checkCommands(context declContext) []Violation {
	return commandsViolations(context.file.Fset, context.file.AbsPath, context.file.AST)
}

func checkProvider(context declContext) []Violation {
	return providerViolations(context.file.Fset, context.file.AbsPath, context.name, context.file.AST)
}

func checkUtils(context declContext) []Violation {
	return utilsViolations(context.file.Fset, context.file.AbsPath, context.file.AST)
}

func checkArchitectureSupport(context declContext) []Violation {
	return architectureSupportViolations(context.file.Fset, context.file.AbsPath, context.file.AST)
}

func checkSupport(context declContext) []Violation {
	if context.layer() == "handler" && context.name.Namespace == "http" {
		return checkArchitectureSupport(context)
	}
	return supportViolations(context.file.Fset, context.file.AbsPath, context.layer(), context.name, context.file.AST)
}

func checkService(context declContext) []Violation {
	violations := servicePersistenceViolations(context.file)
	violations = append(violations, serviceViolations(context.file.Fset, context.file.AbsPath, context.name, context.file.AST)...)
	return violations
}

func checkStore(context declContext) []Violation {
	return storeViolations(context.file, context.name)
}

func checkSchema(context declContext) []Violation {
	return schemaViolations(context.file.Fset, context.file.AbsPath, context.name, context.file.AST)
}

func checkRepository(context declContext) []Violation {
	return repositoryViolations(context.file.Fset, context.file.AbsPath, context.name, context.file.AST)
}
