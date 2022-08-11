package fab

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// F is an adapter that turns a function into a Target.
// The Target's Run method is implemented by the function
// and its ID is random.
func F(f func(context.Context) error) Target {
	return &ftarget{f: f, id: RandID()}
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
	return f.id
}

// Command is a Target that invokes a command in a subprocess in its Run method.
type Command struct {
	// Cmd is the command to run.
	// It must be a full path,
	// or appear in a directory in the PATH environment variable.
	Cmd string

	// Args is the list of command-line arguments to pass to Cmd.
	Args []string

	// Dir is the directory in which to run the command;
	// the current directory by default.
	Dir string

	// Env is a list of environment variables to set while the command runs.
	// It adds to or replaces the values in the existing environment.
	Env []string

	// Prefix is an optional human-readable prefix for the Command's unique ID.
	// (The rest of the ID is auto-generated and random.)
	Prefix string

	// Verbose controls whether the Command runs verbosely.
	// If this is true, or Verbose(ctx) is true when Run is called,
	// then the subprocess's stdout and stderr are sent to os.Stdout and os.Stderr.
	Verbose bool

	id string
}

var _ Target = &Command{}

// Run implements Target.Run.
func (c *Command) Run(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.Cmd, c.Args...)
	cmd.Dir = Dir(ctx)
	cmd.Env = append(os.Environ(), c.Env...)

	var buf *bytes.Buffer
	if c.Verbose || Verbose(ctx) {
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
		c.id = RandID()
		if c.Prefix != "" {
			c.id = c.Prefix + "-" + c.id
		}
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
