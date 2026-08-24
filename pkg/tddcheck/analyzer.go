package tddcheck

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rules/filelayout"
	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rules/layerdeps"
)

const (
	DefaultDocFile        = "docs/tddcheck.index.gen.md"
	AnalysisSchemaVersion = "1"
)

type Options struct {
	// Root is an absolute path or a path relative to the current Go module.
	// An empty value selects internal.
	Root string
	// Config controls package loading and the active architecture profile.
	Config Config
}

type Analyzer struct {
	root   string
	config Config
	engine rulekit.Engine
}

// New constructs an analyzer with the built-in file-layout and dependency
// rules. Registrars may add project-specific typed rules to the same engine.
func New(options Options, registrars ...func(*Engine)) (*Analyzer, error) {
	config, err := options.Config.Compile()
	if err != nil {
		return nil, err
	}
	if err := config.ValidateFileLayout(); err != nil {
		return nil, err
	}
	var engine rulekit.Engine
	filelayout.Register(&engine)
	layerdeps.Register(&engine)
	for _, register := range registrars {
		if register != nil {
			register(&engine)
		}
	}
	return &Analyzer{root: options.Root, config: config, engine: engine}, nil
}

// Analysis is the stable result of one package load and one engine run.
type Analysis struct {
	SchemaVersion string               `json:"schemaVersion"`
	Root          string               `json:"root"`
	ModulePath    string               `json:"modulePath"`
	Index         Index                `json:"index"`
	Diagnostics   []rulekit.Diagnostic `json:"diagnostics,omitempty"`
	LoadErrors    []LoadError          `json:"packageErrors,omitempty"`

	Duration    time.Duration `json:"-"`
	projectRoot string
}

func (a *Analyzer) Analyze(ctx context.Context) (Analysis, error) {
	if a == nil {
		return Analysis{}, fmt.Errorf("tddcheck analyzer is nil")
	}
	start := time.Now()
	snapshot, err := rulekit.Load(ctx, a.root, a.config)
	if err != nil {
		return Analysis{SchemaVersion: AnalysisSchemaVersion, Duration: time.Since(start)}, err
	}
	diagnostics, err := a.engine.Run(ctx, snapshot)
	if err != nil {
		return Analysis{SchemaVersion: AnalysisSchemaVersion, Duration: time.Since(start)}, err
	}
	index := indexFromSnapshot(snapshot)
	return Analysis{
		SchemaVersion: AnalysisSchemaVersion,
		Root:          index.Root,
		ModulePath:    index.ModulePath,
		Index:         index,
		Diagnostics:   diagnostics,
		LoadErrors:    snapshot.LoadErrors,
		Duration:      time.Since(start),
		projectRoot:   snapshot.ProjectRoot,
	}, nil
}

// Assert is the testing adapter for the core context-aware analyzer API.
func Assert(tb testing.TB, analyzer *Analyzer) {
	tb.Helper()
	analysis, err := analyzer.Analyze(context.Background())
	if err != nil {
		tb.Fatal(err)
	}
	if !analysis.Passed() {
		tb.Fatal(analysis.Text())
	}
}

func (a Analysis) WriteMarkdown(outputFile string) error {
	if outputFile == "" {
		outputFile = DefaultDocFile
	}
	outputPath := outputFile
	if !filepath.IsAbs(outputPath) {
		if a.projectRoot == "" {
			return fmt.Errorf("relative output requires an analysis produced by Analyzer.Analyze")
		}
		outputPath = filepath.Join(a.projectRoot, filepath.FromSlash(outputFile))
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(a.Markdown()), 0o644)
}
