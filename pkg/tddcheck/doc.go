// Package tddcheck checks and documents the architecture of a Go project.
//
// It statically scans a Go module subtree, applies file-layout and
// layer-dependency rules, and builds an architecture index from the same parsed
// source files. The index describes discovered handlers, services, stores,
// database tables, and projections.
//
// # Quick start
//
// [Project] is the main entry point. Most projects can enforce the default
// architecture in a test with [Project.Assert]:
//
//	func TestArchitecture(t *testing.T) {
//		tddcheck.Project{Root: "internal"}.Assert(t)
//	}
//
// Assert reports analysis errors and violations through testing.TB. Programs
// that need to inspect or format the result should use [Project.Analyze].
// Analyze returns operational failures as an error; rule failures are recorded
// in [Analysis.Violations] and can be queried with [Analysis.Passed].
//
// # Project roots and scanning
//
// An empty [Project.Root] selects "internal". An absolute root is used as-is. A
// relative root is resolved from the Go module containing the current working
// directory. The selected root must belong to a module with a readable go.mod
// and module directive.
//
// By default, tddcheck validates handler, service, and repository directories.
// It checks their file layout and import direction. Files ending in _test.go,
// hidden directories, and configured skip directories are not scanned. A file
// named x_free.go is scanned but excluded from checks and the architecture
// index.
//
// # Configuration
//
// The zero [Config] has the same behavior as [DefaultConfig]. Defaults are
// applied independently to each slice or map field: a nil field inherits its
// default, while a non-nil empty field explicitly disables that default.
//
// Config can change layer directories, filename modes, allowed file kinds and
// architecture scopes, skipped directories, and dependency rules. Start with
// DefaultConfig when changing only a few values; construct Config directly
// when the project defines a substantially different architecture.
//
// # Architecture index
//
// [Analysis.ProjectIndex] returns the structured [Index]. [Index.Text] renders
// a human-readable summary. Analysis and its nested index types also expose
// JSON tags for machine-readable output.
//
// Indexing is independent of whether the checks pass. It is based on recognized
// source declarations and call patterns; it does not execute application code
// or connect to a database. API endpoints are not part of the architecture
// index; use the project's API or OpenAPI tooling for that contract.
//
// # Generated documentation
//
// [Analysis.Markdown] renders the index and violation count as Markdown.
// [Analysis.WriteMarkdown] writes that document and returns filesystem errors.
// Relative output paths are resolved from the analyzed module's go.mod
// directory, and an empty output path uses [DefaultDocFile].
//
// Tests that keep generated architecture documentation in the repository can
// use [Project.WriteDoc], which performs the analysis and reports failures
// through testing.TB:
//
//	func TestArchitectureDoc(t *testing.T) {
//		tddcheck.Project{Root: "internal"}.WriteDoc(t, "")
//	}
package tddcheck
