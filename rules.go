package fab

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"

	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
)

// Named produces a Target with a give name prefix
// (the suffix is a session-unique number)
// that wraps another Target.
func Named(name string, target Target) Target {
	return &named{name: name, target: target}
}

type named struct {
	name   string
	target Target
	id     string
}

var _ Target = &named{}

func (n *named) Run(ctx context.Context) error {
	if GetVerbose(ctx) {
		fmt.Printf("Running %s\n", n.ID())
	}
	return Run(ctx, n.target)
}

func (n *named) ID() string {
	if n.id == "" {
		n.id = ID(n.name)
	}
	return n.id
}

// All produces a target that runs a collection of targets in parallel.
func All(targets ...Target) Target {
	return &all{targets: targets}
}

type all struct {
	targets []Target
	id      string
}

var _ Target = &all{}

func (a *all) Run(ctx context.Context) error {
	return Run(ctx, a.targets...)
}

func (a *all) ID() string {
	if a.id == "" {
		a.id = ID("All")
	}
	return a.id
}

// Seq produces a target that runs a collection of targets in sequence.
// It exits early when a target in the sequence fails.
func Seq(targets ...Target) Target {
	return &seq{targets: targets}
}

type seq struct {
	targets []Target
	id      string
}

var _ Target = &seq{}

func (s *seq) Run(ctx context.Context) error {
	for _, t := range s.targets {
		if err := Run(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func (s *seq) ID() string {
	if s.id == "" {
		s.id = ID("Seq")
	}
	return s.id
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
	return &ftarget{f: f}
}

type ftarget struct {
	f  func(context.Context) error
	id string
}

var _ Target = &ftarget{}

func (f *ftarget) Run(ctx context.Context) error {
	return f.f(ctx)
}

func (f *ftarget) ID() string {
	if f.id == "" {
		f.id = ID("F")
	}
	return f.id
}

// Command is a target whose Run function executes a command in a subprocess.
type Command struct {
	// Shell is the command to run,
	// as a single string with command name and arguments together.
	// It is parsed as if by a Unix shell,
	// with quoting and so on,
	// in order to produce the command name
	// and a list of individual argument strings.
	//
	// To bypass this parsing behavior,
	// you may specify Cmd and Args directly.
	Shell string

	// Cmd is the command to invoke,
	// either the path to a file,
	// or an executable file found in some directory
	// named in the PATH environment variable.
	//
	// Leave Cmd blank and specify Shell instead
	// to get shell-like parsing of a command and its arguments.
	Cmd string

	// Args is the list of command-line arguments
	// to pass to the command named in Cmd.
	Args []string

	// Dir is the directory in which to run the command.
	// The default is the value of GetDir(ctx) when the Run method is called.
	Dir string

	// Env is a list of VAR=VALUE strings to add to the environment when the command runs.
	Env []string

	// Verbose tells whether to connect the command's stdout and stderr
	// to this process's stdout and stderr.
	// If it's false,
	// the output is saved anyway in case the command fails.
	// If it does,
	// the returned error will be a CommandErr
	// containing the process's (combined) output.
	Verbose bool

	id string
}

var _ Target = &Command{}

func (c *Command) Run(ctx context.Context) error {
	cmdname, args, err := c.getCmdAndArgs()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, cmdname, args...)

	if c.Dir != "" {
		cmd.Dir = c.Dir
	} else {
		cmd.Dir = GetDir(ctx)
	}
	cmd.Env = append(os.Environ(), c.Env...)

	var buf *bytes.Buffer
	if c.Verbose {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	} else {
		buf = new(bytes.Buffer)
		cmd.Stdout, cmd.Stderr = buf, buf
	}

	err = cmd.Run()
	if err != nil && buf != nil {
		err = CommandErr{
			Err:    err,
			Output: buf.Bytes(),
		}
	}
	return err
}

func (c *Command) ID() string {
	if c.id == "" {
		c.id = ID("Command")
	}
	return c.id
}

func (c *Command) getCmdAndArgs() (string, []string, error) {
	if c.Cmd != "" {
		return c.Cmd, c.Args, nil
	}
	words, err := shellwords.Parse(c.Shell)
	if err != nil {
		return "", nil, err
	}
	if len(words) == 0 {
		return "", nil, fmt.Errorf("empty shell command")
	}
	return words[0], words[1:], nil
}

// CommandErr is a type of error that may be returned from Command.Run.
// If output was suppressed
// (because the Verbose field was false)
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
