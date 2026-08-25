package filelayout

import (
	"path/filepath"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func TestViolationsChecksUtilsContent(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/handler/device.utils.go": `package handler

const defaultName = "device"
type DeviceHelper struct{}
func (h DeviceHelper) utilDevice() {}
func buildDevice() {}
`,
	})

	config := rulekit.DefaultConfig()
	config.LayerKindPolicies["handler"]["utils"] = "utils"
	violations, err := checkRoot(filepath.Join(root, "internal"), config)
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "utils files must only declare util* functions")
	assertViolationContains(t, violations, "utils functions must not use receivers")
	assertViolationContains(t, violations, "utils function buildDevice must start with util")
}
