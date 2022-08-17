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

func Deps(target Target, depTargets ...Target) Target {
	return &deps{target: target, deps: depTargets}
}

type deps struct {
	target Target
	deps   []Target
	id     string
}

var _ Target = &deps{}

func (d *deps) Run(ctx context.Context) error {
	if err := Run(ctx, d.deps...); err != nil {
		return err
	}
	return Run(ctx, d.target)
}

func (d *deps) ID() string {
	if d.id == "" {
		d.id = ID("Deps")
	}
	return d.id
}

type Func struct {
	F  func(context.Context) error
	id string
}

var _ Target = &Func{}

func (f *Func) Run(ctx context.Context) error {
	return f.F(ctx)
}

func (f *Func) ID() string {
	if f.id == "" {
		f.id = ID("Func")
	}
	return f.id
}

type Command struct {
	// Shell is parsed into shell words to produce a command and args.
	// It is mutually exclusive with Cmd+Args.
	Shell string

	Cmd  string
	Args []string

	Stdout io.Writer
	Stderr io.Writer

	Dir string
	Env []string

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

	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = c.Stdout, c.Stderr
	if c.Stdout == nil {
		c.Stdout = &buf
	}
	if c.Stderr == nil {
		c.Stderr = &buf
	}

	err = cmd.Run()
	if err != nil && buf.Len() > 0 {
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
