// Package loader parses and type-checks Go packages using only the standard
// library. It deliberately avoids golang.org/x/tools/go/packages so gox has
// zero external dependencies.
//
// The loader shells out to `go list -json` to discover the import graph and
// then drives go/parser + go/types directly. List is split from
// LoadPackage so callers can cheaply enumerate packages (and their file
// metadata) without paying for parse + type-check on cache hits.
package loader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

// PackageInfo is the lightweight package metadata produced by List.
type PackageInfo struct {
	ImportPath string
	Dir        string
	GoFiles    []string // basenames as reported by `go list`
}

// AbsFiles returns the absolute paths of the package's .go files.
func (p *PackageInfo) AbsFiles() []string {
	out := make([]string, len(p.GoFiles))
	for i, name := range p.GoFiles {
		out[i] = filepath.Join(p.Dir, name)
	}
	return out
}

// Package is a fully-parsed and type-checked Go package.
type Package struct {
	Info       *PackageInfo
	Fset       *token.FileSet
	Files      []*ast.File
	Pkg        *types.Package
	TypesInfo  *types.Info
	TypeErrors []error
}

// listEntry mirrors the subset of `go list -json` output we need.
type listEntry struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Error      *struct{ Err string }
}

// List enumerates the packages matched by the patterns and returns their
// file metadata without parsing them.
func List(patterns ...string) ([]*PackageInfo, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	args := append([]string{"list", "-json", "-e"}, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr
	out, runErr := cmd.Output()
	if runErr != nil {
		return nil, fmt.Errorf("go list: %w", runErr)
	}

	var infos []*PackageInfo
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var e listEntry
		if decErr := dec.Decode(&e); decErr != nil {
			return nil, fmt.Errorf("decode go list: %w", decErr)
		}
		if e.Error != nil {
			fmt.Fprintf(os.Stderr, "gox: %s: %s\n", e.ImportPath, e.Error.Err)
			continue
		}
		if len(e.GoFiles) == 0 {
			continue
		}
		infos = append(infos, &PackageInfo{
			ImportPath: e.ImportPath,
			Dir:        e.Dir,
			GoFiles:    e.GoFiles,
		})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].ImportPath < infos[j].ImportPath })
	return infos, nil
}

// LoadPackage parses and type-checks a single package.
func LoadPackage(info *PackageInfo) (*Package, error) {
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(info.GoFiles))
	for _, name := range info.GoFiles {
		path := filepath.Join(info.Dir, name)
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, parseErr)
		}
		files = append(files, f)
	}

	tInfo := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Implicits:  map[ast.Node]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     map[ast.Node]*types.Scope{},
	}

	var typeErrs []error
	conf := &types.Config{
		Importer: importer.Default(),
		Error:    func(err error) { typeErrs = append(typeErrs, err) },
	}
	pkg, checkErr := conf.Check(info.ImportPath, fset, files, tInfo)
	if checkErr != nil && !errors.As(checkErr, new(types.Error)) {
		return nil, checkErr
	}

	return &Package{
		Info:       info,
		Fset:       fset,
		Files:      files,
		Pkg:        pkg,
		TypesInfo:  tInfo,
		TypeErrors: typeErrs,
	}, nil
}

// Load is a convenience wrapper that lists and fully loads every package
// matched by patterns. Use List + LoadPackage when you want per-package
// control (e.g. to consult a cache before parsing).
func Load(patterns ...string) ([]*Package, error) {
	infos, listErr := List(patterns...)
	if listErr != nil {
		return nil, listErr
	}
	pkgs := make([]*Package, 0, len(infos))
	for _, info := range infos {
		p, loadErr := LoadPackage(info)
		if loadErr != nil {
			return nil, fmt.Errorf("%s: %w", info.ImportPath, loadErr)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}
