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

type Main struct {
	Pkgdir  string
	Binfile string
	DBFile  string
	Verbose bool
	Force   bool
	Args    []string
}

func (m Main) Run(ctx context.Context) error {
	var args []string
	if m.Verbose {
		args = append(args, "-v")
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
