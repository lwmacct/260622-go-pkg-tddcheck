package tddcheck

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rules/filelayout"
	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rules/layerdeps"
)

const DefaultDocFile = "docs/tddcheck.index.gen.md"

type Project struct {
	Root   string
	Config Config
}

type Analysis struct {
	Root       string        `json:"root"`
	ModulePath string        `json:"modulePath"`
	Index      Index         `json:"index"`
	Violations []Violation   `json:"violations,omitempty"`
	Duration   time.Duration `json:"duration"`

	projectRoot string
}

func (p Project) Analyze() (Analysis, error) {
	start := time.Now()
	context, err := p.context()
	if err != nil {
		return Analysis{Duration: time.Since(start)}, err
	}
	violations, err := checkContext(context)
	if err != nil {
		return Analysis{Duration: time.Since(start)}, err
	}
	index := indexFromContext(context)
	return Analysis{
		Root:        index.Root,
		ModulePath:  index.ModulePath,
		Index:       index,
		Violations:  violations,
		Duration:    time.Since(start),
		projectRoot: index.projectRoot,
	}, nil
}

func (p Project) Assert(tb testing.TB) {
	tb.Helper()

	analysis, err := p.Analyze()
	if err != nil {
		tb.Fatal(err)
	}
	if !analysis.Passed() {
		tb.Fatal(analysis.Text())
	}
}

func (p Project) WriteDoc(tb testing.TB, outputFile string) {
	tb.Helper()

	analysis, err := p.Analyze()
	if err != nil {
		tb.Fatal(err)
	}
	if outputFile == "" {
		outputFile = DefaultDocFile
	}
	outputPath := outputFile
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(analysis.projectRoot, filepath.FromSlash(outputFile))
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte(analysis.Markdown()), 0o600); err != nil {
		tb.Fatal(err)
	}
	tb.Logf("wrote tddcheck project doc: %s", outputPath)
}

func (a Analysis) WriteMarkdown(outputFile string) error {
	if outputFile == "" {
		outputFile = DefaultDocFile
	}
	outputPath := outputFile
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(a.projectRoot, filepath.FromSlash(outputFile))
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(a.Markdown()), 0o600)
}

func (p Project) context() (*rulekit.Context, error) {
	root := p.Root
	if root == "" {
		root = "internal"
	}
	return rulekit.NewContext(root, "project", p.Config)
}

func checkContext(context *rulekit.Context) ([]Violation, error) {
	var violations []Violation
	for _, rule := range defaultRules() {
		values, err := rule.Check(context)
		if err != nil {
			return violations, err
		}
		for _, value := range values {
			violations = append(violations, Violation{
				Rule:    value.Rule,
				File:    value.File,
				Line:    value.Line,
				Message: value.Message,
			})
		}
	}
	slices.SortFunc(violations, func(a, b Violation) int {
		return strings.Compare(a.String(), b.String())
	})
	return violations, nil
}

func defaultRules() []rulekit.Rule {
	return []rulekit.Rule{
		filelayout.New(""),
		layerdeps.New(""),
	}
}
