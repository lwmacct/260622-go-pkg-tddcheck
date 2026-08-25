package invariants

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func TestCheckAcceptsConfiguredCompositeUniqueGroup(t *testing.T) {
	root := fixture(t, `package repository

import "github.com/uptrace/bun"

type ResourceLedgerConsumptionModel struct {
	bun.BaseModel `+"`bun:\"table:resource_consumptions\"`"+`
	UserID string `+"`bun:\"user_id,notnull,unique:resource_consumption_user_key\"`"+`
	IdempotencyKey string `+"`bun:\"idempotency_key,notnull,unique:resource_consumption_user_key\"`"+`
}
`)

	violations, err := Check(context.Background(), filepath.Join(root, "internal"), rulekit.Config{
		SchemaInvariants: []rulekit.SchemaInvariant{{
			Subject: "resource_ledger",
			Table:   "resource_consumptions",
			Unique:  [][]string{{"user_id", "idempotency_key"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestCheckRejectsGlobalUniqueWhenCompositeIsRequired(t *testing.T) {
	root := fixture(t, `package repository

import "github.com/uptrace/bun"

type ResourceLedgerConsumptionModel struct {
	bun.BaseModel `+"`bun:\"table:resource_consumptions\"`"+`
	UserID string `+"`bun:\"user_id,notnull\"`"+`
	IdempotencyKey string `+"`bun:\"idempotency_key,notnull,unique\"`"+`
}
`)

	violations, err := Check(context.Background(), filepath.Join(root, "internal"), rulekit.Config{
		SchemaInvariants: []rulekit.SchemaInvariant{{
			Subject: "resource_ledger",
			Table:   "resource_consumptions",
			Unique:  [][]string{{"user_id", "idempotency_key"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "global unique") {
		t.Fatalf("expected composite constraint violation, got %#v", violations)
	}
}

func TestCheckReportsMissingSchemaModel(t *testing.T) {
	root := fixture(t, `package repository
type Store struct{}
`)
	violations, err := Check(context.Background(), filepath.Join(root, "internal"), rulekit.Config{
		SchemaInvariants: []rulekit.SchemaInvariant{{Subject: "resource_trial", Table: "resource_trial_claims", Unique: [][]string{{"user_id", "claim_date"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "no matching repository schema model") {
		t.Fatalf("expected missing model violation, got %#v", violations)
	}
}

func TestConfigRejectsInvalidSchemaInvariant(t *testing.T) {
	_, err := (rulekit.Config{SchemaInvariants: []rulekit.SchemaInvariant{{
		Subject: "resource-ledger",
		Table:   "resource_consumptions",
		Unique:  [][]string{{"user_id", "user_id"}},
	}}}).Compile()
	if err == nil {
		t.Fatal("expected invalid schema invariant to fail")
	}
}

func fixture(t *testing.T, source string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "internal", "repository", "resource_ledger.schema.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\ngo 1.27.0\n\nrequire github.com/uptrace/bun v1.2.18\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}
