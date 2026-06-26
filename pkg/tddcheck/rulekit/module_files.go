package rulekit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveRuleRoot(root string, ruleName string) (string, error) {
	if root == "" {
		return "", errors.New(ruleName + ".Root is empty")
	}
	if filepath.IsAbs(root) {
		return root, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	projectRoot, err := FindProjectRoot(wd)
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, root), nil
}

func ShouldSkipModuleScanDir(name string, config Config) bool {
	config = config.WithDefaults()
	if strings.HasPrefix(name, ".") {
		return true
	}
	return StringIn(name, config.SkipDirs)
}

func DisplayFilename(filename string) string {
	projectRoot, err := FindProjectRoot(filepath.Dir(filename))
	if err == nil {
		if relative, relErr := filepath.Rel(projectRoot, filename); relErr == nil && !strings.HasPrefix(relative, "..") {
			return filepath.ToSlash(relative)
		}
	}
	return filepath.ToSlash(filename)
}

func FindProjectRoot(start string) (string, error) {
	wd, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", fmt.Errorf("go.mod not found from %s", start)
		}
		wd = parent
	}
}

func ModulePathForRoot(root string) (string, error) {
	projectRoot, err := FindProjectRoot(root)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return "", err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), "module ")
		if ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), nil
		}
	}
	return "", errors.New("module path not found in go.mod")
}
