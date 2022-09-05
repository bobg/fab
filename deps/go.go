package deps

import (
	"fmt"
	"path/filepath"

	"github.com/bobg/go-generics/set"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"

	"github.com/bobg/fab"
)

// GoPkg produces the list of files involved in building the Go package in the given directory.
// It traverses package dependencies transitively, but only within the original package's module.
func GoPkg(dir string) ([]string, error) {
	pkgpath, err := fab.ToRelPath(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "getting relative path for %s", dir)
	}
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedEmbedFiles | packages.NeedEmbedPatterns | packages.NeedTypes | packages.NeedDeps | packages.NeedImports | packages.NeedModule,
	}

	pkgs, err := packages.Load(config, pkgpath)
	if err != nil {
		return nil, errors.Wrapf(err, "loading package from %s", pkgpath)
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("found %d packages in %s, want 1", len(pkgs), pkgpath)
	}

	modpath := pkgs[0].Module.Path

	var (
		files  = set.New[string]()
		addPkg func(*packages.Package) error
	)
	addPkg = func(pkg *packages.Package) error {
		if pkg.Module == nil {
			return nil
		}
		if pkg.Module.Path != modpath {
			return nil
		}
		files.Add(pkg.GoFiles...)
		files.Add(pkg.CompiledGoFiles...)
		files.Add(pkg.OtherFiles...)
		files.Add(pkg.EmbedFiles...)
		for _, pattern := range pkg.EmbedPatterns {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return errors.Wrapf(err, "in pattern %s", pattern)
			}
			files.Add(matches...)
		}
		for _, imp := range pkg.Imports {
			if err := addPkg(imp); err != nil {
				return errors.Wrapf(err, "in import of %s", imp.PkgPath)
			}
		}
		return nil
	}
	err = addPkg(pkgs[0])
	return files.Slice(), err
}
