package fab

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"io/fs"
	"os"

	"github.com/pkg/errors"
)

// All produces a target that runs a collection of targets in parallel.
func All(targets ...Target) Target {
	return &all{Namer: NewNamer("all"), targets: targets}
}

type all struct {
	*Namer
	targets []Target
}

var _ Target = &all{}

// Run implements Target.Run.
func (a *all) Run(ctx context.Context) error {
	return Run(ctx, a.targets...)
}

// Seq produces a target that runs a collection of targets in sequence.
// Its Run method exits early when a target in the sequence fails.
func Seq(targets ...Target) Target {
	return &seq{Namer: NewNamer("seq"), targets: targets}
}

type seq struct {
	*Namer
	targets []Target
}

var _ Target = &seq{}

// Run implements Target.Run.
func (s *seq) Run(ctx context.Context) error {
	for _, t := range s.targets {
		if err := Run(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

// Deps wraps a target with a set of dependencies,
// making sure those run first.
//
// It is equivalent to Seq(All(depTargets...), target).
func Deps(target Target, depTargets ...Target) Target {
	return Seq(All(depTargets...), target)
}

// F produces a target whose Run function invokes the given function.
func F(f func(context.Context) error) Target {
	return &ftarget{Namer: NewNamer("f"), f: f}
}

type ftarget struct {
	*Namer
	f func(context.Context) error
}

var _ Target = &ftarget{}

// Run implements Target.Run.
func (f *ftarget) Run(ctx context.Context) error {
	return f.f(ctx)
}

// FilesTarget is a HashTarget.
// It contains a list of input files,
// and a list of expected output files.
// It also contains an embedded Target
// whose Run method should produce the expected output files.
//
// The FilesTarget's hash is computed from the target and all the input and output files.
// If none of those have changed since the last time the output files were built,
// then the output files are up to date and running of this FilesTarget can be skipped.
//
// The Target must be of a type that can be JSON-marshaled.
//
// The In list should mention every file where a change should cause a rebuild.
// Ideally this includes any files required by the Target's Run method,
// plus any transitive dependencies.
// See the deps package for helper functions that can compute dependency lists of various kinds.
type FilesTarget struct {
	Target
	In  []string
	Out []string
}

var _ HashTarget = FilesTarget{}

// Hash implements HashTarget.Hash.
func (ft FilesTarget) Hash(ctx context.Context) ([]byte, error) {
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

// Clean is a Target that deletes the files named in Files when it runs.
// Files that don't exist are silently ignored.
func Clean(files ...string) Target {
	return &clean{
		Namer: NewNamer("clean"),
		Files: files,
	}
}

type clean struct {
	*Namer
	Files []string
}

// Run implements Target.Run.
func (c *clean) Run(_ context.Context) error {
	for _, f := range c.Files {
		err := os.Remove(f)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return errors.Wrapf(err, "removing %s", f)
		}
	}
	return nil
}
