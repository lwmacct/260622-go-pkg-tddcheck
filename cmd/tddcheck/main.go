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
	var root string
	var showMap bool
	var format string
	var showVersion bool

	flags := flag.NewFlagSet("tddcheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&root, "root", "internal", "project root or module subtree to check")
	flags.BoolVar(&showMap, "map", false, "print project map")
	flags.StringVar(&format, "format", "text", "project map output format (text or json)")
	flags.BoolVar(&showVersion, "version", false, "print version")
	flags.Usage = func() {
		_, _ = stderr.Write([]byte("Usage: tddcheck [options]\n\nOptions:\n"))
		_, _ = stderr.Write([]byte("  --root string\n"))
		_, _ = stderr.Write([]byte("        project root or module subtree to check (default \"internal\")\n"))
		_, _ = stderr.Write([]byte("  --map\n"))
		_, _ = stderr.Write([]byte("        print project map instead of running checks\n"))
		_, _ = stderr.Write([]byte("  --format string\n"))
		_, _ = stderr.Write([]byte("        project map output format: text or json (default \"text\")\n"))
		_, _ = stderr.Write([]byte("  --version\n"))
		_, _ = stderr.Write([]byte("        print version\n"))
	}

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
	if showVersion {
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	}
	formatSet := false
	flags.Visit(func(flag *flag.Flag) {
		if flag.Name == "format" {
			formatSet = true
		}
	})
	if formatSet && !showMap {
		_, _ = fmt.Fprintln(stderr, "tddcheck: --format requires --map")
		return 2
	}
	if showMap {
		return runMap(root, format, stdout, stderr)
	}

	result := (tddcheck.ProjectRules{Root: root}).Check()
	if result.Err != nil {
		_, _ = fmt.Fprintln(stderr, "tddcheck: "+result.Err.Error())
		return 2
	}

	_, _ = fmt.Fprintln(stdout, result.Text())
	if !result.Passed {
		return 1
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

func runMap(root string, format string, stdout io.Writer, stderr io.Writer) int {
	projectMap, err := (tddcheck.ProjectRules{Root: root}).Map()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
		return 2
	}
	switch format {
	case "text":
		_, _ = fmt.Fprintln(stdout, projectMap.Text())
	case "json":
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(projectMap); err != nil {
			_, _ = fmt.Fprintln(stderr, "tddcheck: "+err.Error())
			return 2
		}
	default:
		_, _ = fmt.Fprintln(stderr, "tddcheck: unsupported map format "+format)
		return 2
	}
	return 0
}
