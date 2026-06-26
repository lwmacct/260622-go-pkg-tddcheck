package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	return runWithArgs(os.Args[1:], os.Stdout, os.Stderr)
}

func runWithArgs(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		return runCheck(nil, stdout, stderr)
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "map":
		return runMap(args[1:], stdout, stderr)
	case "doc":
		return runDoc(args[1:], stdout, stderr)
	case "version":
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	case "help", "--help":
		writeUsage(stderr)
		return 0
	default:
		if strings.HasPrefix(args[0], "-") {
			return runCheck(args, stdout, stderr)
		}
		_, _ = fmt.Fprintln(stderr, "tddcheck: unknown command "+args[0])
		writeUsage(stderr)
		return 2
	}
}

func runCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("check", stderr)
	root := flags.String("root", "internal", "project root or module subtree to check")
	if code := parseFlags(flags, args, stderr); code != 0 {
		return code
	}
	analysis, err := (tddcheck.Project{Root: *root}).Analyze()
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

func runMap(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("map", stderr)
	root := flags.String("root", "internal", "project root or module subtree to check")
	format := flags.String("format", "text", "output format: text or json")
	if code := parseFlags(flags, args, stderr); code != 0 {
		return code
	}
	analysis, err := (tddcheck.Project{Root: *root}).Analyze()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
		return 2
	}
	switch *format {
	case "text":
		_, _ = fmt.Fprintln(stdout, analysis.ProjectMap().Text())
	case "json":
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(analysis); err != nil {
			_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
			return 2
		}
	default:
		_, _ = fmt.Fprintln(stderr, "tddcheck: unsupported map format "+*format)
		return 2
	}
	return 0
}

func runDoc(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("doc", stderr)
	root := flags.String("root", "internal", "project root or module subtree to check")
	output := flags.String("output", tddcheck.DefaultDocFile, "markdown output file")
	if code := parseFlags(flags, args, stderr); code != 0 {
		return code
	}
	analysis, err := (tddcheck.Project{Root: *root}).Analyze()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
		return 2
	}
	if err := analysis.WriteMarkdown(*output); err != nil {
		_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
		return 2
	}
	_, _ = fmt.Fprintln(stdout, "tddcheck: wrote "+*output)
	return 0
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
	_, _ = output.Write([]byte("  map      print project map\n"))
	_, _ = output.Write([]byte("  doc      write project map markdown documentation\n"))
	_, _ = output.Write([]byte("  version  print version\n"))
}
