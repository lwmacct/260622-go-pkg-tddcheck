package tddcheck

import (
	"iter"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

type Engine = rulekit.Engine
type Snapshot = rulekit.Snapshot
type GoFile = rulekit.GoFile
type GoPackage = rulekit.GoPackage

func FileScope(snapshot *Snapshot) iter.Seq[GoFile] {
	return rulekit.FileScope(snapshot)
}

func PackageScope(snapshot *Snapshot) iter.Seq[GoPackage] {
	return rulekit.PackageScope(snapshot)
}

func SnapshotScope(snapshot *Snapshot) iter.Seq[*Snapshot] {
	return rulekit.SnapshotScope(snapshot)
}
