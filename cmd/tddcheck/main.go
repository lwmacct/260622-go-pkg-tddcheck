package main

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx))
}

func run(ctx context.Context) int {
	return runWithArgs(ctx, os.Args[1:], os.Stdout, os.Stderr)
}

func runWithArgs(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		return runCheck(ctx, nil, stdout, stderr)
	}
	switch args[0] {
	case "check":
		return runCheck(ctx, args[1:], stdout, stderr)
	case "index":
		return runIndex(ctx, args[1:], stdout, stderr)
	case "doc":
		return runDoc(ctx, args[1:], stdout, stderr)
	case "version":
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	case "help", "--help":
		writeUsage(stderr)
		return 0
	default:
		if strings.HasPrefix(args[0], "-") {
			return runCheck(ctx, args, stdout, stderr)
		}
		_, _ = fmt.Fprintln(stderr, "tddcheck: unknown command "+args[0])
		writeUsage(stderr)
		return 2
	}
}

func runCheck(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("check", stderr)
	options := newAnalysisFlags(flags)
	if code := parseFlags(flags, args, stderr); code != 0 {
		return code
	}
	analysis, err := analyze(ctx, options)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
		return 2
	}
	_, _ = fmt.Fprintln(stdout, analysis.Text())
	if !analysis.Passed() {
		return 1
	}
	return 0
}

func runIndex(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("index", stderr)
	options := newAnalysisFlags(flags)
	format := flags.String("format", "text", "output format: text or json")
	if code := parseFlags(flags, args, stderr); code != 0 {
		return code
	}
	analysis, err := analyze(ctx, options)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
		return 2
	}
	switch *format {
	case "text":
		_, _ = fmt.Fprintln(stdout, analysis.ArchitectureIndex().Text())
	case "json":
		if err := json.MarshalWrite(stdout, analysis, jsontext.WithIndent("  ")); err != nil {
			_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
			return 2
		}
		_, _ = fmt.Fprintln(stdout)
	default:
		_, _ = fmt.Fprintln(stderr, "tddcheck: unsupported index format "+*format)
		return 2
	}
	return 0
}

func runDoc(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("doc", stderr)
	options := newAnalysisFlags(flags)
	output := flags.String("output", tddcheck.DefaultDocFile, "markdown output file")
	check := flags.Bool("check", false, "verify that markdown output is up to date")
	if code := parseFlags(flags, args, stderr); code != 0 {
		return code
	}
	analysis, err := analyze(ctx, options)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
		return 2
	}
	if *check {
		if err := analysis.CheckMarkdown(*output); err != nil {
			_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
			if errors.Is(err, tddcheck.ErrMarkdownOutOfDate) {
				return 1
			}
			return 2
		}
		_, _ = fmt.Fprintln(stdout, "tddcheck: documentation is up to date")
		return 0
	}
	if err := analysis.WriteMarkdown(*output); err != nil {
		_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
		return 2
	}
	_, _ = fmt.Fprintln(stdout, "tddcheck: wrote "+*output)
	return 0
}

type analysisFlags struct {
	root       *string
	configFile *string
}

func newAnalysisFlags(flags *flag.FlagSet) analysisFlags {
	return analysisFlags{
		root:       flags.String("root", "internal", "project root or module subtree to check"),
		configFile: flags.String("config", "", "JSON configuration file"),
	}
}

func analyze(ctx context.Context, options analysisFlags) (tddcheck.Analysis, error) {
	config, err := readConfig(*options.configFile)
	if err != nil {
		return tddcheck.Analysis{}, err
	}
	analyzer, err := tddcheck.New(tddcheck.Options{Root: *options.root, Config: config})
	if err != nil {
		return tddcheck.Analysis{}, err
	}
	return analyzer.Analyze(ctx)
}

func readConfig(filename string) (tddcheck.Config, error) {
	if filename == "" {
		return tddcheck.Config{}, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return tddcheck.Config{}, err
	}
	defer file.Close()
	var config tddcheck.Config
	if err := json.UnmarshalRead(file, &config, json.RejectUnknownMembers(true)); err != nil {
		return tddcheck.Config{}, fmt.Errorf("decode config %s: %w", filename, err)
	}
	return config, nil
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet("tddcheck "+name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	return flags
}

func parseFlags(flags *flag.FlagSet, args []string, stderr io.Writer) int {
	if err := validateLongFlags(args); err != nil {
		_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
		return 2
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		_, _ = fmt.Fprintln(stderr, "tddcheck: unexpected argument "+flags.Arg(0))
		return 2
	}
	return 0
}

func validateLongFlags(args []string) error {
	for _, arg := range args {
		if arg == "--" {
			return nil
		}
		if strings.HasPrefix(arg, "--") {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return fmt.Errorf("short flags are not supported: %s", arg)
		}
	}
	return nil
}

func writeUsage(output io.Writer) {
	_, _ = output.Write([]byte("Usage: tddcheck <command> [options]\n\n"))
	_, _ = output.Write([]byte("Commands:\n"))
	_, _ = output.Write([]byte("  check    run architecture checks\n"))
	_, _ = output.Write([]byte("  index    print architecture index\n"))
	_, _ = output.Write([]byte("  doc      write architecture index markdown documentation\n"))
	_, _ = output.Write([]byte("  version  print version\n"))
}
