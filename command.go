package fab

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bobg/errors"
	"gopkg.in/yaml.v3"
)

// Command is a Target whose Execute function executes a command in a subprocess.
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
//   - Stdout, the name of a file to which the command's standard output should be written.
//     The file is overwritten unless this is prefixed with >> which means append.
//     This may also be one of these special strings:
//     $stdout (copy the command's output to Fab's standard output);
//     $stderr (copy the command's output to Fab's standard error);
//     $indent (indent the command's output with [IndentingCopier] and copy it to Fab's standard output);
//     $discard (discard the command's output).
//   - Stderr, the name of a file to which the command's standard error should be written.
//     The file is overwritten unless this is prefixed with >> which means append.
//     This may also be one of these special strings:
//     $stdout (copy the command's error output to Fab's standard error);
//     $stderr (copy the command's error output to Fab's standard error);
//     $indent (indent the command's error output with [IndentingCopier] and copy it to Fab's standard error);
//     $discard (discard the command's error output).
//   - Dir, the directory in which the command should run.
//   - Env, a list of VAR=VALUE strings to add to the command's environment.
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
	// until Execute is invoked,
	// at which time its context object is passed to this function to produce the [io.Writer] to use.
	// If the writer produced by this function is also an [io.Closer],
	// its Close method will be called before Execute exits.
	//
	// Stdout, StdoutFile, and StdoutFn are all mutually exclusive.
	StdoutFn func(context.Context) io.Writer `json:"-"`

	// StderrFn lets you defer assigning a value to Stderr
	// until Execute is invoked,
	// at which time its context object is passed to this function to produce the [io.Writer] to use.
	// If the writer produced by this function is also an [io.Closer],
	// its Close method will be called before Execute exits.
	//
	// Stderr, StderrFile, and StderrFn are all mutually exclusive.
	StderrFn func(context.Context) io.Writer `json:"-"`

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
	// The default is the value of GetDir(ctx) when the Execute method is called.
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

// Execute implements Target.Execute.
func (c *Command) Execute(ctx context.Context) (err error) {
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
		if stdoutFile == stderrFile {
			cmd.Stderr = cmd.Stdout
		} else if stderrAppend {
			f, err := os.OpenFile(stderrFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			if err != nil {
				return errors.Wrapf(err, "opening %s for appending", stderrFile)
			}
			defer f.Close()
			cmd.Stderr = f
		} else {
			f, err := os.Create(stderrFile)
			if err != nil {
				return errors.Wrapf(err, "opening %s for writing", stderrFile)
			}
			defer f.Close()
			cmd.Stderr = f
		}
	}

	if cmd.Stdout == nil && c.StdoutFn != nil {
		w := c.StdoutFn(ctx)
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
	if cmd.Stderr == nil && c.StderrFn != nil {
		w := c.StderrFn(ctx)
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

	var buf bytes.Buffer

	if GetVerbose(ctx) {
		if cmd.Stdout == nil {
			cmd.Stdout = IndentingCopier(ctx, os.Stdout, "    ")
		}
		if cmd.Stderr == nil {
			cmd.Stderr = IndentingCopier(ctx, os.Stderr, "    ")
		}
		Indentf(ctx, "  Running command %s", cmd)
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

// CommandErr is a type of error that may be returned from command.Execute.
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

func commandDecoder(_ fs.FS, node *yaml.Node, dir string) (Target, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("got node kind %v, want %v", node.Kind, yaml.MappingNode)
	}

	var c struct {
		Shell  string    `yaml:"Shell"`
		Cmd    string    `yaml:"Cmd"`
		Args   yaml.Node `yaml:"Args"`
		Stdin  string    `yaml:"Stdin"`
		Stdout string    `yaml:"Stdout"`
		Stderr string    `yaml:"Stderr"`
		Dir    string    `yaml:"Dir"`
		Env    yaml.Node `yaml:"Env"`
	}
	if err := node.Decode(&c); err != nil {
		return nil, errors.Wrap(err, "YAML error decoding Command")
	}

	args, err := YAMLStringList(&c.Args)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error decoding Command.Args")
	}
	env, err := YAMLStringList(&c.Env)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error decoding Command.Env")
	}

	result := &Command{
		Shell: c.Shell,
		Cmd:   c.Cmd,
		Args:  args,
		Dir:   c.Dir, // xxx default to dir?
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
		result.StdoutFn = func(ctx context.Context) io.Writer {
			return IndentingCopier(ctx, os.Stdout, "    ")
		}

	default:
		result.StdoutFile = filepath.Join(dir, c.Stdout) // xxx unless absolute
	}

	switch c.Stderr {
	case "$stdout":
		result.Stderr = os.Stdout

	case "$stderr":
		result.Stderr = os.Stderr

	case "$discard":
		result.Stderr = io.Discard

	case "$indent":
		result.StderrFn = func(ctx context.Context) io.Writer {
			return IndentingCopier(ctx, os.Stderr, "    ")
		}

	default:
		result.StderrFile = filepath.Join(dir, c.Stderr) // xxx unless absolute
	}

	return result, nil
}

func init() {
	RegisterYAMLTarget("Command", commandDecoder)
}
