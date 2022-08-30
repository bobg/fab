package fab

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/tools/go/packages"
)

// Main is the structure whose Run methods implements the main logic of the fab command.
type Main struct {
	// Pkgdir is where to find the user's build-rules Go package, e.g. "fab.d".
	Pkgdir string

	// Fabdir is where to find the user's hash DB and compiled binaries, default $HOME/.fab.
	Fabdir string

	// Verbose tells whether to run the driver in verbose mode
	// (by supplying the -v command-line flag).
	Verbose bool

	// List tells whether to run the driver in list-targets mode
	// (by supplying the -list command-line flag).
	List bool

	// Force tells whether to force recompilation of the driver before running it.
	Force bool

	// Args contains the additional command-line arguments to pass to the driver, e.g. target names.
	Args []string
}

// Run executes the main logic of the fab command.
// If a driver binary with the right dirhash does not exist in m.Fabdir,
// or if m.Force is true,
// it is created with Compile.
// It is then invoked with the command-line arguments indicated by the fields of m.
// Typically this will include one or more target names,
// in which case the driver will execute the associated rules as defined by the code in m.Pkgdir.
func (m Main) Run(ctx context.Context) error {
	pkgdir := m.Pkgdir
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
			return fmt.Errorf("package dir %s is not in or under current directory", pkgdir)
		}
		pkgdir = rel
	}
	pkgpath := "./" + filepath.Clean(pkgdir)

	args := []string{"-fab", m.Fabdir}
	if m.Verbose {
		args = append(args, "-v")
	}
	if m.List {
		args = append(args, "-list")
	}
	args = append(args, m.Args...)

	config := &packages.Config{
		Mode:    packages.NeedName | packages.NeedFiles,
		Context: ctx,
	}
	pkgs, err := packages.Load(config, pkgpath)
	if err != nil {
		return errors.Wrapf(err, "loading %s", pkgpath)
	}
	if len(pkgs) != 1 {
		return fmt.Errorf("found %d packages in %s, want 1", len(pkgs), pkgpath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		err = nil
		for _, e := range pkg.Errors {
			err = multierr.Append(err, e)
		}
		return errors.Wrapf(err, "loading package %s", pkg.Name)
	}

	// fset := token.NewFileSet()
	// pkgmap, err := parser.ParseDir(fset, m.Pkgdir, nil, 0)
	// if err != nil {
	// 	return errors.Wrapf(err, "parsing directory %s", m.Pkgdir)
	// }

	// if len(pkgmap) != 1 {
	// 	return fmt.Errorf("found %d Go packages in %s (want 1)", len(pkgmap), m.Pkgdir)
	// }

	// var (
	// 	pkgname string
	// 	astpkg  *ast.Package
	// )
	// for n, p := range pkgmap {
	// 	pkgname, astpkg = n, p
	// 	break
	// }

	// conf := types.Config{
	// 	Importer: importer.Default(),
	// }
	// pkg, err := conf.Check(pkgname, fset, maps.Values(astpkg.Files), nil)
	// if err != nil {
	// 	return errors.Wrapf(err, "type-checking package %s in directory %s", pkgname, m.Pkgdir)
	// }

	driverdir := filepath.Join(m.Fabdir, pkg.PkgPath)
	if err = os.MkdirAll(driverdir, 0755); err != nil {
		return errors.Wrapf(err, "ensuring directory %s/%s exists", m.Fabdir, pkg.PkgPath)
	}

	var (
		hashfile = filepath.Join(driverdir, "hash")
		driver   = filepath.Join(driverdir, "fab.bin")
		compile  bool
		oldhash  []byte
	)

	if m.Force {
		compile = true
		if m.Verbose {
			fmt.Println("Forcing recompilation of driver")
		}
	} else {
		oldhash, err = os.ReadFile(hashfile)
		if errors.Is(err, fs.ErrNotExist) {
			compile = true
			if m.Verbose {
				fmt.Println("Compiling driver")
			}
		} else if err != nil {
			return errors.Wrapf(err, "reading %s", hashfile)
		}
	}

	dh := newDirHasher()
	for _, filename := range pkg.GoFiles {
		if err = addFileToHash(dh, filename); err != nil {
			return errors.Wrapf(err, "hashing file %s", filename)
		}
	}
	newhash, err := dh.hash()
	if err != nil {
		return errors.Wrapf(err, "computing hash of directory %s", m.Pkgdir)
	}

	if !compile {
		if newhash == string(oldhash) {
			if m.Verbose {
				fmt.Println("Using existing driver")
			}
		} else {
			compile = true
			if m.Verbose {
				fmt.Println("Recompiling driver")
			}
		}
	}

	if compile {
		if err = Compile(ctx, m.Pkgdir, driver); err != nil {
			return errors.Wrapf(err, "compiling driver %s", driver)
		}
		if err = os.WriteFile(hashfile, []byte(newhash), 0644); err != nil {
			return errors.Wrapf(err, "writing %s", hashfile)
		}
	}

	cmd := exec.CommandContext(ctx, driver, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	return errors.Wrapf(err, "running %s %s", driver, strings.Join(args, " "))
}

func addFileToHash(dh *dirHasher, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "opening %s", filename)
	}
	defer f.Close()

	return dh.file(filename, f)
}
