// Package tddcheck analyzes and documents the architecture of Go projects.
//
// Analyzer loads the selected module subtree through the Go build system,
// giving every rule a shared syntax tree, package graph, and go/types view.
// Build constraints are honored. Package and type errors are retained for
// best-effort analysis unless Config.StrictPackages is enabled.
//
// # Quick start
//
// Tests can enforce the default handler/service/repository profile and check
// generated architecture documentation with:
//
//	func TestArchitecture(t *testing.T) {
//		analyzer, err := tddcheck.New(tddcheck.Options{Root: "internal"})
//		if err != nil {
//			t.Fatal(err)
//		}
//		tddcheck.Assert(t, analyzer, tddcheck.TestOptions{
//			Markdown: &tddcheck.MarkdownTestOptions{},
//		})
//	}
//
// Programs should call [Analyzer.Analyze] with their own context. Operational
// failures are returned as errors; rule failures are stored in
// [Analysis.Diagnostics] and queried with [Analysis.Passed].
//
// # Configuration
//
// The zero [Config] uses [DefaultConfig]. Nil collections inherit defaults;
// non-nil empty collections disable optional entries, while required per-layer
// maps must still cover every configured layout layer. Configuration is
// validated and deep copied by [New], so callers may safely reuse or mutate
// their input afterward. IncludeTests loads test variants, BuildFlags
// configures the Go build, and StrictPackages rejects incomplete package
// graphs.
//
// # Extensibility
//
// Registrars passed to [New] may add rules through [Engine.Register]. Go 1.27
// generic methods keep file, package, and snapshot rules strongly typed; use
// [FileScope], [PackageScope], or [SnapshotScope] as appropriate.
//
// # Index and output
//
// The same analysis-local Snapshot drives diagnostics and the architecture [Index].
// [Analysis.Markdown] renders generated documentation,
// [Analysis.WriteMarkdown] writes it relative to the analyzed module, and
// [Analysis.CheckMarkdown] detects committed-document drift without writing.
// The CLI emits schema-versioned JSON with encoding/json/v2.
package tddcheck
