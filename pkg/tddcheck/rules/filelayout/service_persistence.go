package filelayout

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func servicePersistenceViolations(file rulekit.GoFile) []Violation {
	var violations []Violation
	for _, decl := range file.AST.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			violations = append(violations, servicePersistenceGenDeclViolations(file, typed)...)
		case *ast.FuncDecl:
			violations = append(violations, servicePersistenceFuncViolations(file, typed)...)
		}
	}
	return violations
}

func servicePersistenceGenDeclViolations(file rulekit.GoFile, decl *ast.GenDecl) []Violation {
	var violations []Violation
	switch decl.Tok {
	case token.IMPORT:
		for _, importSpec := range decl.Specs {
			spec, ok := importSpec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			path := importPath(spec)
			if forbiddenServicePersistenceImport(path) {
				violations = append(violations, violationAt(file.Fset, file.AbsPath, spec.Pos(), "service files must not import persistence package "+path))
			}
		}
	case token.TYPE:
		for _, spec := range decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			ast.Inspect(typeSpec.Type, func(node ast.Node) bool {
				expr, ok := node.(ast.Expr)
				if !ok {
					return true
				}
				if forbiddenServicePersistenceType(file, expr) {
					violations = append(violations, violationAt(file.Fset, file.AbsPath, expr.Pos(), "service files must not depend on persistence handle types"))
					return false
				}
				return true
			})
		}
	}
	return violations
}

func servicePersistenceFuncViolations(file rulekit.GoFile, decl *ast.FuncDecl) []Violation {
	var violations []Violation
	if decl.Type != nil {
		ast.Inspect(decl.Type, func(node ast.Node) bool {
			expr, ok := node.(ast.Expr)
			if !ok {
				return true
			}
			if forbiddenServicePersistenceType(file, expr) {
				violations = append(violations, violationAt(file.Fset, file.AbsPath, expr.Pos(), "service files must not depend on persistence handle types"))
				return false
			}
			return true
		})
	}
	if decl.Body == nil {
		return violations
	}
	ast.Inspect(decl.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			if forbiddenServicePersistenceCall(file, typed.Fun) {
				violations = append(violations, violationAt(file.Fset, file.AbsPath, typed.Fun.Pos(), "service files must not call persistence APIs directly"))
				return false
			}
		case *ast.SelectorExpr:
			if forbiddenRepositoryModelSelector(file, typed) {
				violations = append(violations, violationAt(file.Fset, file.AbsPath, typed.Pos(), "service files must not reference repository schema models"))
				return false
			}
		}
		return true
	})
	return violations
}

func forbiddenServicePersistenceImport(importPath string) bool {
	if strings.HasPrefix(importPath, "github.com/uptrace/bun") ||
		strings.HasPrefix(importPath, "gorm.io/driver/") ||
		strings.HasPrefix(importPath, "github.com/jackc/pgx/") ||
		strings.HasPrefix(importPath, "entgo.io/ent") ||
		strings.HasPrefix(importPath, "github.com/volatiletech/sqlboiler") ||
		strings.HasPrefix(importPath, "github.com/upper/db") ||
		strings.HasPrefix(importPath, "go.mongodb.org/mongo-driver/mongo") ||
		strings.HasPrefix(importPath, "cloud.google.com/go/firestore") ||
		strings.HasPrefix(importPath, "github.com/aws/aws-sdk-go-v2/service/dynamodb") {
		return true
	}
	return oneOf(
		importPath,
		"database/sql",
		"gorm.io/gorm",
		"github.com/jmoiron/sqlx",
		"github.com/lib/pq",
		"github.com/go-sql-driver/mysql",
		"github.com/mattn/go-sqlite3",
		"modernc.org/sqlite",
		"xorm.io/xorm",
	)
}

func forbiddenServicePersistenceType(file rulekit.GoFile, expr ast.Expr) bool {
	if semanticType := file.TypeOf(expr); semanticType != nil {
		return forbiddenPersistenceType(semanticType)
	}
	return forbiddenServicePersistenceTypeSyntax(expr)
}

