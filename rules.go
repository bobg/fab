package fab

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// F is an adapter that turns a function into a Target.
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

type Command struct {
	Cmd      string
	Args     []string
	Dir      string
	Env      []string
	Verbose  bool
	IDPrefix string

	id string
}

var _ Target = &Command{}

func (c *Command) Run(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.Cmd, c.Args...)
	cmd.Dir = c.Dir
	cmd.Env = append(os.Environ(), c.Env...)

	var buf *bytes.Buffer
	if c.Verbose {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
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

func (c *Command) ID() string {
	if c.id == "" {
		c.id = RandID()
		if c.IDPrefix != "" {
			c.id = c.IDPrefix + "-" + c.id
		}
	}
	return c.id
}

type CommandErr struct {
	Err    error
	Output []byte
}

func (e CommandErr) Error() string {
	return fmt.Sprintf("%s; output follows\n%s", e.Err, string(e.Output))
}

func (e CommandErr) Unwrap() error {
	return e.Err
}
