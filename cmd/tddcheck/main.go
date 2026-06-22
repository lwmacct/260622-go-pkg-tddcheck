package main

import (
	"errors"
	"flag"
	"os"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	var root string
	var showVersion bool

	flags := flag.NewFlagSet("tddcheck", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&root, "root", "internal", "project root or module subtree to check")
	flags.BoolVar(&showVersion, "version", false, "print version")

	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if showVersion {
		_, _ = os.Stdout.WriteString(version + "\n")
		return 0
	}

	result := (tddcheck.ProjectRules{Root: root}).Check()
	if result.Err != nil {
		_, _ = os.Stderr.WriteString("tddcheck: " + result.Err.Error() + "\n")
		return 2
	}

	_, _ = os.Stdout.WriteString(result.Text() + "\n")
	if !result.Passed {
		return 1
	}
	return 0
}
