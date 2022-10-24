package rules

import (
	"sort"

	"github.com/bobg/go-generics/set"
	"github.com/bobg/go-generics/slices"
	"github.com/pkg/errors"

	"github.com/bobg/fab"
	"github.com/bobg/fab/deps"
)

// Proto produces a target that compiles protocol-buffer files using the "protoc" command.
// Inputs is a list of .proto input files;
// outputs is a list of the expected output files;
// includes is a list of directories in which to find .proto files;
// otherOpts are options (other than -I / --proto_path options) for the protoc command line.
// Typically otherOpts includes at least "--foo_out=DIR" for some target language foo.
// This function uses [deps.Proto] to find the dependencies of the input files.
func Proto(inputs, outputs, includes, otherOpts []string) (fab.Target, error) {
	alldeps := set.New[string](inputs...)
	for _, inp := range inputs {
		d, err := deps.Proto(inp, includes)
		if err != nil {
			return nil, errors.Wrapf(err, "computing dependencies for %s", inp)
		}
		alldeps.Add(d...)
	}

	alldepsSlice := alldeps.Slice()
	sort.Strings(alldepsSlice)

	args, _ := slices.Map(includes, func(_ int, inc string) (string, error) {
		return "-I" + inc, nil
	})
	args = append(args, otherOpts...)
	args = append(args, inputs...)
	return &fab.FilesTarget{
		Target: fab.Command("protoc", fab.CmdArgs(args...)),
		In:     alldepsSlice,
		Out:    outputs,
	}, nil
}
