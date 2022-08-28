package fab

import (
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/fatih/camelcase"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// Compile compiles a "driver" from a directory of user code
// (combined with a main function supplied by fab)
// and places the executable result in a given file.
// The driver converts command-line target names into the necessary Fab rule invocations.
//
// The package of user code should contain one or more exported identifiers
// whose types satisfy the [Target] interface.
// These become the build rules that the driver can invoke.
//
// When the driver already exists, the "fab" command simply executes it.
// When it doesn't exist, the fab command compiles it and _then_ executes it.
//
// The driver binary knows the "dir hash" of the Go files from which it was compiled.
// When the driver runs, it checks that the dir hash is still the same.
// If it's not, then the build rules have changed and the driver binary is out of date.
// In this case the driver recompiles, replaces, and reruns itself.
//
// When Compile runs
// (including when the driver recompiles itself)
// the "go" program must exist in the user's PATH.
// It must be Go version 1.19 or later.
//
// How it works:
//
//   - The user's code is loaded with packages.Load.
//   - The set of exported top-level identifiers is filtered
//     to find those implementing the fab.Target interface.
//   - The user's code is then copied to a temp directory
//     together with a main package (and main() function)
//     that records the set of targets,
//     the dir hash of the user's code,
//     and the value of binfile.
//   - The go compiler is invoked to produce an executable,
//     which is renamed into place as binfile.
func Compile(ctx context.Context, pkgdir, binfile string) error {
	if filepath.IsAbs(pkgdir) {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "getting current directory")
		}
		rel, err := filepath.Rel(cwd, pkgdir)
		if err != nil {
			return errors.Wrapf(err, "getting relative path to %s", pkgdir)
		}
		if strings.HasPrefix(rel, "../") {
			return fmt.Errorf("pkgdir must be in or under current directory")
		}
		pkgdir = rel
	}
	pkgpath := "./" + filepath.Clean(pkgdir)

	config := &packages.Config{
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedDeps,
		Context: ctx,
	}
	ppkgs, err := packages.Load(config, pkgpath)
	if err != nil {
		return errors.Wrapf(err, "loading %s", pkgpath)
	}
	if len(ppkgs) != 1 {
		return fmt.Errorf("found %d packages in %s, want 1", len(ppkgs), pkgpath)
	}
	ppkg := ppkgs[0]
	if len(ppkg.Errors) > 0 {
		err = nil
		for _, e := range ppkg.Errors {
			err = multierr.Append(err, e)
		}
		return errors.Wrapf(err, "loading package %s", ppkg.Name)
	}

	fset := token.NewFileSet()
	astpkgs, err := parser.ParseDir(fset, pkgdir, nil, parser.ParseComments)
	if err != nil {
		return errors.Wrapf(err, "parsing %s", pkgdir)
	}
	if len(astpkgs) != 1 {
		return fmt.Errorf("found %d packages in %s, want 1", len(astpkgs), pkgdir)
	}
	astpkg, ok := astpkgs[ppkg.Name]
	if !ok {
		return fmt.Errorf("package %s not found in %s", ppkg.Name, pkgdir)
	}

	var (
		scope   = ppkg.Types.Scope()
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
		return fmt.Errorf("found no targets after loading %s", ppkg.Name)
	}

	sort.Strings(targets)

	docstrs := make(map[string]string) // ident -> docstring
	for _, target := range targets {
		docstrs[target] = ""
	}

	dpkg := doc.New(astpkg, pkgpath, 0)
	for _, v := range dpkg.Vars {
		for _, name := range v.Names {
			if _, ok := docstrs[name]; ok {
				dstr := string(dpkg.Text(v.Doc))
				dstr = strings.TrimSpace(dstr)
				docstrs[name] = strconv.Quote(dstr)
			}
		}
	}

	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		return errors.Wrap(err, "creating tempdir")
	}
	defer os.RemoveAll(tmpdir)

	if err = populateFabDir(tmpdir); err != nil {
		return errors.Wrap(err, "copying fab code")
	}

	subpkgdir := filepath.Join(tmpdir, "pkg", ppkg.Name)
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
		if strings.HasSuffix(entry.Name(), "_test.go") {
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
		Name, SnakeName, Doc string
	}
	data := struct {
		Subpkg  string
		Targets []templateTarget
	}{
		Subpkg: ppkg.Name,
	}
	for _, target := range targets {
		data.Targets = append(data.Targets, templateTarget{
			Name:      target,
			SnakeName: toSnake(target),
			Doc:       docstrs[target],
		})
	}

	driverOut, err := os.Create(filepath.Join(tmpdir, "driver.go"))
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
	cmd.Dir = tmpdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error in go mod init fab: %w; output follows\n%s", err, string(output))
	}

	gomodPath := filepath.Join(tmpdir, "go.mod")
	gomodData, err := os.ReadFile(gomodPath)
	if err != nil {
		return errors.Wrapf(err, "reading %s", gomodPath)
	}
	mf, err := modfile.Parse(gomodPath, gomodData, nil)
	if err != nil {
		return errors.Wrapf(err, "parsing %s", gomodPath)
	}
	if err = mf.AddReplace("github.com/bobg/fab", "", "./fab", ""); err != nil {
		return errors.Wrapf(err, "adding replace directive in %s", gomodPath)
	}
	gomodData, err = mf.Format()
	if err != nil {
		return errors.Wrapf(err, "formatting go.mod in %s", gomodPath)
	}
	if err = os.WriteFile(gomodPath, gomodData, 0644); err != nil {
		return errors.Wrapf(err, "rewriting %s", gomodPath)
	}

	cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = tmpdir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error in go mod tidy: %w; output follows\n%s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "go", "build")
	cmd.Dir = tmpdir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error in go build: %w; output follows\n%s", err, string(output))
	}

	return os.Rename(filepath.Join(tmpdir, "x"), binfile)
}

func populateFabDir(tmpdir string) error {
	return populateFabSubdir(filepath.Join(tmpdir, "fab"), ".")
}

func populateFabSubdir(destdir, subdir string) error {
	if err := os.MkdirAll(destdir, 0755); err != nil {
		return errors.Wrapf(err, "creating %s", destdir)
	}
	entries, err := embeds.ReadDir(subdir)
	if err != nil {
		return errors.Wrap(err, "reading embeds")
	}
	for _, entry := range entries {
		if entry.IsDir() {
			err = populateFabSubdir(filepath.Join(destdir, entry.Name()), filepath.Join(subdir, entry.Name()))
			if err != nil {
				return errors.Wrapf(err, "populating dir %s", entry.Name())
			}
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		contents, err := embeds.ReadFile(filepath.Join(subdir, entry.Name()))
		if err != nil {
			return errors.Wrapf(err, "reading embedded file %s", entry.Name())
		}
		dest := filepath.Join(destdir, entry.Name())
		err = os.WriteFile(dest, contents, 0644)
		if err != nil {
			return errors.Wrapf(err, "writing %s", dest)
		}
	}
	return nil
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
	return errors.Wrapf(err, "copying %s to %s", filename, destdir)
}

func toSnake(inp string) string {
	parts := camelcase.Split(inp)
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "_")
}
