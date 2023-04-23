package ts

import (
	"context"
	"os"

	"github.com/bobg/errors"
	"github.com/bobg/tsdecls"

	"github.com/bobg/fab"
	"github.com/bobg/fab/deps"
)

// Decls
func Decls(dir, typename, prefix, outfile string) (fab.Target, error) {
	gopkg, err := deps.Go(dir, false)
	if err != nil {
		return nil, errors.Wrapf(err, "getting deps for %s", dir)
	}
	obj := declsType{
		dir:      dir,
		typename: typename,
		prefix:   prefix,
		outfile:  outfile,
	}
	return fab.Files{
		Target: fab.F(obj.run),
		In:     gopkg,
		Out:    []string{outfile},
	}, nil
}

type declsType struct {
	dir, typename, prefix, outfile string
}

func (t declsType) run(context.Context) error {
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
