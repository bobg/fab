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
// Decls is implemented in terms of [fab.Files].
// Any opts are passed through to that function.
//
// It is JSON-encodable
// (and therefore usable as the subtarget in [fab.Files]).
//
// A Decls target may be specified in YAML using the tag !ts.Decls,
// which introduces a mapping whose fields are:
//
//   - Dir: the directory containing a Go package
//   - Type: the Go type to examine for producing TypeScript declarations
//   - Prefix: the path prefix for the generated POST URL
//   - Out: the output file
//   - Autoclean: a boolean indicating whether the output file should be added to the "autoclean registry."
//     See [fab.Autoclean] for more about this feature.
//
// Both Dir and Out are either absolute or relative to the directory containing the YAML file.
func Decls(dir, typename, prefix, outfile string, opts ...fab.FilesOpt) (fab.Target, error) {
	gopkg, err := golang.Deps(dir, false, false)
	if err != nil {
		return nil, errors.Wrapf(err, "getting deps for %s", dir)
	}
	subtarget := &declsType{
		Dir:      dir,
		Typename: typename,
		Prefix:   prefix,
		Outfile:  outfile,
	}

	return fab.Files(subtarget, gopkg, []string{outfile}, opts...), nil
}

// MustDecls is the same as [Decls] but panics on error.
func MustDecls(dir, typename, prefix, outfile string) fab.Target {
	target, err := Decls(dir, typename, prefix, outfile)
	if err != nil {
		panic(err)
	}
	return target
}

type declsType struct {
	Dir, Typename, Prefix, Outfile string
}

var _ fab.Target = &declsType{}

func (t *declsType) Run(ctx context.Context, _ *fab.Controller) error {
	if fab.GetDryRun(ctx) {
		return nil
	}
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

func declsDecoder(con *fab.Controller, node *yaml.Node, dir string) (fab.Target, error) {
	var d struct {
		Dir       string `yaml:"Dir"`
		Type      string `yaml:"Type"`
		Prefix    string `yaml:"Prefix"`
		Out       string `yaml:"Out"`
		Autoclean bool   `yaml:"Autoclean"`
	}
	if err := node.Decode(&d); err != nil {
		return nil, errors.Wrap(err, "YAML error decoding ts.Decls node")
	}

	return Decls(con.JoinPath(dir, d.Dir), d.Type, d.Prefix, con.JoinPath(dir, d.Out), fab.Autoclean(d.Autoclean))
}

func init() {
	fab.RegisterYAMLTarget("ts.Decls", declsDecoder)
}
