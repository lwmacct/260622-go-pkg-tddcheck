package filelayout

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

// identityCollisionViolations keeps one architectural identity from being
// silently split across multiple files. Package-kind layers are excluded: in
// that layout the kind is intentionally shared by several subject packages.
func identityCollisionViolations(snapshot *rulekit.Snapshot) []Violation {
	type identityKey struct {
		layer, subject, namespace, kind string
	}
	filesByIdentity := map[identityKey][]rulekit.GoFile{}
	for _, file := range snapshot.Files {
		if file.IsTest || !file.IdentityOK {
			continue
		}
		layer, ok := snapshot.Profile.Layer(file.Layer)
		if !ok || layer.FileNameMode != rulekit.FileNameModeQualifiedKind {
			continue
		}
		identity := file.Identity
		if identity.Kind == "free" {
			continue
		}
		key := identityKey{identity.Layer, identity.Subject, identity.Namespace, identity.Kind}
		filesByIdentity[key] = append(filesByIdentity[key], file)
	}
	var violations []Violation
	for key, files := range filesByIdentity {
		if len(files) < 2 {
			continue
		}
		sort.Slice(files, func(i, j int) bool { return files[i].AbsPath < files[j].AbsPath })
		label := key.subject
		if key.namespace != "" {
			label = "x." + key.namespace
		}
		for _, file := range files {
			other := files[0]
			if other.AbsPath == file.AbsPath {
				other = files[1]
			}
			violations = append(violations, Violation{
				File:    rulekit.DisplayFilename(file.AbsPath),
				Line:    1,
				Code:    RuleID + "/duplicate-identity",
				Message: fmt.Sprintf("file identity %s.%s.go is also declared by %s", label, key.kind, rulekit.DisplayFilename(other.AbsPath)),
			})
		}
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].File != violations[j].File {
			return violations[i].File < violations[j].File
		}
		return violations[i].Message < violations[j].Message
	})
	return violations
}

// publicTypeBoundaryViolations prevents persistence-shaped repository types
// from becoming part of an exported service method. It checks only public
// method signatures; repository types remain valid inside implementation
// helpers and private methods.
func publicTypeBoundaryViolations(snapshot *rulekit.Snapshot) []Violation {
	suffixes := snapshot.Profile.PublicTypeBoundarySuffixes
	if len(suffixes) == 0 {
		return nil
	}
	var violations []Violation
	for _, file := range snapshot.Files {
		if file.IsTest || file.Layer != "service" || file.AST == nil {
			continue
		}
		for _, declaration := range file.AST.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || !ast.IsExported(function.Name.Name) {
				continue
			}
			for _, field := range signatureFields(function.Type) {
				if typeName := repositoryBoundaryType(file, field.Type, suffixes); typeName != "" {
					violations = append(violations, Violation{
						File:    rulekit.DisplayFilename(file.AbsPath),
						Line:    file.Fset.Position(field.Pos()).Line,
						Code:    RuleID + "/public-type-boundary",
						Message: fmt.Sprintf("service public method %s must not expose repository type %s", function.Name.Name, typeName),
					})
				}
			}
		}
	}
	return violations
}

func signatureFields(function *ast.FuncType) []*ast.Field {
	if function == nil {
		return nil
	}
	var fields []*ast.Field
	if function.Params != nil {
		fields = append(fields, function.Params.List...)
	}
	if function.Results != nil {
		fields = append(fields, function.Results.List...)
	}
	return fields
}

func repositoryBoundaryType(file rulekit.GoFile, expression ast.Expr, suffixes []string) string {
	if semantic := file.TypeOf(expression); semantic != nil {
		if name := repositoryNamedType(semantic, suffixes); name != "" {
			return name
		}
	}
	var result string
	ast.Inspect(expression, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || result != "" {
			return true
		}
		packageName, ok := selector.X.(*ast.Ident)
		if !ok || packageName.Name != "repository" || !hasSuffix(selector.Sel.Name, suffixes) {
			return true
		}
		result = packageName.Name + "." + selector.Sel.Name
		return false
	})
	return result
}

func repositoryNamedType(value types.Type, suffixes []string) string {
	value = types.Unalias(value)
	switch typed := value.(type) {
	case *types.Pointer:
		if name := repositoryNamedType(typed.Elem(), suffixes); name != "" {
			return "*" + name
		}
	case *types.Slice:
		if name := repositoryNamedType(typed.Elem(), suffixes); name != "" {
			return "[]" + name
		}
	case *types.Array:
		if name := repositoryNamedType(typed.Elem(), suffixes); name != "" {
			return "[array]" + name
		}
	case *types.Map:
		if name := repositoryNamedType(typed.Key(), suffixes); name != "" {
			return "map[" + name + "]..."
		}
		if name := repositoryNamedType(typed.Elem(), suffixes); name != "" {
			return "map[...]" + name
		}
	case *types.Named:
		object := typed.Obj()
		if object == nil || object.Pkg() == nil || !isRepositoryPackage(object.Pkg().Path()) || !hasSuffix(object.Name(), suffixes) {
			return ""
		}
		return object.Pkg().Name() + "." + object.Name()
	}
	return ""
}

func isRepositoryPackage(path string) bool {
	return strings.HasSuffix(path, "/repository") || strings.Contains(path, "/repository/")
}

func hasSuffix(value string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
}
