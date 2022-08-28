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
// If m.Binfile does not exist,
// or if m.Force is true,
// it is created with Compile.
// It is then invoked with the command-line arguments indicated by the fields of m.
// Typically this will include one or more target names,
// in which case the driver will execute the associated rules as defined by the code in m.Pkgdir.
func (m Main) Run(ctx context.Context) error {
	args := []string{"-fab", m.Fabdir}
	if m.Verbose {
		args = append(args, "-v")
	}
	if m.List {
		args = append(args, "-list")
	}
	args = append(args, m.Args...)

	driver, err := m.getDriver(ctx)
	if err != nil {
		return errors.Wrap(err, "ensuring driver binary is up to date")
	}

	cmd := exec.CommandContext(ctx, driver, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	return errors.Wrapf(err, "running %s %s", driver, strings.Join(args, " "))
}

func (m Main) getDriver(ctx context.Context) (string, error) {
	entries, err := os.ReadDir(m.Pkgdir)
	if err != nil {
		return "", errors.Wrapf(err, "reading directory %s", m.Pkgdir)
	}
	dh := newDirHasher()
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
		if err = addFileToHash(dh, filepath.Join(m.Pkgdir, entry.Name())); err != nil {
			return "", errors.Wrapf(err, "hashing file %s/%s", m.Pkgdir, entry.Name())
		}
	}
	dhval, err := dh.hash()
	if err != nil {
		return "", errors.Wrapf(err, "computing hash for directory %s", m.Pkgdir)
	}

	var (
		driver  = filepath.Join(m.Fabdir, dhval)
		compile bool
	)
	info, err := os.Stat(driver)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if m.Verbose {
			fmt.Println("Compiling driver")
		}
		compile = true
	case err != nil:
		return "", errors.Wrapf(err, "statting %s", driver)
	case info.Mode().Perm()&1 == 0:
		return "", fmt.Errorf("file %s exists but is not world-executable", driver)
	case m.Force:
		if m.Verbose {
			fmt.Println("Forcing recompilation of driver")
		}
		compile = true
	case m.Verbose:
		fmt.Println("Using existing driver")
	}

	if !compile {
		return driver, nil
	}

	if err = os.MkdirAll(m.Fabdir, 0755); err != nil {
		return "", errors.Wrapf(err, "creating directory %s", m.Fabdir)
	}

	err = Compile(ctx, m.Pkgdir, driver)
	return driver, errors.Wrapf(err, "compiling %s", driver)
}

func addFileToHash(dh *dirHasher, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "opening %s", filename)
	}
	defer f.Close()

	return dh.file(filename, f)
}
