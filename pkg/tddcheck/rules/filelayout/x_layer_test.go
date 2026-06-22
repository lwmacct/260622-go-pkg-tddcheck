package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsRejectsArchitectureScopeInWrongLayer(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/x_database.service.go": `package service
func NewDatabaseService() {}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `architecture scope "x_database" must not use x_database.service.go in service`)
}
