package tddcheck_test

import (
	"context"
	"fmt"
	"log"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck"
)

func ExampleAnalyzer_Analyze() {
	analyzer, err := tddcheck.New(tddcheck.Options{Root: "internal"})
	if err != nil {
		log.Fatal(err)
	}
	analysis, err := analyzer.Analyze(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(analysis.Text())
}

func ExampleDefaultConfig() {
	config := tddcheck.DefaultConfig()
	config.DependencyLayerDirs = append(config.DependencyLayerDirs, "runtime")
	config.LayerRules = append(config.LayerRules, tddcheck.LayerDependencyRule{
		SourceLayer: "runtime",
		TargetLayer: "service",
		Message:     "runtime must not import service",
	})

	analyzer, err := tddcheck.New(tddcheck.Options{
		Root:   "internal",
		Config: config,
	})
	if err != nil {
		log.Fatal(err)
	}
	analysis, err := analyzer.Analyze(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(analysis.Text())
}
