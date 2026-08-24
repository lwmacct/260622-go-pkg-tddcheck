package rulekit

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// Snapshot is the analysis-local source model shared by every rule and
// extractor in one run. AST positions from all packages belong to Fset.
type Snapshot struct {
	Root        string
	ProjectRoot string
	ModulePath  string
	Config      Config
	Profile     Profile
	Fset        *token.FileSet
	Packages    []GoPackage
	Files       []GoFile
	LoadErrors  []LoadError

	filesByPath  map[string]bool
	loadErrorSet map[string]bool
}

// GoPackage contains the semantic information produced for one buildable Go
// package under the analyzed root.
type GoPackage struct {
	ID        string
	Name      string
	Path      string
	Dir       string
	Types     *types.Package
	TypesInfo *types.Info
	Syntax    []*ast.File
}

// GoFile joins filesystem, syntax, package, and type information for a source
// file selected by the active build configuration.
type GoFile struct {
	AbsPath     string
	RelPath     string
	Dir         string
	Base        string
	IsTest      bool
	Layer       string
	PackageID   string
	PackagePath string
	Fset        *token.FileSet
	AST         *ast.File
	Types       *types.Package
	TypesInfo   *types.Info
	Imports     []Import
}

type Import struct {
	Name        string
	Path        string
	PackagePath string
	Line        int
	Column      int
}

type LoadError struct {
	PackagePath string        `json:"packagePath"`
	Position    string        `json:"position,omitempty"`
	Message     string        `json:"message"`
	Kind        LoadErrorKind `json:"kind"`
}

type LoadErrorKind string

const (
	LoadErrorList  LoadErrorKind = "list"
	LoadErrorParse LoadErrorKind = "parse"
	LoadErrorType  LoadErrorKind = "type"
)

// Load constructs a single semantic snapshot for root. Package type errors are
// retained in Snapshot.LoadErrors so rules can still inspect work in progress;
// syntax and driver failures prevent construction.
func Load(ctx context.Context, root string, config Config) (*Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resolved, err := ResolveRoot(root)
	if err != nil {
		return nil, err
	}
	config, err = config.Compile()
	if err != nil {
		return nil, err
	}
	projectRoot, err := FindProjectRoot(resolved)
	if err != nil {
		return nil, err
	}
	modulePath, err := modulePath(projectRoot)
	if err != nil {
		return nil, err
	}
	pattern, err := packagePattern(projectRoot, resolved)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	loaded, err := packages.Load(&packages.Config{
		Context:    ctx,
		Dir:        projectRoot,
		Fset:       fset,
		Tests:      config.IncludeTests,
		BuildFlags: slices.Clone(config.BuildFlags),
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedModule,
	}, pattern)
	if err != nil {
		return nil, fmt.Errorf("load Go packages: %w", err)
	}

	snapshot := &Snapshot{
		Root:         resolved,
		ProjectRoot:  projectRoot,
		ModulePath:   modulePath,
		Config:       config,
		Profile:      config.Profile(),
		Fset:         fset,
		filesByPath:  make(map[string]bool),
		loadErrorSet: make(map[string]bool),
	}
	for _, loadedPackage := range loaded {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		snapshot.addPackage(loadedPackage)
	}
	for _, loadErr := range snapshot.LoadErrors {
		if loadErr.Kind == LoadErrorParse {
			return nil, fmt.Errorf("parse package %s: %s", loadErr.PackagePath, loadErr.Message)
		}
	}
	if len(snapshot.Packages) == 0 {
		return nil, fmt.Errorf("no Go packages found under %s", resolved)
	}
	slices.SortFunc(snapshot.Packages, func(a, b GoPackage) int {
		return strings.Compare(a.Path, b.Path)
	})
	slices.SortFunc(snapshot.Files, func(a, b GoFile) int {
		return strings.Compare(a.AbsPath, b.AbsPath)
	})
	slices.SortFunc(snapshot.LoadErrors, func(a, b LoadError) int {
		return strings.Compare(a.PackagePath+"\x00"+a.Position+"\x00"+a.Message, b.PackagePath+"\x00"+b.Position+"\x00"+b.Message)
	})
	if config.StrictPackages && len(snapshot.LoadErrors) > 0 {
		first := snapshot.LoadErrors[0]
		return nil, fmt.Errorf("load package %s: %s", first.PackagePath, first.Message)
	}
	return snapshot, nil
}

