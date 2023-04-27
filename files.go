package fab

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"

	"github.com/bobg/errors"
	json "github.com/gibson042/canonicaljson-go"
	"gopkg.in/yaml.v3"
)

var (
	fileRegistryMu sync.Mutex
	fileRegistry   = make(map[string]*files)
)

// Files is a target that contains a list of input files
// and a list of expected output files.
// It also contains a nested subtarget
// whose Run method should produce the expected output files.
//
// The Files target's hash is computed from the target and all the input and output files.
// If none of those have changed since the last time the output files were built,
// then the output files are up to date and running of this Files target can be skipped.
//
// The Target must be of a type that can be JSON-marshaled.
//
// The In list should mention every file where a change should cause a rebuild.
// Ideally this includes any files required by the Target's Run method,
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

// Run implements Target.Run.
func (ft *files) Run(ctx context.Context) error {
	if err := ft.runPrereqs(ctx); err != nil {
		return errors.Wrap(err, "in prerequisites")
	}

	if GetForce(ctx) {
		return Run(ctx, ft.Target)
	}

	db := GetHashDB(ctx)
	if db == nil {
		return Run(ctx, ft.Target)
	}

	h, err := ft.computeHash(ctx)
	if err != nil {
		return errors.Wrap(err, "computing hash before running subtarget")
	}
	has, err := db.Has(ctx, h)
	if err != nil {
		return errors.Wrap(err, "checking hash db")
	}
	if has {
		if GetVerbose(ctx) {
			Indentf(ctx, "%s is up to date", Describe(ft))
		}
		return nil
	}

	if err = Run(ctx, ft.Target); err != nil {
		return errors.Wrap(err, "running subtarget")
	}

	h, err = ft.computeHash(ctx)
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

// TODO: should this incorporate debug.ReadBuildInfo?
func (ft *files) computeHash(ctx context.Context) ([]byte, error) {
	inHashes, err := fileHashes(ft.In)
	if err != nil {
		return nil, errors.Wrapf(err, "computing input hash(es) for %s", Describe(ft))
	}
	outHashes, err := fileHashes(ft.Out)
	if err != nil {
		return nil, errors.Wrapf(err, "computing output hash(es) for %s", Describe(ft))
	}
	s := struct {
		Target
		In  map[string][]byte `json:"in,omitempty"`
		Out map[string][]byte `json:"out,omitempty"`
	}{
		Target: ft.Target,
		In:     inHashes,
		Out:    outHashes,
	}
	j, err := json.Marshal(s)
	if err != nil {
		return nil, errors.Wrap(err, "in JSON marshaling")
	}
	sum := sha256.Sum224(j)
	return sum[:], nil
}

func (ft *files) runPrereqs(ctx context.Context) error {
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
	return Run(ctx, prereqs...)
}

func fileHashes(files []string) (map[string][]byte, error) {
	hashes := make(map[string][]byte)
	for _, file := range files {
		h, err := hashFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			h = nil
		} else if err != nil {
			return nil, errors.Wrapf(err, "computing hash of %s", file)
		}
		hashes[file] = h
	}
	return hashes, nil
}

func hashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "opening %s", path)
	}
	defer f.Close()
	hasher := sha256.New224()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return nil, errors.Wrapf(err, "hashing %s", path)
	}
	return hasher.Sum(nil), nil
}

func filesDecoder(node *yaml.Node) (Target, error) {
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

	target, err := YAMLTarget(&yfiles.Target)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Target child of Files node")
	}

	in, err := YAMLStringList(&yfiles.In)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Files.In node")
	}

	out, err := YAMLStringList(&yfiles.Out)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Files.Out node")
	}

	return Files(target, in, out), nil
}

func init() {
	RegisterYAMLTarget("Files", filesDecoder)
}
