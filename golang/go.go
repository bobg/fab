package golang

import (
	"path/filepath"
	"sort"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/set"
	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v3"

	"github.com/bobg/fab"
)

// Binary is a target describing how to compile a Go binary whose main package is in `dir`.
// The resulting binary gets written to `outfile`.
// Additional command-line arguments for `go build` can be specified with `flags`.
//
// A Binary target may be specified in YAML using the tag !go.Binary,
// which introduces a mapping whose fields are:
//
//   - Dir: the directory containing the main Go package
//   - Out: the output file that will contain the compiled binary
//   - Flags: a sequence of additional command-line flags for `go build`
func Binary(dir, outfile string, flags ...string) (fab.Target, error) {
	deps, err := Deps(dir, false)
	if err != nil {
		return nil, errors.Wrapf(err, "computing dependencies")
	}
	args := append([]string{"build", "-o", outfile, "-C", dir}, flags...)
	args = append(args, ".")
	c := &fab.Command{
		Cmd:  "go",
		Args: args,
	}
	return fab.Files(c, deps, []string{outfile}), nil
}

func binaryDecoder(node *yaml.Node) (fab.Target, error) {
	var b struct {
		Dir   string    `yaml:"Dir"`
		Out   string    `yaml:"Out"`
		Flags yaml.Node `yaml:"Flags"`
	}

	if err := node.Decode(&b); err != nil {
		return nil, errors.Wrap(err, "YAML error decoding go.Binary")
	}

	flags, err := fab.YAMLStringList(&b.Flags)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error decoding go.Binary.Flags")
	}

	return Binary(b.Dir, b.Out, flags...)
}

// Deps produces the list of files involved in building the Go package in the given directory.
// It traverses package dependencies transitively,
// but only within the original package's module.
// The list is sorted for consistent, predictable results.
func Deps(dir string, recursive bool) ([]string, error) {
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

func depsDecoder(node *yaml.Node) ([]string, error) {
	var gd struct {
		Dir       string `yaml:"Dir"`
		Recursive bool   `yaml:"Recursive"`
	}

	if err := node.Decode(&gd); err != nil {
		return nil, errors.Wrap(err, "YAML error decoding go.Deps")
	}

	return Deps(gd.Dir, gd.Recursive)
}

func init() {
	fab.RegisterYAMLTarget("go.Binary", binaryDecoder)
	fab.RegisterYAMLStringList("go.Deps", depsDecoder)
}
