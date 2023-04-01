package deps

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/bobg/go-generics/set"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

// Go produces the list of files involved in building the Go package in the given directory.
// It traverses package dependencies transitively,
// but only within the original package's module.
// The list is sorted for consistent, predictable results.
func Go(dir string) ([]string, error) {
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedEmbedFiles | packages.NeedEmbedPatterns | packages.NeedTypes | packages.NeedDeps | packages.NeedImports | packages.NeedModule,
		Dir:  dir,
	}

	pkgs, err := packages.Load(config, ".")
	if err != nil {
		return nil, errors.Wrapf(err, "loading package from %s", dir)
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("found %d packages in %s, want 1", len(pkgs), dir)
	}

	files := set.New[string]()
	err = gopkgAdd(pkgs[0], pkgs[0].Module.Path, files)
	slice := files.Slice()
	sort.Strings(slice)
	return slice, err
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
