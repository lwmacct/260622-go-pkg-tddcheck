package filelayout

import (
	"go/ast"
	"go/token"
	"strings"
)

func servicePersistenceViolations(fileSet *token.FileSet, filename string, parsedFile *ast.File) []Violation {
	var violations []Violation
	for _, decl := range parsedFile.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			violations = append(violations, servicePersistenceGenDeclViolations(fileSet, filename, typed)...)
		case *ast.FuncDecl:
			violations = append(violations, servicePersistenceFuncViolations(fileSet, filename, typed)...)
		}
	}
	return violations
}

func servicePersistenceGenDeclViolations(fileSet *token.FileSet, filename string, decl *ast.GenDecl) []Violation {
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
				violations = append(violations, violationAt(fileSet, filename, spec.Pos(), "service files must not import persistence package "+path))
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
				if forbiddenServicePersistenceType(expr) {
					violations = append(violations, violationAt(fileSet, filename, expr.Pos(), "service files must not depend on persistence handle types"))
					return false
				}
				return true
			})
		}
	}
	return violations
}

func servicePersistenceFuncViolations(fileSet *token.FileSet, filename string, decl *ast.FuncDecl) []Violation {
	var violations []Violation
	if decl.Type != nil {
		ast.Inspect(decl.Type, func(node ast.Node) bool {
			expr, ok := node.(ast.Expr)
			if !ok {
				return true
			}
			if forbiddenServicePersistenceType(expr) {
				violations = append(violations, violationAt(fileSet, filename, expr.Pos(), "service files must not depend on persistence handle types"))
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
			if forbiddenServicePersistenceCall(typed.Fun) {
				violations = append(violations, violationAt(fileSet, filename, typed.Fun.Pos(), "service files must not call persistence APIs directly"))
				return false
			}
		case *ast.SelectorExpr:
			if forbiddenRepositoryModelSelector(typed) {
				violations = append(violations, violationAt(fileSet, filename, typed.Pos(), "service files must not reference repository schema models"))
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

func forbiddenServicePersistenceType(expr ast.Expr) bool {
	switch typed := expr.(type) {
	case *ast.StarExpr:
		return forbiddenServicePersistenceType(typed.X)
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
		return forbiddenServicePersistenceType(typed.Elt)
	case *ast.MapType:
		return forbiddenServicePersistenceType(typed.Key) || forbiddenServicePersistenceType(typed.Value)
	case *ast.ChanType:
		return forbiddenServicePersistenceType(typed.Value)
	}
	return false
}

func forbiddenServicePersistenceCall(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	name := selector.Sel.Name
	if forbiddenRepositoryModelSelector(selector) {
		return true
	}
	if selectorPackage(selector) == "sql" && oneOf(name, "Open", "OpenDB") {
		return true
	}
	if name == "RunInTx" {
		return selectorReceiverName(selector) == "db"
	}
	return oneOf(
		name,
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

func forbiddenRepositoryModelSelector(expr *ast.SelectorExpr) bool {
	return selectorPackage(expr) == "repository" && strings.HasSuffix(expr.Sel.Name, "Model")
}

func selectorReceiverName(expr *ast.SelectorExpr) string {
	ident, ok := expr.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}
