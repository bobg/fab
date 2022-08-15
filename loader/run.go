package loader

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/bobg/fab"
)

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

	return Load(ctx, pkgdir, func(dir string) error {
		prog := filepath.Join(dir, "x")

		var allargs []string
		if fab.GetVerbose(ctx) {
			allargs = append(allargs, "-v")
		}
		allargs = append(allargs, cwd, tmpfile.Name())
		allargs = append(allargs, args...)

		cmd := exec.CommandContext(ctx, prog, allargs...)
		cmd.Dir = dir
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

		err := cmd.Run()
		if err != nil {
			return errors.Wrap(err, "running subprocess")
		}
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
