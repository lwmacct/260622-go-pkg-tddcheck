package filelayout

import (
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

type declContext struct {
	name fileName
	file rulekit.GoFile
}

type declPolicy struct {
	kind  string
	match func(declContext) bool
	check func(declContext) []Violation
}

func declarationViolations(name fileName, file rulekit.GoFile) []Violation {
	context := declContext{name: name, file: file}
	for _, policy := range defaultDeclPolicies {
		if policy.kind != name.kind {
			continue
		}
		if policy.match != nil && !policy.match(context) {
			continue
		}
		return policy.check(context)
	}
	return nil
}

func (c declContext) layer() string {
	return c.file.Layer
}

var defaultDeclPolicies = []declPolicy{
	{kind: "context", match: layerScope("handler", "x_http"), check: checkArchitectureContext},
	{kind: "endpoint", match: layerScope("handler", "x_http"), check: checkArchitectureEndpoint},
	{kind: "dto", check: checkDTO},
	{kind: "handler", match: layer("handler"), check: checkHandler},
	{kind: "mapper", check: checkMapper},
	{kind: "middleware", match: layerScope("handler", "x_http"), check: checkArchitectureMiddleware},
	{kind: "commands", check: checkCommands},
	{kind: "provider", match: layer("service"), check: checkProvider},
	{kind: "utils", check: checkUtils},
	{kind: "support", match: architectureSupport, check: checkArchitectureSupport},
	{kind: "support", check: checkSupport},
	{kind: "service", match: layer("service"), check: checkService},
	{kind: "store", match: layer("repository"), check: checkStore},
	{kind: "schema", match: layer("repository"), check: checkSchema},
	{kind: "repository", match: layer("repository"), check: checkRepository},
}

func layer(value string) func(declContext) bool {
	return func(context declContext) bool {
		return context.layer() == value
	}
}

func layerScope(layerValue string, scope string) func(declContext) bool {
	return func(context declContext) bool {
		return context.layer() == layerValue && context.name.scope == scope
	}
}

func architectureSupport(context declContext) bool {
	return context.layer() == "handler" && context.name.scope == "x_http"
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
	if strings.HasPrefix(context.name.scope, architectureScopePrefix) {
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
	return supportViolations(context.file.Fset, context.file.AbsPath, context.layer(), context.name, context.file.AST)
}

func checkService(context declContext) []Violation {
	violations := servicePersistenceViolations(context.file.Fset, context.file.AbsPath, context.file.AST)
	violations = append(violations, serviceViolations(context.file.Fset, context.file.AbsPath, context.name, context.file.AST)...)
	return violations
}

func checkStore(context declContext) []Violation {
	return storeViolations(context.file.Fset, context.file.AbsPath, context.name, context.file.AST)
}

func checkSchema(context declContext) []Violation {
	return schemaViolations(context.file.Fset, context.file.AbsPath, context.name, context.file.AST)
}

func checkRepository(context declContext) []Violation {
	return repositoryViolations(context.file.Fset, context.file.AbsPath, context.name, context.file.AST)
}
