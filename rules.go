package fab

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

func F(f func(context.Context) error) Target {
	return &ftarget{f: f}
}

type ftarget struct {
	once sync.Once
	f    func(context.Context) error
}

var _ Target = &ftarget{}

func (f *ftarget) Run(ctx context.Context) error {
	return f.f(ctx)
}

func (f *ftarget) Once() *sync.Once {
	return &f.once
}

type Command struct {
	Cmd     string
	Args    []string
	Dir     string
	Env     []string
	Verbose bool

	once sync.Once
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

func (c *Command) Once() *sync.Once {
	return &c.once
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
