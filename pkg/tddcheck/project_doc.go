package tddcheck

import (
	"os"
	"path/filepath"
	"testing"
)

const DefaultProjectMapDocFile = "docs/tddcheck.gen.md"

type ProjectMapDoc struct {
	Root       string
	OutputFile string
	Config     Config
}

func (d ProjectMapDoc) Write(tb testing.TB) {
	tb.Helper()

	root := d.Root
	if root == "" {
		root = "internal"
	}
	outputFile := d.OutputFile
	if outputFile == "" {
		outputFile = DefaultProjectMapDocFile
	}

	projectMap, err := (ProjectRules{Root: root, Config: d.Config}).Map()
	if err != nil {
		tb.Fatal(err)
	}

	outputPath := outputFile
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(projectMap.projectRoot, filepath.FromSlash(outputFile))
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte(projectMap.Markdown()), 0o600); err != nil {
		tb.Fatal(err)
	}
	tb.Logf("wrote tddcheck project map: %s", outputPath)
}
