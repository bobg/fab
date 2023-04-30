package fab

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"sort"
	"sync"

	"github.com/bobg/errors"
	json "github.com/gibson042/canonicaljson-go"
	"gopkg.in/yaml.v3"
)

var (
	fileRegistryMu sync.Mutex
	fileRegistry   = make(map[string]*files)
)

// Files creates a target that contains a list of input files
// and a list of expected output files.
// It also contains a nested subtarget
// whose Execute method should produce or update the expected output files.
//
// When the Files target runs,
// a hash is computed from the nested subtarget
// and all the input and output files.
// If none of those has changed since the last time the output files were built,
// then the output files are up to date and running of this Files target can be skipped.
//
// The nested subtarget must be of a type that can be JSON-marshaled.
// Note that this excludes [F],
// among others.
//
// When a Files target runs,
// it checks to see whether any of its input files
// are listed as output files in other Files targets.
// Other targets found in this way are [Run] first, as prerequisites.
//
// The list of input files should mention every file where a change should cause a rebuild.
// Ideally this includes any files required by the nested subtarget
// plus any transitive dependencies.
// See the Deps function in the golang subpackage
// for an example of a function that can compute such a list for a Go package.
//
// A Files target may be specified in YAML using the !Files tag,
// which introduces a mapping whose fields are:
//
//   - Target: the nested subtarget, or target name
//   - In: the list of input files, interpreted with [YAMLStringList]
//   - Out: the list of output files, interpreted with [YAMLStringList]
//
// Example:
//
//	Foo: !Files
//	  Target: !Command
//	    - go build -o thingify ./cmd/thingify
//	  In: !golang.Deps
//	    Dir: cmd
//	  Out:
//	    - thingify
//
// This creates target Foo,
// which runs the given `go build` command
// to update the output file `thingify`
// when any files depended on by the Go package in `cmd` change.
func Files(target Target, in, out []string) Target {
	result := &files{
		Target: target,
		In:     in,
		Out:    out,
	}

	fileRegistryMu.Lock()
	for _, o := range out {
		fileRegistry[o] = result
	}
	fileRegistryMu.Unlock()

	return result
}

type files struct {
	Target Target
	In     []string
	Out    []string
}

var _ Target = &files{}

// Execute implements Target.Execute.
func (ft *files) Execute(ctx context.Context, con *Controller) error {
	if err := ft.runPrereqs(ctx, con); err != nil {
		return errors.Wrap(err, "in prerequisites")
	}

	if GetForce(ctx) {
		return con.Run(ctx, ft.Target)
	}

	db := GetHashDB(ctx)
	if db == nil {
		return con.Run(ctx, ft.Target)
	}

	h, err := ft.computeHash(ctx, con)
	if err != nil {
		return errors.Wrap(err, "computing hash before running subtarget")
	}
	has, err := db.Has(ctx, h)
	if err != nil {
		return errors.Wrap(err, "checking hash db")
	}
	if has {
		if GetVerbose(ctx) {
			con.Indentf("%s is up to date", con.Describe(ft))
		}
		return nil
	}

	if err = con.Run(ctx, ft.Target); err != nil {
		return errors.Wrap(err, "running subtarget")
	}

	h, err = ft.computeHash(ctx, con)
	if err != nil {
		return errors.Wrap(err, "computing hash after running subtarget")
	}
	err = db.Add(ctx, h)
	return errors.Wrap(err, "adding hash to db")
}

// Desc implements Target.Desc.
func (*files) Desc() string {
	return "Files"
}

func (ft *files) computeHash(ctx context.Context, con *Controller) ([]byte, error) {
	inHashes, err := fileHashes(ft.In)
	if err != nil {
		return nil, errors.Wrapf(err, "computing input hash(es) for %s", con.Describe(ft))
	}
	outHashes, err := fileHashes(ft.Out)
	if err != nil {
		return nil, errors.Wrapf(err, "computing output hash(es) for %s", con.Describe(ft))
	}
	tt := reflect.TypeOf(ft.Target)
	s := struct {
		Target     Target   `json:"target"`
		TargetType string   `json:"target_type"`
		In         []string `json:"in,omitempty"`  // [filename, hash, filename, hash, ...]
		Out        []string `json:"out,omitempty"` // [filename, hash, filename, hash, ...]
	}{
		Target:     ft.Target,
		TargetType: tt.String(),
		In:         inHashes,
		Out:        outHashes,
	}
	j, err := json.Marshal(s)
	if err != nil {
		return nil, errors.Wrap(err, "in JSON marshaling")
	}

	sum := sha256.Sum224(j)
	return sum[:], nil
}

func (ft *files) runPrereqs(ctx context.Context, con *Controller) error {
	var prereqs []Target

	fileRegistryMu.Lock()
	for _, in := range ft.In {
		if target, ok := fileRegistry[in]; ok {
			prereqs = append(prereqs, target)
		}
	}
	fileRegistryMu.Unlock()

	if len(prereqs) == 0 {
		return nil
	}
	return con.Run(ctx, prereqs...)
}

// Returns [filename, hash, filename, hash, ...],
// with filenames sorted.
func fileHashes(files []string) ([]string, error) {
	sorted := make([]string, len(files))
	copy(sorted, files)
	sort.Strings(sorted)

	result := make([]string, 0, 2*len(files))
	for _, file := range sorted {
		h, err := hashFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			h = ""
		} else if err != nil {
			return nil, errors.Wrapf(err, "computing hash of %s", file)
		}
		result = append(result, file, h)
	}

	return result, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "opening %s", path)
	}
	defer f.Close()
	hasher := sha256.New224()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return "", errors.Wrapf(err, "hashing %s", path)
	}
	h := hasher.Sum(nil)
	return hex.EncodeToString(h), nil
}

func filesDecoder(con *Controller, node *yaml.Node, dir string) (Target, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("got node kind %v, want %v", node.Kind, yaml.MappingNode)
	}

	var yfiles struct {
		In     yaml.Node `yaml:"In"`
		Out    yaml.Node `yaml:"Out"`
		Target yaml.Node `yaml:"Target"`
	}
	if err := node.Decode(&yfiles); err != nil {
		return nil, errors.Wrap(err, "YAML error in Files node")
	}

	target, err := con.YAMLTarget(&yfiles.Target, dir)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Target child of Files node")
	}

	in, err := YAMLFileList(&yfiles.In, dir)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Files.In node")
	}

	out, err := YAMLFileList(&yfiles.Out, dir)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Files.Out node")
	}

	return Files(target, in, out), nil
}

func init() {
	RegisterYAMLTarget("Files", filesDecoder)
}
