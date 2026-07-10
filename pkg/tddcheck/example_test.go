package tddcheck_test

import (
	"fmt"
	"log"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck"
)

func ExampleProject_Analyze() {
	analysis, err := (tddcheck.Project{Root: "internal"}).Analyze()
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

	analysis, err := (tddcheck.Project{
		Root:   "internal",
		Config: config,
	}).Analyze()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(analysis.Text())
}