func (s *Snapshot) addPackage(loaded *packages.Package) {
	packageDir := ""
	if len(loaded.CompiledGoFiles) > 0 {
		packageDir = filepath.Dir(loaded.CompiledGoFiles[0])
	} else if len(loaded.GoFiles) > 0 {
		packageDir = filepath.Dir(loaded.GoFiles[0])
	}
	if packageDir != "" && s.skipDir(packageDir) {
		return
	}
	goPackage := GoPackage{
		ID:        loaded.ID,
		Name:      loaded.Name,
		Path:      loaded.PkgPath,
		Types:     loaded.Types,
		TypesInfo: loaded.TypesInfo,
		Syntax:    loaded.Syntax,
	}
	goPackage.Dir = packageDir
	s.Packages = append(s.Packages, goPackage)
	for _, loadErr := range loaded.Errors {
		key := loaded.PkgPath + "\x00" + loadErr.Pos + "\x00" + loadErr.Msg
		if s.loadErrorSet[key] {
			continue
		}
		s.LoadErrors = append(s.LoadErrors, LoadError{
			PackagePath: loaded.PkgPath,
			Position:    loadErr.Pos,
			Message:     loadErr.Msg,
			Kind:        loadErrorKind(loadErr.Kind),
		})
		s.loadErrorSet[key] = true
	}
	for _, parsedFile := range loaded.Syntax {
		filename := s.Fset.PositionFor(parsedFile.Pos(), true).Filename
		if filename == "" || !pathWithin(s.Root, filename) || s.skipFile(filename) {
			continue
		}
		if s.filesByPath[filename] {
			continue
		}
		file, err := s.goFile(loaded, filename, parsedFile)
		if err != nil {
			s.LoadErrors = append(s.LoadErrors, LoadError{
				PackagePath: loaded.PkgPath,
				Position:    filename,
				Message:     err.Error(),
				Kind:        LoadErrorParse,
			})
			continue
		}
		s.Files = append(s.Files, file)
		s.filesByPath[filename] = true
	}
}

func loadErrorKind(kind packages.ErrorKind) LoadErrorKind {
	switch kind {
	case packages.ParseError:
		return LoadErrorParse
	case packages.TypeError:
		return LoadErrorType
	default:
		return LoadErrorList
	}
}

func (s *Snapshot) goFile(loaded *packages.Package, filename string, parsedFile *ast.File) (GoFile, error) {
	rel, err := filepath.Rel(s.Root, filename)
	if err != nil {
		return GoFile{}, err
	}
	file := GoFile{
		AbsPath:     filename,
		RelPath:     filepath.ToSlash(rel),
		Dir:         filepath.ToSlash(filepath.Dir(rel)),
		Base:        filepath.Base(filename),
		IsTest:      strings.HasSuffix(filename, "_test.go"),
		Layer:       LayerForRelPath(rel, s.Config.LayerDirs),
		PackageID:   loaded.ID,
		PackagePath: loaded.PkgPath,
		Fset:        s.Fset,
		AST:         parsedFile,
		Types:       loaded.Types,
		TypesInfo:   loaded.TypesInfo,
	}
	for _, spec := range parsedFile.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return GoFile{}, fmt.Errorf("invalid import path %s: %w", spec.Path.Value, err)
		}
		name := ""
		if spec.Name != nil {
			name = spec.Name.Name
		}
		position := s.Fset.PositionFor(spec.Pos(), true)
		target := loaded.Imports[path]
		targetPath := path
		if target != nil && target.PkgPath != "" {
			targetPath = target.PkgPath
		}
		file.Imports = append(file.Imports, Import{
			Name:        name,
			Path:        path,
			PackagePath: targetPath,
			Line:        position.Line,
			Column:      position.Column,
		})
	}
	return file, nil
}

func (s *Snapshot) skipFile(filename string) bool {
	return s.skipDir(filepath.Dir(filename))
}

func (s *Snapshot) skipDir(dir string) bool {
	rel, err := filepath.Rel(s.Root, dir)
	if err != nil {
		return true
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		if part == "." {
			continue
		}
		if ShouldSkipModuleScanDir(part, s.Config) {
			return true
		}
	}
	return false
}

func (s *Snapshot) DisplayPath(filename string) string {
	if relative, err := filepath.Rel(s.ProjectRoot, filename); err == nil && pathWithin(s.ProjectRoot, filename) {
		return filepath.ToSlash(relative)
	}
	return filepath.ToSlash(filename)
}

func (s *Snapshot) Package(path string) (GoPackage, bool) {
	for _, goPackage := range s.Packages {
		if goPackage.Path == path {
			return goPackage, true
		}
	}
	return GoPackage{}, false
}

func (f GoFile) TypeOf(expr ast.Expr) types.Type {
	if f.TypesInfo == nil {
		return nil
	}
	return f.TypesInfo.TypeOf(expr)
}

func ResolveRoot(root string) (string, error) {
	if root == "" {
		root = "internal"
	}
	if filepath.IsAbs(root) {
		return filepath.Clean(root), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	projectRoot, err := FindProjectRoot(wd)
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, root), nil
}

func packagePattern(projectRoot string, root string) (string, error) {
	rel, err := filepath.Rel(projectRoot, root)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return "./...", nil
	}
	if rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		return "", fmt.Errorf("analysis root %s is outside module %s", root, projectRoot)
	}
	return "./" + filepath.ToSlash(rel) + "/...", nil
}

func modulePath(projectRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return "", err
	}
	path := modfile.ModulePath(data)
	if path == "" {
		return "", errorsNewModulePath()
	}
	return path, nil
}

func errorsNewModulePath() error {
	return fmt.Errorf("module path not found in go.mod")
}

func pathWithin(root string, filename string) bool {
	rel, err := filepath.Rel(root, filename)
	return err == nil && rel != ".." && !strings.HasPrefix(filepath.ToSlash(rel), "../")
}

func LayerForRelPath(rel string, layers []string) string {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if slices.Contains(layers, part) {
			return part
		}
	}
	return ""
}
