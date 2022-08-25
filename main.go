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

	// Binfile is where to place the compiled driver binary, e.g. "fab.bin".
	Binfile string

	// DBFile is where to find the hash DB file, e.g. ".fab.db".
	DBFile string

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
	var args []string
	if m.Verbose {
		args = append(args, "-v")
	}
	if m.List {
		args = append(args, "-list")
	}
	if m.DBFile != "" {
		args = append(args, "-db", m.DBFile)
	}

	var compile bool

	info, err := os.Stat(m.Binfile)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if m.Verbose {
			fmt.Printf("Compiling %s\n", m.Binfile)
		}
		compile = true

	case err != nil:
		return errors.Wrapf(err, "statting %s", m.Binfile)

	case info.IsDir():
		return fmt.Errorf("%s is a directory", m.Binfile)

	case info.Mode().Perm()&1 == 0:
		return fmt.Errorf("file %s exists but is not world-executable", m.Binfile)

	case m.Force:
		if m.Verbose {
			fmt.Printf("Forcing recompilation of %s\n", m.Binfile)
		}
		compile = true

	default:
		if m.Verbose {
			fmt.Printf("Using existing %s\n", m.Binfile)
		}
	}

	if compile {
		if err := Compile(ctx, m.Pkgdir, m.Binfile); err != nil {
			return errors.Wrapf(err, "compiling %s", m.Binfile)
		}
		args = append(args, "-nocheck")
	}

	args = append(args, m.Args...)

	abs, err := filepath.Abs(m.Binfile)
	if err != nil {
		return errors.Wrapf(err, "computing absolute pathname for %s", m.Binfile)
	}
	cmd := exec.CommandContext(ctx, abs, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	return errors.Wrapf(err, "running %s %s", abs, strings.Join(args, " "))
}
