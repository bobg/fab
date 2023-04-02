package fab

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bobg/go-generics/v2/slices"
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
// A driver binary with a name matching m.Pkgdir is sought in m.Fabdir.
// If it does not exist,
// or if its corresponding dirhash is wrong
// (i.e., out of date with respect to the user's code),
// or if m.Force is true,
// it is created with Compile.
// It is then invoked with the command-line arguments indicated by the fields of m.
// Typically this will include one or more target names,
// in which case the driver will execute the associated rules
// as defined by the code in m.Pkgdir.
func (m Main) Run(ctx context.Context) error {
	args := []string{"-fab", m.Fabdir}
	if m.Verbose {
		args = append(args, "-v")
	}
	if m.List {
		args = append(args, "-list")
	}
	if m.Force {
		args = append(args, "-f")
	}
	args = append(args, m.Args...)

	driver, err := m.getDriver(ctx)
	if err != nil {
		return errors.Wrap(err, "ensuring driver is up to date")
	}

	cmd := exec.CommandContext(ctx, driver, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	return errors.Wrapf(err, "running %s %s", driver, strings.Join(args, " "))
}

func (m Main) getDriver(ctx context.Context) (string, error) {
	config := &packages.Config{
		Mode:    LoadMode,
		Context: ctx,
		Dir:     m.Pkgdir,
	}
	pkgs, err := packages.Load(config, ".")
	if err != nil {
		return "", errors.Wrapf(err, "loading %s", m.Pkgdir)
	}
	if len(pkgs) != 1 {
		return "", fmt.Errorf(
			"found %d packages in %s, want 1 %v",
			len(pkgs),
			m.Pkgdir,
			slices.Map(pkgs, func(p *packages.Package) string { return p.PkgPath }),
		)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		err = nil
		for _, e := range pkg.Errors {
			err = multierr.Append(err, e)
		}
		return "", errors.Wrapf(err, "loading package %s", pkg.Name)
	}

	driverdir := filepath.Join(m.Fabdir, pkg.PkgPath)
	if err = os.MkdirAll(driverdir, 0755); err != nil {
		return "", errors.Wrapf(err, "ensuring directory %s/%s exists", m.Fabdir, pkg.PkgPath)
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
		_, err = os.Stat(driver)
		if errors.Is(err, fs.ErrNotExist) {
			compile = true
			if m.Verbose {
				fmt.Println("Compiling driver")
			}
		} else if err != nil {
			return "", errors.Wrapf(err, "statting %s", driver)
		}
	}

	if !compile {
		oldhash, err = os.ReadFile(hashfile)
		if errors.Is(err, fs.ErrNotExist) {
			compile = true
			if m.Verbose {
				fmt.Println("Compiling driver")
			}
		} else if err != nil {
			return "", errors.Wrapf(err, "reading %s", hashfile)
		}
	}

	dh := newDirHasher()
	for _, filename := range pkg.GoFiles { // xxx other files too?
		if err = addFileToHash(dh, filename); err != nil {
			return "", errors.Wrapf(err, "hashing file %s", filename)
		}
	}
	newhash, err := dh.hash()
	if err != nil {
		return "", errors.Wrapf(err, "computing hash of directory %s", m.Pkgdir)
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
		if err = CompilePackage(ctx, pkg, driver); err != nil {
			return "", errors.Wrapf(err, "compiling driver %s", driver)
		}
		if err = os.WriteFile(hashfile, []byte(newhash), 0644); err != nil {
			return "", errors.Wrapf(err, "writing %s", hashfile)
		}
	}

	return driver, nil
}

func addFileToHash(dh *dirHasher, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "opening %s", filename)
	}
	defer f.Close()

	return dh.file(filename, f)
}
