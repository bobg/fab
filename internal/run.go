package internal

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/bobg/fab"
)

// Run uses Load to load the Go package in the given directory and turn it into an executable binary.
// It then runs the program as:
//
//   PROG [-v] CWD TMPFILE ARGS...
//
// where CWD is the current working directory,
// TMPFILE is the name of a temporary file where the program sends its output,
// and ARGS are the additional arguments passed to Run.
//
// Run parses the output in the temporary file:
// a JSON-encoded list of error strings.
// If the list is empty, Run returns nil.
// Otherwise it converts those strings to an error
// (using multierr.Combine if there are two or more)
// and returns it.
func Run(ctx context.Context, pkgdir string, args ...string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "getting working directory")
	}
	tmpfile, err := os.CreateTemp("", "fab")
	if err != nil {
		return errors.Wrap(err, "creating tempfile")
	}
	err = tmpfile.Close()
	if err != nil {
		return errors.Wrap(err, "closing tempfile")
	}
	defer os.Remove(tmpfile.Name())

	return Load(ctx, pkgdir, func(cmd *exec.Cmd) error {
		if fab.GetVerbose(ctx) {
			cmd.Args = append(cmd.Args, "-v")
		}
		cmd.Args = append(cmd.Args, cwd, tmpfile.Name())
		cmd.Args = append(cmd.Args, args...)

		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

		err := cmd.Run()
		if err != nil {
			return errors.Wrap(err, "running subprocess")
		}

		// Output from cmd is now in tmpfile.

		f, err := os.Open(tmpfile.Name())
		if err != nil {
			return errors.Wrapf(err, "opening tempfile")
		}
		defer f.Close()
		dec := json.NewDecoder(f)
		var errstrs []string
		err = dec.Decode(&errstrs)
		if err != nil {
			return errors.Wrap(err, "parsing subprocess output")
		}

		switch len(errstrs) {
		case 0:
			return nil
		case 1:
			return errors.New(errstrs[0])
		default:
			errs := make([]error, 0, len(errstrs))
			for _, e := range errstrs {
				errs = append(errs, errors.New(e))
			}
			return multierr.Combine(errs...)
		}
	})
}
