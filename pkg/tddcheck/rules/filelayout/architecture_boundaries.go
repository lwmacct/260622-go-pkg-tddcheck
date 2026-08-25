package filelayout

import (
	"fmt"
	"go/ast"
	"go/token"
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
		if file.IsTest || file.Layer != "service" || file.AST == nil || (file.IdentityOK && file.Identity.Kind == "free") {
			continue
		}
		for _, declaration := range file.AST.Decls {
			switch typed := declaration.(type) {
			case *ast.FuncDecl:
				if !ast.IsExported(typed.Name.Name) {
					continue
				}
				for _, field := range signatureFields(typed.Type) {
					if typeName := repositoryBoundaryType(file, field.Type, suffixes); typeName != "" {
						kind := "function"
						if typed.Recv != nil {
							kind = "method"
						}
						violations = append(violations, Violation{
							File:    rulekit.DisplayFilename(file.AbsPath),
							Line:    file.Fset.Position(field.Pos()).Line,
							Code:    RuleID + "/public-type-boundary",
							Message: fmt.Sprintf("service public %s %s must not expose repository type %s", kind, typed.Name.Name, typeName),
						})
					}
				}
			case *ast.GenDecl:
				if typed.Tok != token.TYPE {
					continue
				}
				for _, spec := range typed.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !ast.IsExported(typeSpec.Name.Name) {
						continue
					}
					if typeName := repositoryBoundaryType(file, typeSpec.Type, suffixes); typeName != "" {
						violations = append(violations, Violation{
							File:    rulekit.DisplayFilename(file.AbsPath),
							Line:    file.Fset.Position(typeSpec.Pos()).Line,
							Code:    RuleID + "/public-type-boundary",
							Message: fmt.Sprintf("service public type %s must not expose repository type %s", typeSpec.Name.Name, typeName),
						})
					}
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
		if name := repositoryNamedTypeDeep(semantic, suffixes, map[string]bool{}); name != "" {
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

func repositoryNamedTypeDeep(value types.Type, suffixes []string, seen map[string]bool) string {
	value = types.Unalias(value)
	switch typed := value.(type) {
	case *types.Pointer:
		return repositoryNamedTypeDeep(typed.Elem(), suffixes, seen)
	case *types.Slice:
		return repositoryNamedTypeDeep(typed.Elem(), suffixes, seen)
	case *types.Array:
		return repositoryNamedTypeDeep(typed.Elem(), suffixes, seen)
	case *types.Map:
		if name := repositoryNamedTypeDeep(typed.Key(), suffixes, seen); name != "" {
			return name
		}
		return repositoryNamedTypeDeep(typed.Elem(), suffixes, seen)
	case *types.Chan:
		return repositoryNamedTypeDeep(typed.Elem(), suffixes, seen)
	case *types.Named:
		object := typed.Obj()
		if object == nil {
			return ""
		}
		if object.Pkg() != nil && isRepositoryPackage(object.Pkg().Path()) {
			if hasSuffix(object.Name(), suffixes) {
				return object.Pkg().Name() + "." + object.Name()
			}
			return ""
		}
		pkgPath := ""
		if object.Pkg() != nil {
			pkgPath = object.Pkg().Path()
		}
		key := pkgPath + "\x00" + object.Name()
		if seen[key] {
			return ""
		}
		seen[key] = true
		return repositoryNamedTypeDeep(typed.Underlying(), suffixes, seen)
	case *types.Struct:
		for index := 0; index < typed.NumFields(); index++ {
			if name := repositoryNamedTypeDeep(typed.Field(index).Type(), suffixes, seen); name != "" {
				return name
			}
		}
	case *types.Interface:
		for index := 0; index < typed.NumMethods(); index++ {
			if name := repositoryNamedTypeDeep(typed.Method(index).Type(), suffixes, seen); name != "" {
				return name
			}
		}
	case *types.Signature:
		if name := repositoryTupleType(typed.Params(), suffixes, seen); name != "" {
			return name
		}
		return repositoryTupleType(typed.Results(), suffixes, seen)
	}
	return ""
}

func repositoryTupleType(tuple *types.Tuple, suffixes []string, seen map[string]bool) string {
	if tuple == nil {
		return ""
	}
	for index := 0; index < tuple.Len(); index++ {
		if name := repositoryNamedTypeDeep(tuple.At(index).Type(), suffixes, seen); name != "" {
			return name
		}
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
