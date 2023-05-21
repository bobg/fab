package fab

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/slices"
	"gopkg.in/yaml.v3"
)

// Command is a Target whose Run function executes a command in a subprocess.
//
// It is JSON-encodable
// (and therefore usable as the subtarget in [Files]).
//
// A Command target may be specified in YAML using the !Command tag,
// which introduces a mapping with the following fields:
//
//   - Shell, the command string to execute with $SHELL,
//     mutually exclusive with Cmd.
//   - Cmd, an executable command invoked with Args as its arguments,
//     mutually exclusive with Shell.
//   - Args, list of arguments for Cmd.
//   - Stdin, the name of a file from which the command's standard input should be read,
//     or the special string $stdin to mean read Fab's standard input.
//   - Stdout, the name of a file to which the command's standard output should be written,
//     either absolute or relative to the directory in which the YAML file is found.
//     The file is overwritten unless this is prefixed with >> which means append.
//     This may also be one of these special strings:
//     $stdout (copy the command's output to Fab's standard output);
//     $stderr (copy the command's output to Fab's standard error);
//     $indent (indent the command's output with [IndentingCopier] and copy it to Fab's standard output);
//     $verbose (like $indent, but produce output only when fab is running in verbose mode [with the -v flag]);
//     $discard (discard the command's output).
//   - Stderr, the name of a file to which the command's standard error should be written,
//     either absolute or relative to the directory in which the YAML file is found.
//     The file is overwritten unless this is prefixed with >> which means append.
//     This may also be one of these special strings:
//     $stdout (copy the command's error output to Fab's standard error);
//     $stderr (copy the command's error output to Fab's standard error);
//     $indent (indent the command's error output with [IndentingCopier] and copy it to Fab's standard error);
//     $verbose (like $indent, but produce output only when fab is running in verbose mode [with the -v flag]);
//     $discard (discard the command's error output).
//   - Dir, the directory in which the command should run,
//     either absolute or relative to the directory in which the YAML file is found.
//   - Env, a list of VAR=VALUE strings to add to the command's environment.
//
// As a special case,
// a !Command whose shell is a list instead of a single string
// will produce a [Seq] of Commands,
// one for each of the Shell strings.
// The Commands in the Seq are otherwise identical,
// with one further special case:
// if Stdout and/or Stderr refers to a file,
// then the second and subsequent Commands in the Seq
// will always append to the file rather than overwrite it,
// even without the >> prefix.
// (If you really do want some command in the sequence to overwrite a file,
// you can always add >FILE to the Shell string.)
type Command struct {
	// Shell is the command to run,
	// as a single string with command name and arguments together.
	// It is invoked with $SHELL -c,
	// with $SHELL defaulting to /bin/sh.
	//
	// If you prefer to specify a command that is not executed by a shell,
	// leave Shell blank and fill in Cmd and Args instead.
	//
	// To bypass this parsing behavior,
	// you may specify Cmd and Args directly.
	Shell string `json:"shell,omitempty"`

	// Cmd is the command to invoke,
	// either the path to a file,
	// or an executable file found in some directory
	// named in the PATH environment variable.
	//
	// If you need your command string to be parsed by a shell,
	// leave Cmd and Args blank and specify Shell instead.
	Cmd string `json:"cmd,omitempty"`

	// Args is the list of command-line arguments
	// to pass to the command named in Cmd.
	Args []string `json:"args,omitempty"`

	// Stdout tells where to send the command's output.
	// When no output destination is specified,
	// the default depends on whether Fab is running in verbose mode
	// (i.e., if [GetVerbose] returns true).
	// In verbose mode,
	// the command's output is indented and copied to Fab's standard output
	// (using [IndentingCopier]).
	// Otherwise,
	// the command's output is captured
	// and bundled together with any error into a [CommandErr].
	//
	// Stdout, StdoutFile, and StdoutFn are all mutually exclusive.
	Stdout io.Writer `json:"-"`

	// Stderr tells where to send the command's error output.
	// When no error-output destination is specified,
	// the default depends on whether Fab is running in verbose mode
	// (i.e., if [GetVerbose] returns true).
	// In verbose mode,
	// the command's error output is indented and copied to Fab's standard error
	// (using [IndentingCopier]).
	// Otherwise,
	// the command's error output is captured
	// and bundled together with any error into a [CommandErr].
	//
	// Stderr, StderrFile, and StderrFn are all mutually exclusive.
	Stderr io.Writer `json:"-"`

	// StdoutFn lets you defer assigning a value to Stdout
	// until Run is invoked,
	// at which time this function is called with the context and the [Controller]
	// to produce the [io.Writer] to use.
	// If the writer produced by this function is also an [io.Closer],
	// its Close method will be called before Run exits.
	//
	// Stdout, StdoutFile, and StdoutFn are all mutually exclusive.
	StdoutFn func(context.Context, *Controller) io.Writer `json:"-"`

	// StderrFn lets you defer assigning a value to Stderr
	// until Run is invoked,
	// at which time this function is called with the context and the [Controller]
	// to produce the [io.Writer] to use.
	// If the writer produced by this function is also an [io.Closer],
	// its Close method will be called before Run exits.
	//
	// Stderr, StderrFile, and StderrFn are all mutually exclusive.
	StderrFn func(context.Context, *Controller) io.Writer `json:"-"`

	// StdoutFile is the name of a file to which the command's standard output should go.
	// When the command runs,
	// the file is created or overwritten,
	// unless this string has a >> prefix,
	// which means "append."
	// If StdoutFile and StderrFile name the same file,
	// output from both streams is combined there.
	//
	// Stdout, StdoutFile, and StdoutFn are all mutually exclusive.
	StdoutFile string `json:"stdout_file,omitempty"`

	// StderrFile is the name of a file to which the command's standard error should go.
	// When the command runs,
	// the file is created or overwritten,
	// unless this string has a >> prefix,
	// which means "append."
	// If StdoutFile and StderrFile name the same file,
	// output from both streams is combined there.
	//
	// Stderr, StderrFile, and StderrFn are all mutually exclusive.
	StderrFile string `json:"stderr_file,omitempty"`

	// Stdin tells where to read the command's standard input.
	Stdin io.Reader `json:"-"`

	// StdinFile is the name of a file from which the command should read its standard input.
	// It is mutually exclusive with Stdin.
	// It is an error for the file not to exist when the command runs.
	StdinFile string `json:"stdin_file,omitempty"`

	// Dir is the directory in which to run the command.
	Dir string `json:"dir,omitempty"`

	// Env is a list of VAR=VALUE strings to add to the environment when the command runs.
	Env []string `json:"env,omitempty"`
}

