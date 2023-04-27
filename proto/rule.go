package proto

import (
	"sort"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/set"
	"github.com/bobg/go-generics/v2/slices"
	"gopkg.in/yaml.v3"

	"github.com/bobg/fab"
)

// Proto produces a target that compiles protocol-buffer files using the "protoc" command.
// Inputs is a list of .proto input files;
// outputs is a list of the expected output files;
// includes is a list of directories in which to find .proto files;
// otherOpts are options (other than -I / --proto_path options) for the protoc command line.
// Typically otherOpts includes at least "--foo_out=DIR" for some target language foo.
// This function uses [Deps] to find the dependencies of the input files.
//
// A Proto target may be specified in YAML using the !proto.Proto tag,
// which introduces a mapping whose fields are:
//
//   - Inputs: the list of .proto input files
//   - Outputs: the list of expected output files
//   - Includes: the list of include directories
//   - Opts: the list of "other options" (see above) to pass to the protoc command line
func Proto(inputs, outputs, includes, otherOpts []string) (fab.Target, error) {
	alldeps := set.New[string](inputs...)
	for _, inp := range inputs {
		d, err := Deps(inp, includes)
		if err != nil {
			return nil, errors.Wrapf(err, "computing dependencies for %s", inp)
		}
		alldeps.Add(d...)
	}

	alldepsSlice := alldeps.Slice()
	sort.Strings(alldepsSlice)

	args := slices.Map(includes, func(inc string) string { return "-I" + inc })
	args = append(args, otherOpts...)
	args = append(args, inputs...)
	return fab.Files(&fab.Command{Cmd: "protoc", Args: args}, alldepsSlice, outputs), nil
}

func protoDecoder(node *yaml.Node) (fab.Target, error) {
	var p struct {
		Inputs   []string `yaml:"Inputs"`
		Outputs  []string `yaml:"Outputs"`
		Includes []string `yaml:"Includes"`
		Opts     []string `yaml:"Opts"`
	}
	if err := node.Decode(&p); err != nil {
		return nil, errors.Wrap(err, "YAML error decoding proto.Proto node")
	}
	return Proto(p.Inputs, p.Outputs, p.Includes, p.Opts)
}

func init() {
	fab.RegisterYAMLTarget("proto.Proto", protoDecoder)
}
