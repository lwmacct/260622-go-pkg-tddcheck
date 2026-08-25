package filelayout

import (
	"path/filepath"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func TestViolationsWarnOnCrossSubjectPrivateAccessWhenEnabled(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
		"internal/service/account.service.go": `package service
type AccountService struct{}
func NewAccountService() *AccountService { return &AccountService{} }
`,
		"internal/service/account.support.go": `package service
func utilAccount() string { return "account" }
`,
		"internal/service/device.support.go": `package service
func utilDevice() string { return utilAccount() }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{
		WarnSubjectPrivateAccess: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, `service subject "device" uses private declaration utilAccount`)
}

func TestViolationsWarnOnUnclassifiedFilesWhenEnabled(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/runtime/worker.go": "package runtime\nfunc Run() {}\n",
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{
		WarnUnclassifiedFiles: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "file is outside configured architecture layers")
}

func TestViolationsWarnOnOversizedSharedFileWhenEnabled(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/x.shared.support.go": `package service
type SharedOne struct {
	Value string
}
type SharedTwo struct {
	Value string
}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"), rulekit.Config{
		MaxSharedDeclarationLines: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "x.shared declarations occupy")
}
