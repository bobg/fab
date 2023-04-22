package deps

import (
	"path/filepath"
	"sort"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/set"
	"golang.org/x/tools/go/packages"
)

// Go produces the list of files involved in building the Go package in the given directory.
// It traverses package dependencies transitively,
// but only within the original package's module.
// The list is sorted for consistent, predictable results.
func Go(dir string, recursive bool) ([]string, error) {
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedEmbedFiles | packages.NeedEmbedPatterns | packages.NeedTypes | packages.NeedDeps | packages.NeedImports | packages.NeedModule,
		Dir:  dir,
	}

	arg := "."
	if recursive {
		arg = "./..."
	}

	pkgs, err := packages.Load(config, arg)
	if err != nil {
		return nil, errors.Wrapf(err, "loading from %s", dir)
	}

	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			err = errors.Join(err, e)
		}
	}
	if err != nil {
		return nil, errors.Wrapf(err, "after loading from %s", dir)
	}

	files := set.New[string]()
	for _, pkg := range pkgs {
		if err = gopkgAdd(pkg, pkg.Module.Path, files); err != nil {
			return nil, errors.Wrapf(err, "adding files from %s", pkg.PkgPath)
		}
	}

	slice := files.Slice()
	sort.Strings(slice)
	return slice, nil
}

func gopkgAdd(pkg *packages.Package, modpath string, files set.Of[string]) error {
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
		if err := gopkgAdd(imp, modpath, files); err != nil {
			return errors.Wrapf(err, "in import of %s", imp.PkgPath)
		}
	}
	return nil
}
