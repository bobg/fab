package ts

import (
	"context"
	"os"

	"github.com/bobg/errors"
	"github.com/bobg/tsdecls"
	"gopkg.in/yaml.v3"

	"github.com/bobg/fab"
	"github.com/bobg/fab/deps"
)

// Decls uses [tsdecls.Write]
// to write TypeScript type declarations to `outfile`
// based on the Go `typename`
// found in the package in `dir`.
//
// A Decls target may be specified in YAML using the tag !ts.Decls,
// which introduces a mapping whose fields are:
//
//   - Dir: the directory containing a Go package
//   - Type: the Go type to examine for producing TypeScript declarations
//   - Prefix: the path prefix for the generated POST URL
//   - Out: the output file
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

func declsDecoder(node *yaml.Node) (fab.Target, error) {
	var d struct {
		Dir    string `yaml:"Dir"`
		Type   string `yaml:"Type"`
		Prefix string `yaml:"Prefix"`
		Out    string `yaml:"Out"`
	}
	if err := node.Decode(&d); err != nil {
		return nil, errors.Wrap(err, "YAML error decoding ts.Decls node")
	}
	return Decls(d.Dir, d.Type, d.Prefix, d.Out)
}

func init() {
	fab.RegisterYAMLTarget("ts.Decls", declsDecoder)
}
