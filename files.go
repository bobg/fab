package fab

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/bobg/errors"
	json "github.com/gibson042/canonicaljson-go"
	"gopkg.in/yaml.v3"
)

// Files is a HashTarget.
// It contains a list of input files,
// and a list of expected output files.
// It also contains an embedded Target
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
// See the deps package for helper functions that can compute dependency lists of various kinds.
type Files struct {
	Target Target
	In     []string
	Out    []string
}

var _ HashTarget = Files{}

func (ft Files) Run(ctx context.Context) error {
	return ft.Target.Run(ctx)
}

func (ft Files) Name() string {
	return ft.Target.Name()
}

func (ft Files) SetName(name string) {
	ft.Target.SetName(name)
}

// Hash implements HashTarget.Hash.
func (ft Files) Hash(ctx context.Context) ([]byte, error) {
	var (
		inHashes  = make(map[string][]byte)
		outHashes = make(map[string][]byte)
	)
	err := fillWithFileHashes(ft.In, inHashes)
	if err != nil {
		return nil, errors.Wrapf(err, "computing input hash(es) for %s", ft.Name())
	}
	err = fillWithFileHashes(ft.Out, outHashes)
	if err != nil {
		return nil, errors.Wrapf(err, "computing output hash(es) for %s", ft.Name())
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

func fillWithFileHashes(files []string, hashes map[string][]byte) error {
	for _, file := range files {
		h, err := hashFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			h = nil
		} else if err != nil {
			return errors.Wrapf(err, "computing hash of %s", file)
		}
		hashes[file] = h
	}
	return nil
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
		In     []yaml.Node `yaml:"In"`
		Out    []string    `yaml:"Out"`
		Target yaml.Node   `yaml:"Target"`
	}
	if err := node.Decode(&yfiles); err != nil {
		return nil, errors.Wrap(err, "YAML error in Files node")
	}

	target, err := YAMLTarget(&yfiles.Target)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Target child of Files node")
	}

	var in []string
	for i, child := range yfiles.In {
		strs, err := YAMLStringList(&child)
		if err != nil {
			return nil, errors.Wrapf(err, "YAML error in Files.In node (index %d)", i)
		}
		in = append(in, strs...)
	}

	return Files{Target: target, In: in, Out: yfiles.Out}, nil
}

func init() {
	RegisterYAMLTarget("Files", filesDecoder)
}
