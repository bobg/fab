package fab

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"

	json "github.com/gibson042/canonicaljson-go"
	"github.com/pkg/errors"
)

// F is an adapter that turns a function into a Target.
// The Target's Run method invokes the function.
// The target's ID is F-<number> by default.
func F(f func(context.Context) error, opts ...FOpt) Target {
	result := &ftarget{
		f:  f,
		id: ID("F"),
	}
	for _, opt := range opts {
		opt(result)
	}
	return result
}

type ftarget struct {
	f    func(context.Context) error
	deps []Target
	id   string
}

// FOpt is the type of an option passed to F.
type FOpt func(*ftarget)

// FPrefix changes the target's ID prefix from the default of "F".
func FPrefix(prefix string) FOpt {
	return func(f *ftarget) {
		f.id = ID(prefix)
	}
}

// FDeps adds dependencies to the target.
// The target's Run method will ensure that the dependencies run
// before the target's function.
func FDeps(deps ...Target) FOpt {
	return func(f *ftarget) {
		f.deps = append(f.deps, deps...)
	}
}

var _ Target = &ftarget{}

func (f *ftarget) Run(ctx context.Context) error {
	if len(f.deps) > 0 {
		err := Run(ctx, f.deps...)
		if err != nil {
			return err
		}
	}
	return f.f(ctx)
}

func (f *ftarget) ID() string {
	return f.id
}

// Command is a Target that invokes a command in a subprocess in its Run method.
type Command struct {
	// Cmd is the command to run.
	// It must be a full path,
	// or appear in a directory in the PATH environment variable.
	Cmd string `json:"cmd"`

	// Args is the list of command-line arguments to pass to Cmd.
	Args []string `json:"args,omitempty"`

	// Dir is the directory in which to run the command;
	// the current directory by default.
	Dir string `json:"dir,omitempty"`

	// Env is a list of environment variables to set while the command runs.
	// It adds to or replaces the values in the existing environment.
	Env []string `json:"env,omitempty"`

	// Prefix is an optional human-readable prefix for the Command's unique ID.
	// (The rest of the ID is auto-generated and random.)
	Prefix string `json:"prefix,omitempty"`

	// Verbose controls whether the Command runs verbosely.
	// If this is true, or Verbose(ctx) is true when Run is called,
	// then the subprocess's stdout and stderr are sent to os.Stdout and os.Stderr.
	Verbose bool `json:"-"`

	id string
}

var _ Target = &Command{}

// Run implements Target.Run.
func (c *Command) Run(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.Cmd, c.Args...)
	cmd.Dir = GetDir(ctx)
	cmd.Env = append(os.Environ(), c.Env...)

	var buf *bytes.Buffer
	if c.Verbose || GetVerbose(ctx) {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		fmt.Printf("Running %s\n", c.ID())
	} else {
		buf = new(bytes.Buffer)
		cmd.Stdout, cmd.Stderr = buf, buf
	}

	err := cmd.Run()
	if err != nil && buf != nil {
		err = CommandErr{
			Err:    err,
			Output: buf.Bytes(),
		}
	}
	return err
}

// ID implements Target.ID.
func (c *Command) ID() string {
	if c.id == "" {
		prefix := c.Prefix
		if prefix == "" {
			prefix = "Command"
		}
		c.id = ID(prefix)
	}
	return c.id
}

// CommandErr is a type of error that may be returned from Command.Run.
// If output was suppressed
// (because Command.Verbose and Verbose(ctx) were both false)
// this contains both the underlying error and the subprocess's combined output.
type CommandErr struct {
	Err    error
	Output []byte
}

// Error implements error.Error.
func (e CommandErr) Error() string {
	return fmt.Sprintf("%s; output follows\n%s", e.Err, string(e.Output))
}

// Unwrap produces the underlying error.
func (e CommandErr) Unwrap() error {
	return e.Err
}

// FilesCommand is a HashTarget.
// It contains a Command,
// a list of input files,
// and a list of expected output files.
type FilesCommand struct {
	*Command
	In  []string
	Out []string
}

var _ HashTarget = FilesCommand{}

// Hash implements HashTarget.
func (fc FilesCommand) Hash(_ context.Context) ([]byte, error) {
	var (
		inHashes  = make(map[string][]byte)
		outHashes = make(map[string][]byte)
	)
	err := fillWithFileHashes(fc.In, inHashes)
	if err != nil {
		return nil, errors.Wrapf(err, "computing input hash(es) for %s", fc.ID())
	}
	err = fillWithFileHashes(fc.Out, outHashes)
	if err != nil {
		return nil, errors.Wrapf(err, "computing output hash(es) for %s", fc.ID())
	}
	s := struct {
		*Command
		In  map[string][]byte `json:"in,omitempty"`
		Out map[string][]byte `json:"out,omitempty"`
	}{
		Command: fc.Command,
		In:      inHashes,
		Out:     outHashes,
	}
	j, err := json.Marshal(s)
	if err != nil {
		return nil, errors.Wrap(err, "in JSON marshaling")
	}
	sum := sha256.Sum256(j)
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
	hasher := sha256.New()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return nil, errors.Wrapf(err, "hashing %s", path)
	}
	return hasher.Sum(nil), nil
}
