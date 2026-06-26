package rulekit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

type Diagnostic struct {
	Rule    string
	File    string
	Line    int
	Message string
}

type Rule interface {
	ID() string
	Check(*Context) ([]Diagnostic, error)
}

type Context struct {
	Root       string
	ModulePath string
	Config     Config
	Profile    Profile
	Files      []GoFile
}

type GoFile struct {
	AbsPath string
	RelPath string
	Dir     string
	Base    string
	Layer   string
	Fset    *token.FileSet
	AST     *ast.File
	Imports []Import
}

type Import struct {
	Path string
	Line int
}

func NewContext(root string, ruleName string, config Config) (*Context, error) {
	resolved, err := ResolveRuleRoot(root, ruleName)
	if err != nil {
		return nil, err
	}
	config = config.WithDefaults()
	profile := config.Profile()
	modulePath, err := ModulePathForRoot(resolved)
	if err != nil {
		return nil, err
	}
	files, err := scanGoFiles(resolved, config, profile)
	if err != nil {
		return nil, err
	}
	return &Context{
		Root:       resolved,
		ModulePath: modulePath,
		Config:     config,
		Profile:    profile,
		Files:      files,
	}, nil
}

func (c *Context) DisplayPath(filename string) string {
	if relative, err := filepath.Rel(c.ProjectRoot(), filename); err == nil && !strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(relative)
	}
	return filepath.ToSlash(filename)
}

func (c *Context) ProjectRoot() string {
	projectRoot, err := FindProjectRoot(c.Root)
	if err != nil {
		return c.Root
	}
	return projectRoot
}

func scanGoFiles(root string, config Config, profile Profile) ([]GoFile, error) {
	var files []GoFile
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if ShouldSkipModuleScanDir(entry.Name(), config) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		file, err := parseGoFile(root, path, profile)
		if err != nil {
			return err
		}
		files = append(files, file)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(files, func(a, b GoFile) int {
		return strings.Compare(a.AbsPath, b.AbsPath)
	})
	return files, nil
}

func parseGoFile(root string, filename string, profile Profile) (GoFile, error) {
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, filename, nil, parser.SkipObjectResolution)
	if err != nil {
		return GoFile{}, err
	}
	rel, err := filepath.Rel(root, filename)
	if err != nil {
		return GoFile{}, err
	}
	goFile := GoFile{
		AbsPath: filename,
		RelPath: filepath.ToSlash(rel),
		Dir:     filepath.ToSlash(filepath.Dir(rel)),
		Base:    filepath.Base(filename),
		Layer:   LayerForRelPath(rel, profile.LayerNames()),
		Fset:    fileSet,
		AST:     parsedFile,
	}
	for _, importSpec := range parsedFile.Imports {
		goFile.Imports = append(goFile.Imports, Import{
			Path: strings.Trim(importSpec.Path.Value, `"`),
			Line: fileSet.Position(importSpec.Pos()).Line,
		})
	}
	return goFile, nil
}

func LayerForRelPath(rel string, layers []string) string {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if slices.Contains(layers, part) {
			return part
		}
	}
	return ""
}

func FreeFile(base string) bool {
	return base == "x_free.go"
}
