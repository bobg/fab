package internal

import (
	"context"
	_ "embed" // For the go:embed below.
	"fmt"
	"go/ast"
	"go/types"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/fatih/camelcase"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"

	"github.com/bobg/fab"
)

//go:embed driver.go.tmpl
var driverStr string

// Load builds a Go binary out of the Go package at `pkgdir` and a special main function.
// It then prepares an exec.Cmd for running that binary and passes it to a callback function.
//
// After the Go package at `pkgdir` is loaded,
// it is scanned for exported symbols
// whose types satisfy the fab.Target interface.
// Those become named targets
// runnable from the command line of the generated binary.
// For the command line,
// the exported names are transformed from Upper to lower case,
// and from CamelCase to snake_case.
//
// The arguments for the generated binary are:
//
//   - "-v" [optional] run verbosely
//   - DIR [required] the directory in which rules should run
//   - OUTFILE [required] the name of a file in which to write the output
//
// ...followed by zero or more targets names
// (downcased and snake_cased as described).
// If there are zero targets,
// a target named Default is used,
// if one is defined;
// otherwise this is an error.
//
// The output of the binary,
// placed in OUTFILE,
// is a JSON-encoded array of error strings
// produced by the targets that ran.
// If this are no errors,
// this will be an empty array.
func Load(ctx context.Context, pkgdir string, f func(*exec.Cmd) error) error {
	loadpath := pkgdir
	if !filepath.IsAbs(pkgdir) {
		_, err := os.Stat(pkgdir)
		if errors.Is(err, fs.ErrNotExist) {
			// do nothing
		} else if err != nil {
			return errors.Wrapf(err, "statting %s", pkgdir)
		} else {
			loadpath = "./" + filepath.Clean(loadpath)
		}
	}
	config := &packages.Config{
		Mode:    packages.NeedName | packages.NeedTypes | packages.NeedDeps,
		Context: ctx,
	}
	pkgs, err := packages.Load(config, loadpath)
	if err != nil {
		return errors.Wrapf(err, "loading %s", loadpath)
	}
	if len(pkgs) != 1 {
		return errors.Wrapf(err, "found %d packages in %s, want 1", len(pkgs), loadpath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		var errs []string
		for _, e := range pkg.Errors {
			errs = append(errs, e.Error())
		}
		return fmt.Errorf("error(s) loading %s: %s", loadpath, strings.Join(errs, ";\n  "))
	}

	return LoadPkg(ctx, pkgdir, pkg.Name, pkg.Types.Scope(), f)
}

func LoadPkg(ctx context.Context, pkgdir, pkgname string, scope *types.Scope, f func(*exec.Cmd) error) error {
	var (
		idents  = scope.Names()
		targets []string // Top-level identifiers with types that implement fab.Target
	)
	for _, ident := range idents {
		if !ast.IsExported(ident) {
			continue
		}
		obj := scope.Lookup(ident)
		if obj == nil {
			continue
		}
		if err := checkImplementsTarget(obj.Type()); err != nil {
			continue
		}
		targets = append(targets, ident)
	}
	if len(targets) == 0 {
		return fmt.Errorf("found no targets after loading %s", pkgname)
	}

	sort.Strings(targets)

	return LoadTargets(ctx, pkgdir, pkgname, targets, f)
}

func LoadTargets(ctx context.Context, pkgdir, pkgname string, targets []string, f func(*exec.Cmd) error) error {
	dir, err := os.MkdirTemp("", "fab")
	if err != nil {
		return errors.Wrap(err, "creating tempdir")
	}
	defer os.RemoveAll(dir)

	fabsubdir := filepath.Join(dir, "fab")
	if err = os.Mkdir(fabsubdir, 0755); err != nil {
		return errors.Wrapf(err, "creating %s", fabsubdir)
	}
	goFiles, err := fab.GoFiles.ReadDir(".")
	if err != nil {
		return errors.Wrap(err, "reading GoFiles")
	}
	for _, goFile := range goFiles {
		contents, err := fab.GoFiles.ReadFile(goFile.Name())
		if err != nil {
			return errors.Wrapf(err, "reading Go file %s", goFile.Name())
		}
		dest := filepath.Join(fabsubdir, goFile.Name())
		err = os.WriteFile(dest, contents, 0644)
		if err != nil {
			return errors.Wrapf(err, "writing %s", dest)
		}
	}

	// TODO: refactor to harmonize the copying above with the copying below.

	subpkgdir := filepath.Join(dir, "pkg", pkgname)
	if err = os.MkdirAll(subpkgdir, 0755); err != nil {
		return errors.Wrapf(err, "creating %s", subpkgdir)
	}
	entries, err := os.ReadDir(pkgdir)
	if err != nil {
		return errors.Wrapf(err, "reading entries from %s", pkgdir)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if err = copyFile(filepath.Join(pkgdir, entry.Name()), subpkgdir); err != nil {
			return errors.Wrapf(err, "copying %s to tmp subdir", entry.Name())
		}
	}

	type templateTarget struct {
		Name, SnakeName string
	}
	data := struct {
		Subpkg  string
		Targets []templateTarget
	}{
		Subpkg: pkgname,
	}
	for _, target := range targets {
		data.Targets = append(data.Targets, templateTarget{
			Name:      target,
			SnakeName: toSnake(target),
		})
	}

	driverOut, err := os.Create(filepath.Join(dir, "driver.go"))
	if err != nil {
		return errors.Wrap(err, "creating driver.go in temp dir")
	}
	defer driverOut.Close()

	tmpl := template.New("")
	_, err = tmpl.Parse(driverStr)
	if err != nil {
		return errors.Wrap(err, "parsing driver template")
	}
	if err = tmpl.Execute(driverOut, data); err != nil {
		return errors.Wrap(err, "rendering driver.go template")
	}
	if err = driverOut.Close(); err != nil {
		return errors.Wrap(err, "closing driver.go")
	}

	cmd := exec.CommandContext(ctx, "go", "mod", "init", "x")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error in go mod init fab: %w; output follows\n%s", err, string(output))
	}

	gomodPath := filepath.Join(dir, "go.mod")
	gomodData, err := os.ReadFile(gomodPath)
	if err != nil {
		return errors.Wrapf(err, "reading %s", gomodPath)
	}
	mf, err := modfile.Parse(gomodPath, gomodData, nil)
	if err != nil {
		return errors.Wrapf(err, "parsing %s", gomodPath)
	}
	err = mf.AddReplace("github.com/bobg/fab", "", "./fab", "")
	if err != nil {
		return errors.Wrapf(err, "adding replace directive in %s", gomodPath)
	}
	gomodData, err = mf.Format()
	if err != nil {
		return errors.Wrapf(err, "formatting go.mod in %s", gomodPath)
	}
	err = os.WriteFile(gomodPath, gomodData, 0644)
	if err != nil {
		return errors.Wrapf(err, "rewriting %s", gomodPath)
	}

	cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error in go mod tidy: %w; output follows\n%s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "go", "build")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error in go build: %w; output follows\n%s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, filepath.Join(dir, "x"))
	return f(cmd)
}

func copyFile(filename, destdir string) error {
	outfilename := filepath.Join(destdir, filepath.Base(filename))
	out, err := os.OpenFile(outfilename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return errors.Wrapf(err, "creating %s", outfilename)
	}
	defer out.Close()

	in, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "opening %s", filename)
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return errors.Wrapf(err, "copying %s", filename)
	}

	return out.Close()
}

func toSnake(inp string) string {
	parts := camelcase.Split(inp)
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "_")
}
