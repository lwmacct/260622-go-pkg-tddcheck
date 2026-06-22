package tddcheck

import "github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"

type Config = rulekit.Config

type LayerDependencyRule = rulekit.LayerDependencyRule

func DefaultConfig() Config {
	return rulekit.DefaultConfig()
}
