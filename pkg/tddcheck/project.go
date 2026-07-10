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

// DefaultDocFile is the module-relative output path used when no Markdown
// output filename is provided.
const DefaultDocFile = "docs/tddcheck.index.gen.md"

// Project identifies the module subtree to analyze and the rules to apply.
type Project struct {
	// Root is an absolute path or a path relative to the current Go module. An
	// empty Root selects "internal".
	Root string
	// Config controls scanning and architecture rules. Its zero value uses the
	// defaults returned by [DefaultConfig].
	Config Config
}

// Analysis contains the architecture checks and index produced by one project
// scan.
type Analysis struct {
	// Root is the analyzed root displayed relative to the module when possible.
	Root string `json:"root"`
	// ModulePath is the module directive read from go.mod.
	ModulePath string `json:"modulePath"`
	// Index is the structured architecture index.
	Index Index `json:"index"`
	// Violations contains all failed architecture rules in stable text order.
	Violations []Violation `json:"violations,omitempty"`
	// Duration is the elapsed analysis time.
	Duration time.Duration `json:"duration"`

	projectRoot string
}

// Analyze scans the project once and returns its rule violations and
// architecture index. Errors describe failures to resolve, read, or parse the
// project; architecture rule failures are returned in [Analysis.Violations].
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

// Assert analyzes the project and fails tb on an analysis error or any rule
// violation.
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

// WriteDoc analyzes the project and writes its Markdown architecture index.
// It fails tb on analysis or filesystem errors. An empty outputFile uses
// [DefaultDocFile], and relative paths are resolved from the analyzed module.
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

// WriteMarkdown writes the analysis as a Markdown architecture index. An empty
// outputFile uses [DefaultDocFile]. For an Analysis returned by
// [Project.Analyze], relative paths are resolved from the analyzed module.
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
