package rulekit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ShouldSkipModuleScanDir(name string, config Config) bool {
	if config.SkipDirs == nil {
		config.SkipDirs = DefaultConfig().SkipDirs
	}
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
