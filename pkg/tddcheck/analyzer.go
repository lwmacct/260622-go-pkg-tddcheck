package tddcheck

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

const (
	DefaultDocFile        = "docs/tddcheck.index.gen.md"
	AnalysisSchemaVersion = "2"
)

// ErrMarkdownOutOfDate identifies missing or stale generated documentation.
var ErrMarkdownOutOfDate = errors.New("generated markdown is out of date")

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
	if err := filelayout.ValidateProfile(config.Profile()); err != nil {
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
	FreeFiles     []FreeFile           `json:"freeFiles,omitempty"`
	Diagnostics   []rulekit.Diagnostic `json:"diagnostics,omitempty"`
	LoadErrors    []LoadError          `json:"packageErrors,omitempty"`

	Duration    time.Duration `json:"-"`
	projectRoot string
}

// FreeFile identifies a source file whose declaration policy is unrestricted.
type FreeFile struct {
	Identity FileIdentity `json:"identity"`
	File     string       `json:"file"`
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
		FreeFiles:     freeFilesFromSnapshot(snapshot),
		Diagnostics:   diagnostics,
		LoadErrors:    snapshot.LoadErrors,
		Duration:      time.Since(start),
		projectRoot:   snapshot.ProjectRoot,
	}, nil
}

// TestOptions controls the artifacts checked or updated by [Assert].
type TestOptions struct {
	// Markdown enables generated architecture documentation checks when non-nil.
	Markdown *MarkdownTestOptions
}

// MarkdownTestOptions controls generated Markdown handling in [Assert].
type MarkdownTestOptions struct {
	// OutputFile is absolute or relative to the analyzed module. An empty value
	// selects [DefaultDocFile].
	OutputFile string
	// Update writes the current document instead of checking for drift.
	Update bool
}

// Assert analyzes once, fails the test on rule violations, and then checks or
// updates configured artifacts.
func Assert(tb testing.TB, analyzer *Analyzer, options TestOptions) {
	tb.Helper()
	analysis, err := analyzer.Analyze(tb.Context())
	if err != nil {
		tb.Fatal(err)
	}
	if !analysis.Passed() {
		tb.Fatal(analysis.Text())
	}
	if options.Markdown == nil {
		return
	}
	if options.Markdown.Update {
		if err := analysis.WriteMarkdown(options.Markdown.OutputFile); err != nil {
			tb.Fatal(err)
		}
		outputFile := options.Markdown.OutputFile
		if outputFile == "" {
			outputFile = DefaultDocFile
		}
		tb.Logf("tddcheck: wrote %s", outputFile)
		return
	}
	if err := analysis.CheckMarkdown(options.Markdown.OutputFile); err != nil {
		tb.Fatal(err)
	}
}

// WriteMarkdown writes the current generated architecture documentation.
func (a Analysis) WriteMarkdown(outputFile string) error {
	outputPath, err := a.markdownPath(outputFile)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return err
	}
	return writeFileAtomic(outputPath, []byte(a.Markdown()), 0o644)
}

// CheckMarkdown reports whether outputFile exactly matches the current
// generated architecture documentation without modifying it.
func (a Analysis) CheckMarkdown(outputFile string) error {
	outputPath, err := a.markdownPath(outputFile)
	if err != nil {
		return err
	}
	actual, err := os.ReadFile(outputPath)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s does not exist", ErrMarkdownOutOfDate, outputPath)
	}
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, []byte(a.Markdown())) {
		return fmt.Errorf("%w: %s differs from current analysis", ErrMarkdownOutOfDate, outputPath)
	}
	return nil
}

func (a Analysis) markdownPath(outputFile string) (string, error) {
	if outputFile == "" {
		outputFile = DefaultDocFile
	}
	if filepath.IsAbs(outputFile) {
		return outputFile, nil
	}
	if a.projectRoot == "" {
		return "", fmt.Errorf("relative output requires an analysis produced by Analyzer.Analyze")
	}
	return filepath.Join(a.projectRoot, filepath.FromSlash(outputFile)), nil
}

func writeFileAtomic(filename string, data []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(filename), "."+filepath.Base(filename)+".*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryName)
	}()
	if err := temporary.Chmod(mode); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, filename)
}

func freeFilesFromSnapshot(snapshot *rulekit.Snapshot) []FreeFile {
	var result []FreeFile
	for _, file := range snapshot.Files {
		if file.IsTest || !file.IdentityOK || file.Identity.Kind != "free" {
			continue
		}
		result = append(result, FreeFile{
			Identity: file.Identity,
			File:     snapshot.DisplayPath(file.AbsPath),
		})
	}
	slices.SortFunc(result, func(a, b FreeFile) int {
		return strings.Compare(a.File, b.File)
	})
	return result
}
