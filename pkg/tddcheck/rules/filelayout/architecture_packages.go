package filelayout

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

// packageBoundaryViolations enforces the optional layer package contract.
// The default profile uses it for the three top-level packages, while custom
// layers remain compatible unless LayerPackageNames explicitly opts them in.
func packageBoundaryViolations(snapshot *rulekit.Snapshot) []Violation {
	type packageFiles struct {
		name  string
		dir   string
		files []rulekit.GoFile
	}

	byLayer := make(map[string]map[string]*packageFiles)
	for _, file := range snapshot.Files {
		if file.IsTest || file.Layer == "" {
			continue
		}
		expected, strict := snapshot.Profile.PackageName(file.Layer)
		if !strict || expected == "" {
			continue
		}
		key := file.PackageID
		if key == "" {
			key = file.PackagePath + "\x00" + file.Dir
		}
		items := byLayer[file.Layer]
		if items == nil {
			items = make(map[string]*packageFiles)
			byLayer[file.Layer] = items
		}
		item := items[key]
		if item == nil {
			item = &packageFiles{name: packageName(snapshot, file.PackagePath), dir: filepath.Dir(file.AbsPath)}
			items[key] = item
		}
		item.files = append(item.files, file)
	}

	var violations []Violation
	for layer, packages := range byLayer {
		expected, _ := snapshot.Profile.PackageName(layer)
		expectedDir := filepath.Join(snapshot.Root, layer)
		for _, item := range packages {
			if len(item.files) == 0 {
				continue
			}
			sort.Slice(item.files, func(i, j int) bool {
				return item.files[i].AbsPath < item.files[j].AbsPath
			})
			file := item.files[0]
			if filepath.Clean(item.dir) != filepath.Clean(expectedDir) {
				violations = append(violations, Violation{
					File:    rulekit.DisplayFilename(file.AbsPath),
					Line:    1,
					Code:    RuleID + "/package-boundary",
					Message: fmt.Sprintf("%s layer must be a direct package at %s; found package directory %s", layer, rulekit.DisplayFilename(expectedDir), rulekit.DisplayFilename(item.dir)),
				})
			}
			if item.name != "" && item.name != expected {
				violations = append(violations, Violation{
					File:    rulekit.DisplayFilename(file.AbsPath),
					Line:    1,
					Code:    RuleID + "/package-name",
					Message: fmt.Sprintf("%s layer package must be named %q, found %q", layer, expected, item.name),
				})
			}
		}
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].File != violations[j].File {
			return violations[i].File < violations[j].File
		}
		return strings.Compare(violations[i].Message, violations[j].Message) < 0
	})
	return violations
}

func packageName(snapshot *rulekit.Snapshot, path string) string {
	if path == "" {
		return ""
	}
	pkg, ok := snapshot.Package(path)
	if !ok {
		return ""
	}
	return pkg.Name
}
