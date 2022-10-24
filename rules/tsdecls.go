package rules

import (
	"context"
	"os"

	"github.com/bobg/tsdecls"
	"github.com/pkg/errors"

	"github.com/bobg/fab"
	"github.com/bobg/fab/deps"
)

// Tsdecls
func Tsdecls(dir, typename, prefix, outfile string) (fab.Target, error) {
	gopkg, err := deps.Go(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "getting deps for %s", dir)
	}
	obj := tsdeclsType{
		dir:      dir,
		typename: typename,
		prefix:   prefix,
		outfile:  outfile,
	}
	return &fab.FilesTarget{
		Target: fab.F(obj.run),
		In:     gopkg,
		Out:    []string{outfile},
	}, nil
}

type tsdeclsType struct {
	dir, typename, prefix, outfile string
}

func (t tsdeclsType) run(context.Context) error {
	f, err := os.Create(t.outfile)
	if err != nil {
		return errors.Wrapf(err, "opening %s for writing", t.outfile)
	}
	defer f.Close()

	if err = tsdecls.Write(f, t.dir, t.typename, t.prefix); err != nil {
		return errors.Wrapf(err, "generating %s", t.outfile)
	}
	return f.Close()
}