func forbiddenPersistenceType(value types.Type) bool {
	value = types.Unalias(value)
	switch typed := value.(type) {
	case *types.Pointer:
		return forbiddenPersistenceType(typed.Elem())
	case *types.Slice:
		return forbiddenPersistenceType(typed.Elem())
	case *types.Array:
		return forbiddenPersistenceType(typed.Elem())
	case *types.Map:
		return forbiddenPersistenceType(typed.Key()) || forbiddenPersistenceType(typed.Elem())
	case *types.Chan:
		return forbiddenPersistenceType(typed.Elem())
	case *types.Named:
		object := typed.Obj()
		if object == nil || object.Pkg() == nil {
			return false
		}
		path := object.Pkg().Path()
		name := object.Name()
		switch path {
		case "database/sql", "github.com/jmoiron/sqlx":
			return oneOf(name, "DB", "Tx", "Conn", "Stmt", "Rows", "Row")
		case "github.com/uptrace/bun":
			return name == "DB" || name == "Tx" || name == "IDB" || strings.HasSuffix(name, "Query")
		case "gorm.io/gorm":
			return name == "DB"
		default:
			return forbiddenServicePersistenceImport(path)
		}
	}
	return false
}

func forbiddenServicePersistenceTypeSyntax(expr ast.Expr) bool {
	switch typed := expr.(type) {
	case *ast.StarExpr:
		return forbiddenServicePersistenceTypeSyntax(typed.X)
	case *ast.SelectorExpr:
		pkg := selectorPackage(typed)
		name := typed.Sel.Name
		switch pkg {
		case "sql":
			return name == "DB" || name == "Tx" || name == "Conn" || name == "Stmt" || name == "Rows" || name == "Row"
		case "bun":
			return name == "DB" || name == "Tx" || name == "IDB" || strings.HasSuffix(name, "Query")
		case "gorm":
			return name == "DB"
		case "sqlx":
			return name == "DB" || name == "Tx" || name == "Conn" || name == "Stmt" || name == "Rows" || name == "Row"
		case "pgx", "pgxpool", "mongo", "firestore", "dynamodb", "ent", "xorm":
			return true
		}
	case *ast.ArrayType:
		return forbiddenServicePersistenceTypeSyntax(typed.Elt)
	case *ast.MapType:
		return forbiddenServicePersistenceTypeSyntax(typed.Key) || forbiddenServicePersistenceTypeSyntax(typed.Value)
	case *ast.ChanType:
		return forbiddenServicePersistenceTypeSyntax(typed.Value)
	}
	return false
}

func forbiddenServicePersistenceCall(file rulekit.GoFile, expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	name := selector.Sel.Name
	if forbiddenRepositoryModelSelector(file, selector) {
		return true
	}
	if file.TypesInfo != nil {
		if object := file.TypesInfo.ObjectOf(selector.Sel); object != nil {
			if object.Pkg() == nil || !forbiddenServicePersistenceImport(object.Pkg().Path()) {
				return false
			}
			return persistenceCallName(name)
		}
	}
	if selectorPackage(selector) == "sql" && oneOf(name, "Open", "OpenDB") {
		return true
	}
	if name == "RunInTx" {
		return selectorReceiverName(selector) == "db"
	}
	return persistenceCallName(name)
}

func persistenceCallName(name string) bool {
	return oneOf(
		name,
		"Open",
		"OpenDB",
		"RunInTx",
		"NewSelect",
		"NewInsert",
		"NewUpdate",
		"NewDelete",
		"NewRaw",
		"ScanAndCount",
		"Query",
		"QueryContext",
		"QueryRow",
		"QueryRowContext",
		"Exec",
		"ExecContext",
		"Prepare",
		"PrepareContext",
		"Begin",
		"BeginTx",
		"BeginTxx",
		"Transaction",
		"Raw",
		"Table",
		"AutoMigrate",
	)
}

func forbiddenRepositoryModelSelector(file rulekit.GoFile, expr *ast.SelectorExpr) bool {
	if file.TypesInfo != nil {
		semanticObject := file.TypesInfo.ObjectOf(expr.Sel)
		if semanticObject != nil {
			object, ok := semanticObject.(*types.TypeName)
			if !ok || object.Pkg() == nil {
				return false
			}
			path := object.Pkg().Path()
			isRepository := strings.HasSuffix(path, "/repository") || strings.Contains(path, "/repository/")
			return isRepository && strings.HasSuffix(object.Name(), "Model")
		}
	}
	return selectorPackage(expr) == "repository" && strings.HasSuffix(expr.Sel.Name, "Model")
}

func selectorReceiverName(expr *ast.SelectorExpr) string {
	ident, ok := expr.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}
