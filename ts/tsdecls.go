package ts

import (
	"context"
	"os"

	"github.com/bobg/errors"
	"github.com/bobg/tsdecls"
	"gopkg.in/yaml.v3"

	"github.com/bobg/fab"
	"github.com/bobg/fab/golang"
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
	gopkg, err := golang.Deps(dir, false)
	if err != nil {
		return nil, errors.Wrapf(err, "getting deps for %s", dir)
	}
	return &fab.Files{
		Target: &declsType{
			Dir:      dir,
			Typename: typename,
			Prefix:   prefix,
			Outfile:  outfile,
		},
		In:  gopkg,
		Out: []string{outfile},
	}, nil
}

type declsType struct {
	Dir, Typename, Prefix, Outfile string
}

var _ fab.Target = &declsType{}

func (t *declsType) Run(context.Context) error {
	f, err := os.Create(t.Outfile)
	if err != nil {
		return errors.Wrapf(err, "opening %s for writing", t.Outfile)
	}
	defer f.Close()

	if err = tsdecls.Write(f, t.Dir, t.Typename, t.Prefix); err != nil {
		return errors.Wrapf(err, "generating %s", t.Outfile)
	}
	return f.Close()
}

func (*declsType) Desc() string {
	return "ts.Decls"
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
