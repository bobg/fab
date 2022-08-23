package fab

import (
	"context"
	"fmt"
	"go/ast"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/fatih/camelcase"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

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
	pkgs, err := packages.Load(config, pkgpath)
	if err != nil {
		return errors.Wrapf(err, "loading %s", pkgpath)
	}
	if len(pkgs) != 1 {
		return errors.Wrapf(err, "found %d packages in %s, want 1", len(pkgs), pkgpath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		err = nil
		for _, e := range pkg.Errors {
			err = multierr.Append(err, e)
		}
		return err
	}
	c := compiler{pkg: pkg, pkgdir: pkgdir, binfile: binfile}
	return c.compile(ctx)
}

type compiler struct {
	pkg     *packages.Package
	pkgdir  string
	binfile string
}

func (c *compiler) compile(ctx context.Context) error {
	var (
		scope   = c.pkg.Types.Scope()
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
		return fmt.Errorf("found no targets after loading %s", c.pkg.Name)
	}

	sort.Strings(targets)

	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		return errors.Wrap(err, "creating tempdir")
	}
	defer os.RemoveAll(tmpdir)

	if err = populateFabDir(tmpdir); err != nil {
		return errors.Wrap(err, "copying fab code")
	}

	subpkgdir := filepath.Join(tmpdir, "pkg", c.pkg.Name)
	if err = os.MkdirAll(subpkgdir, 0755); err != nil {
		return errors.Wrapf(err, "creating %s", subpkgdir)
	}

	entries, err := os.ReadDir(c.pkgdir)
	if err != nil {
		return errors.Wrapf(err, "reading entries from %s", c.pkgdir)
	}

	dh := NewDirHasher()

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
		if err = copyAndHash(filepath.Join(c.pkgdir, entry.Name()), subpkgdir, dh); err != nil {
			return errors.Wrapf(err, "copying %s to tmp subdir", entry.Name())
		}
	}

	dirhash, err := dh.Hash()
	if err != nil {
		return errors.Wrap(err, "getting dirhash")
	}

	type templateTarget struct {
		Name, SnakeName string
	}
	data := struct {
		Subpkg  string
		Dirhash string
		Pkgdir  string
		Binfile string
		Targets []templateTarget
	}{
		Subpkg:  c.pkg.Name,
		Dirhash: dirhash,
		Pkgdir:  c.pkgdir,
		Binfile: c.binfile,
	}
	for _, target := range targets {
		data.Targets = append(data.Targets, templateTarget{
			Name:      target,
			SnakeName: toSnake(target),
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

	return os.Rename(filepath.Join(tmpdir, "x"), c.binfile)
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

func copyAndHash(filename, destdir string, dh *DirHasher) error {
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

	// Send one copy of the file to out and one copy to the dirhasher.
	tee := io.TeeReader(in, out)
	return dh.File(filename, tee)
}

func toSnake(inp string) string {
	parts := camelcase.Split(inp)
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "_")
}
