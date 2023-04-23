package fab

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/bobg/errors"
	"github.com/mattn/go-shellwords"
	"gopkg.in/yaml.v3"
)

// Command is a target whose Run function executes a command in a subprocess.
//
// If `CmdArgs` appears among the options,
// then `cmd` is the name of a command to run
// and its arguments are given by the `CmdArgs` option.
// Otherwise `cmd` is the complete command
// and is parsed as if by a Unix shell,
// with quoting and so on
// (but not tilde escapes or backtick substitution etc.)
// in order to produce the command name
// and argument list.
func Command(cmd string, opts ...CommandOpt) Target {
	c := &command{
		Namer: NewNamer("Command"),
		Shell: cmd,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type command struct {
	*Namer

	// Shell is the command to run,
	// as a single string with command name and arguments together.
	// It is parsed as if by a Unix shell,
	// with quoting and so on,
	// in order to produce the command name
	// and a list of individual argument strings.
	//
	// To bypass this parsing behavior,
	// you may specify Cmd and Args directly.
	Shell string `json:"shell,omitempty"`

	// Cmd is the command to invoke,
	// either the path to a file,
	// or an executable file found in some directory
	// named in the PATH environment variable.
	//
	// Leave Cmd blank and specify Shell instead
	// to get shell-like parsing of a command and its arguments.
	Cmd string `json:"cmd,omitempty"`

	// Args is the list of command-line arguments
	// to pass to the command named in Cmd.
	Args []string `json:"args,omitempty"`

	// Stdout and Stderr tell where to send the command's output.
	// If either or both is nil,
	// that output is saved in case the subprocess encounters an error.
	// Then the returned error is a CommandErr containing that output.
	Stdout io.Writer `json:"-"`
	Stderr io.Writer `json:"-"`

	// StdoutFile is the name of a file to which the command's standard output should go.
	// It is mutually exclusive with Stdout.
	// When the command runs, the file is created or overwritten.
	StdoutFile string `json:"stdout_file,omitempty"`

	// Dir is the directory in which to run the command.
	// The default is the value of GetDir(ctx) when the Run method is called.
	Dir string `json:"dir,omitempty"`

	// Env is a list of VAR=VALUE strings to add to the environment when the command runs.
	Env []string `json:"env,omitempty"`
}

var _ Target = &command{}

// CommandOpt is the type of an option to [Command].
type CommandOpt func(*command)

// CmdArgs sets the arguments for the command to run.
// When this option is used,
// the string passed to Command is used as argument 0
// (i.e., the command name).
func CmdArgs(args ...string) CommandOpt {
	return func(c *command) {
		c.Cmd = c.Shell
		c.Args = args
	}
}

// CmdStdout sets the stdout for the command.
func CmdStdout(w io.Writer) CommandOpt {
	return func(c *command) {
		c.Stdout = w
	}
}

// CmdStdoutFile sets a filename for the command's standard output.
// The file is created or overwritten when the command runs.
func CmdStdoutFile(name string) CommandOpt {
	return func(c *command) {
		c.StdoutFile = name
	}
}

// CmdStderr sets the stderr for the command.
func CmdStderr(w io.Writer) CommandOpt {
	return func(c *command) {
		c.Stderr = w
	}
}

// CmdDir sets the working directory for the command.
func CmdDir(dir string) CommandOpt {
	return func(c *command) {
		c.Dir = dir
	}
}

// CmdEnv adds to the environment variables for the command.
func CmdEnv(env []string) CommandOpt {
	return func(c *command) {
		c.Env = env
	}
}

// Run implements Target.Run.
func (c *command) Run(ctx context.Context) error {
	cmdname, args, err := c.getCmdAndArgs()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, cmdname, args...)

	cmd.Dir = c.Dir
	cmd.Env = append(os.Environ(), c.Env...)

	cmd.Stdout, cmd.Stderr = c.Stdout, c.Stderr
	if c.StdoutFile != "" {
		f, err := os.Create(c.StdoutFile)
		if err != nil {
			return errors.Wrapf(err, "opening %s for writing", c.StdoutFile)
		}
		defer f.Close()
		cmd.Stdout = f
	}

	var buf bytes.Buffer
	if cmd.Stdout == nil {
		cmd.Stdout = &buf
	}
	if cmd.Stderr == nil {
		cmd.Stderr = &buf
	}

	if GetVerbose(ctx) {
		Indentf(ctx, "  Running command %s", cmd)
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

func (c *command) getCmdAndArgs() (string, []string, error) {
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

// CommandErr is a type of error that may be returned from command.Run.
// If the command's Stdout or Stderr field was nil,
// then that output from the subprocess is in CommandErr.Output
// and the underlying error is in CommandErr.Err.
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

func commandDecoder(node *yaml.Node) (Target, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("got node kind %v, want %v", node.Kind, yaml.SequenceNode)
	}
	if len(node.Content) == 0 {
		return nil, fmt.Errorf("no child nodes")
	}
	cmdnode := node.Content[0]
	if cmdnode.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("got Command child node kind %v, want %v", cmdnode.Kind, yaml.ScalarNode)
	}
	cmd := cmdnode.Value

	var opts []CommandOpt
	for i := 1; i < len(node.Content); i++ {
		opt, err := commandOptDecoder(node.Content[i])
		if err != nil {
			return nil, errors.Wrapf(err, "YAML error in Command-node option (child %d)", i)
		}
		opts = append(opts, opt)
	}

	return Command(cmd, opts...), nil
}

func commandOptDecoder(node *yaml.Node) (CommandOpt, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Value {
		case "stdout":
			return CmdStdout(os.Stdout), nil
		case "stderr":
			return CmdStderr(os.Stderr), nil
		default:
			// TODO: implement others
			return nil, fmt.Errorf("unknown command option %s", node.Value)
		}

	default:
		// TODO: implement others
		return nil, fmt.Errorf("unknown command option node kind %v", node.Kind)
	}
}

func init() {
	RegisterYAMLTarget("Command", commandDecoder)
}
