package proto

import (
	"bufio"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
// otherOpts are options (other than -I / --proto_path options) for the protoc command line;
// and filesOpts are passed through to [fab.Files]
// (which this target is implemented in terms of).
//
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
//   - Autoclean: a boolean indicating whether the files listed in Outputs should be added to the "autoclean registry."
//     See [fab.Autoclean] for more about this feature.
func Proto(inputs, outputs, includes, otherOpts []string, filesOpts ...fab.FilesOpt) (fab.Target, error) {
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
	return fab.Files(&fab.Command{Cmd: "protoc", Args: args}, alldepsSlice, outputs, filesOpts...), nil
}

func protoDecoder(con *fab.Controller, node *yaml.Node, dir string) (fab.Target, error) {
	var p struct {
		Inputs    yaml.Node `yaml:"Inputs"`
		Outputs   yaml.Node `yaml:"Outputs"`
		Includes  yaml.Node `yaml:"Includes"`
		Opts      []string  `yaml:"Opts"`
		Autoclean bool      `yaml:"Autoclean"`
	}
	if err := node.Decode(&p); err != nil {
		return nil, errors.Wrap(err, "YAML error decoding proto.Proto node")
	}

	inputs, err := con.YAMLFileList(&p.Inputs, dir)
	if err != nil {
		return nil, errors.Wrap(err, "parsing protoc input files")
	}

	outputs, err := con.YAMLFileList(&p.Outputs, dir)
	if err != nil {
		return nil, errors.Wrap(err, "parsing protoc output files")
	}

	includes, err := con.YAMLFileList(&p.Includes, dir)
	if err != nil {
		return nil, errors.Wrap(err, "parsing protoc include list")
	}

	return Proto(inputs, outputs, includes, p.Opts, fab.Autoclean(p.Autoclean))
}

func init() {
	fab.RegisterYAMLTarget("proto.Proto", protoDecoder)
}

// Deps reads a protocol-buffer file and returns its list of dependencies.
// Included in the dependencies is the file itself,
// plus any files it imports
// (directly or indirectly)
// that can be found among the given include directories.
// The list is sorted for consistent, predictable results.
func Deps(filename string, includes []string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "opening %s", filename)
	}
	defer f.Close()

	result := set.New[string](filename)
	err = protodeps(f, includes, result)
	slice := result.Slice()
	sort.Strings(slice)
	return slice, err
}

var importRegex = regexp.MustCompile(`^import(\s+public)?\s*"([^"]+)"`)

func protodeps(r io.Reader, includes []string, result set.Of[string]) error {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		m := importRegex.FindStringSubmatch(sc.Text())
		if len(m) == 0 {
			continue
		}
		if err := protodepsImport(m[2], includes, result); err != nil {
			return err
		}
	}
	return sc.Err()
}

func protodepsImport(imp string, includes []string, result set.Of[string]) error {
	for _, inc := range includes {
		full := filepath.Join(inc, imp)

		if result.Has(full) {
			continue
		}

		f, err := os.Open(full)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return errors.Wrapf(err, "opening %s", full)
		}
		defer f.Close()

		result.Add(full)
		return protodeps(f, includes, result)
	}
	return nil
}

func protodepsDecoder(con *fab.Controller, node *yaml.Node, dir string) ([]string, error) {
	var pd struct {
		File     string   `yaml:"File"`
		Includes []string `yaml:"Includes"`
	}
	if err := node.Decode(&pd); err != nil {
		return nil, errors.Wrap(err, "YAML error in proto.Deps node")
	}
	return Deps(con.JoinPath(dir, pd.File), pd.Includes)
}

func init() {
	fab.RegisterYAMLStringList("proto.Deps", protodepsDecoder)
}
