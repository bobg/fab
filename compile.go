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
	"strconv"
	"strings"
	"text/template"

	"github.com/bobg/go-generics/maps"
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
// When Compile runs
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
//     that registers (with Register) that set of targets.
//   - The go compiler is invoked to produce an executable,
//     which is renamed into place as binfile.
//
// For the synthesized calls to Register on Target-valued variables,
// the driver uses the variable's name as the "name" argument
// and the variable's doc comment as the "doc" argument.
//
// The user's code is able to make its own calls to Register during program initialization
// in order to augment the set of available targets.
func Compile(ctx context.Context, pkgdir, binfile string) error {
	config := &packages.Config{
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedDeps,
		Context: ctx,
		Dir:     pkgdir,
	}
	ppkgs, err := packages.Load(config, ".")
	if err != nil {
		return errors.Wrapf(err, "loading %s", pkgdir)
	}
	if len(ppkgs) != 1 {
		return fmt.Errorf("found %d packages in %s, want 1", len(ppkgs), pkgdir)
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

	type targetPair struct {
		Name, Doc string
	}
	var (
		scope   = ppkg.Types.Scope()
		idents  = scope.Names()
		targets = make(map[string]*targetPair)
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
		targets[ident] = &targetPair{Name: ident}
	}

	var (
		dpkg   = doc.New(astpkg, pkgdir, 0)
		parser = dpkg.Parser()
		pr     = dpkg.Printer()
	)
	for _, v := range dpkg.Vars {
		for _, name := range v.Names {
			if tp, ok := targets[name]; ok {
				dstr := string(pr.Text(parser.Parse(v.Doc)))
				dstr = strings.TrimRight(dstr, "\r\n")
				tp.Doc = strconv.Quote(dstr)
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

	data := struct {
		Subpkg  string
		Targets []*targetPair
	}{
		Subpkg:  ppkg.Name,
		Targets: maps.Values(targets),
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