var _ Target = &Command{}

// Shellf is a convenience routine that produces a *Command
// whose Shell field is initialized by processing `format` and `args` with [fmt.Sprintf].
func Shellf(format string, args ...any) *Command {
	return &Command{
		Shell: fmt.Sprintf(format, args...),
	}
}

// Run implements Target.Run.
func (c *Command) Run(ctx context.Context, con *Controller) (err error) {
	var (
		cmdname = c.Cmd
		args    = c.Args
	)
	if cmdname == "" {
		if cmdname = os.Getenv("SHELL"); cmdname == "" {
			cmdname = "/bin/sh"
		}
		args = []string{"-c", c.Shell}
	}

	cmd := exec.CommandContext(ctx, cmdname, args...)

	cmd.Dir = c.Dir
	cmd.Env = append(os.Environ(), c.Env...)

	if GetDryRun(ctx) {
		if GetVerbose(ctx) {
			con.Indentf("  Would run command %s", cmd)
		}
		return nil
	}

	cmd.Stdout, cmd.Stderr = c.Stdout, c.Stderr

	var (
		stdoutFile   = c.StdoutFile
		stderrFile   = c.StderrFile
		stdoutAppend = strings.HasPrefix(stdoutFile, ">>")
		stderrAppend = strings.HasPrefix(stderrFile, ">>")
	)

	if stdoutAppend {
		stdoutFile = strings.TrimLeft(stdoutFile, "> ")
	}
	if stderrAppend {
		stderrFile = strings.TrimLeft(stderrFile, "> ")
	}

	if stdoutFile == stderrFile && stdoutAppend != stderrAppend {
		return fmt.Errorf("stdout and stderr name the same file but disagree about append vs. overwrite")
	}

	if stdoutFile != "" {
		if stdoutAppend {
			f, err := os.OpenFile(stdoutFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			if err != nil {
				return errors.Wrapf(err, "opening %s for appending", stdoutFile)
			}
			defer func() {
				closeErr := f.Close()
				if err == nil {
					err = errors.Wrapf(closeErr, "closing stdout file %s", stdoutFile)
				}
			}()
			cmd.Stdout = f
		} else {
			f, err := os.Create(stdoutFile)
			if err != nil {
				return errors.Wrapf(err, "opening %s for writing", stdoutFile)
			}
			defer func() {
				closeErr := f.Close()
				if err == nil {
					err = errors.Wrapf(closeErr, "closing stderr file %s", stdoutFile)
				}
			}()
			cmd.Stdout = f
		}
	}

	if stderrFile != "" {
		switch {
		case stdoutFile == stderrFile:
			cmd.Stderr = cmd.Stdout

		case stderrAppend:
			f, err := os.OpenFile(stderrFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			if err != nil {
				return errors.Wrapf(err, "opening %s for appending", stderrFile)
			}
			defer f.Close()
			cmd.Stderr = f

		default:
			f, err := os.Create(stderrFile)
			if err != nil {
				return errors.Wrapf(err, "opening %s for writing", stderrFile)
			}
			defer f.Close()
			cmd.Stderr = f
		}
	}

	if cmd.Stdout == nil && c.StdoutFn != nil {
		if w := c.StdoutFn(ctx, con); w != nil {
			if closer, ok := w.(io.Closer); ok {
				defer func() {
					closeErr := closer.Close()
					if err == nil {
						err = errors.Wrap(closeErr, "closing stdout")
					}
				}()
			}
			cmd.Stdout = w
		}
	}
	if cmd.Stderr == nil && c.StderrFn != nil {
		if w := c.StderrFn(ctx, con); w != nil {
			if closer, ok := w.(io.Closer); ok {
				defer func() {
					closeErr := closer.Close()
					if err == nil {
						err = errors.Wrap(closeErr, "closing stderr")
					}
				}()
			}
			cmd.Stderr = w
		}
	}

	var buf bytes.Buffer

	if GetVerbose(ctx) {
		if cmd.Stdout == nil {
			cmd.Stdout = con.IndentingCopier(os.Stdout, "    ")
		}
		if cmd.Stderr == nil {
			cmd.Stderr = con.IndentingCopier(os.Stderr, "    ")
		}
		con.Indentf("  Running command %s", cmd)
	} else {
		if cmd.Stdout == nil {
			cmd.Stdout = &buf
		}
		if cmd.Stderr == nil {
			cmd.Stderr = &buf
		}
	}

	cmd.Stdin = c.Stdin
	if c.StdinFile != "" {
		f, err := os.Open(c.StdinFile)
		if err != nil {
			return errors.Wrapf(err, "opening %s", c.StdinFile)
		}
		defer f.Close()
		cmd.Stdin = f
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

// Desc implements Target.Desc.
func (*Command) Desc() string {
	return "Command"
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

func commandDecoder(con *Controller, node *yaml.Node, dir string) (Target, error) {
	if node.Kind != yaml.MappingNode {
		return nil, BadYAMLNodeKindError{Got: node.Kind, Want: yaml.MappingNode}
	}

	var c commandYAML
	if err := node.Decode(&c); err != nil {
		return nil, errors.Wrap(err, "YAML error decoding Command")
	}

	args, err := YAMLStringList(con, &c.Args, dir)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error decoding Command.Args")
	}
	env, err := YAMLStringList(con, &c.Env, dir)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error decoding Command.Env")
	}

	if c.Cmd == "" {
		strs, err := YAMLStringList(con, &c.Shell, dir)

		var e BadYAMLNodeKindError
		switch {
		case errors.As(err, &e):
			// Do nothing (fall through below)

		case err != nil:
			return nil, errors.Wrap(err, "decoding Command.Shell as a string list")

		default:
			// Special case: Shell is a list of strings.
			// Make this a Seq of identical-except-for-the-shell-string Commands.

			targets, err := slices.Mapx(strs, func(idx int, str string) (Target, error) {
				return c.toTarget(con, str, dir, args, env, idx > 0), nil
			})
			return Seq(targets...), err
		}
	}

	var shell string
	switch c.Shell.Kind {
	case 0:
		// Do nothing

	case yaml.ScalarNode:
		shell = c.Shell.Value

	default:
		return nil, errors.Wrap(BadYAMLNodeKindError{Got: c.Shell.Kind, Want: yaml.ScalarNode}, "in Command.Shell node")
	}

	return c.toTarget(con, shell, dir, args, env, false), nil
}

type commandYAML struct {
	Shell  yaml.Node `yaml:"Shell"`
	Cmd    string    `yaml:"Cmd"`
	Args   yaml.Node `yaml:"Args"`
	Stdin  string    `yaml:"Stdin"`
	Stdout string    `yaml:"Stdout"`
	Stderr string    `yaml:"Stderr"`
	Dir    string    `yaml:"Dir"`
	Env    yaml.Node `yaml:"Env"`
}

func (c commandYAML) toTarget(con *Controller, shell, dir string, args, env []string, forceAppend bool) Target {
	result := &Command{
		Shell: shell,
		Cmd:   c.Cmd,
		Args:  args,
		Dir:   con.JoinPath(dir, c.Dir),
		Env:   env,
	}

	if c.Stdin == "$stdin" {
		result.Stdin = os.Stdin
	}

	switch c.Stdout {
	case "$stdout":
		result.Stdout = os.Stdout

	case "$stderr":
		result.Stdout = os.Stderr // who am I to judge

	case "$discard":
		result.Stdout = io.Discard

	case "$indent":
		result.StdoutFn = deferredIndent(os.Stdout)

	case "$verbose":
		result.StdoutFn = maybeIndent(os.Stdout)

	case "":
		// do nothing

	default:
		result.StdoutFile = con.JoinPath(dir, c.Stdout)
		if forceAppend && !strings.HasPrefix(result.StdoutFile, ">>") {
			result.StdoutFile = ">>" + result.StdoutFile
		}
	}

	switch c.Stderr {
	case "$stdout":
		result.Stderr = os.Stdout

	case "$stderr":
		result.Stderr = os.Stderr

	case "$discard":
		result.Stderr = io.Discard

	case "$indent":
		result.StderrFn = deferredIndent(os.Stderr)

	case "$verbose":
		result.StderrFn = maybeIndent(os.Stderr)

	case "":
		// do nothing

	default:
		result.StderrFile = con.JoinPath(dir, c.Stderr)
		if forceAppend && !strings.HasPrefix(result.StderrFile, ">>") {
			result.StderrFile = ">>" + result.StderrFile
		}
	}

	return result
}

func deferredIndent(w io.Writer) func(context.Context, *Controller) io.Writer {
	return func(_ context.Context, con *Controller) io.Writer {
		return con.IndentingCopier(w, "    ")
	}
}

func maybeIndent(w io.Writer) func(context.Context, *Controller) io.Writer {
	return func(ctx context.Context, con *Controller) io.Writer {
		if GetVerbose(ctx) {
			return con.IndentingCopier(w, "    ")
		}
		return nil
	}
}

func init() {
	RegisterYAMLTarget("Command", commandDecoder)
}
