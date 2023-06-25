package cc

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/set"
	"github.com/bobg/go-generics/v2/slices"

	"github.com/bobg/fab"
)

// Compile compiles a C or C++ file into an object file.
func Compile(cfile string, opts ...CcOpt) (fab.Target, error) {
	c := newCompile(cfile, opts...)
	return c.compile()
}

func (c *compile) compile() (fab.Target, error) {
	deps, err := c.deps()
	if err != nil {
		return nil, errors.Wrapf(err, "computing dependencies for %s", c.cfile)
	}

	args := slices.Map(c.includeDirs, func(dir string) string {
		return "-I" + dir
	})

	for k, v := range c.defines {
		args = append(args, "-D"+k+"="+v)
	}

	args = append(args, "-c", c.cfile)

	subtarget := &fab.Command{
		Cmd:  c.compiler,
		Args: args,
	}

	in := set.New[string](c.cfile)
	in.Add(deps...)

	var (
		ext  = filepath.Ext(c.cfile)
		root = c.cfile[:len(c.cfile)-len(ext)]
		out  = root + ".o"
	)

	return fab.Files(subtarget, in.Slice(), []string{out}, fab.Autoclean(true)), nil
}

type compile struct {
	compiler    string
	cfile       string
	includeDirs []string
	defines     map[string]string
}

type CcOpt func(*compile)

func Includes(dirs ...string) CcOpt {
	return func(c *compile) {
		c.includeDirs = append(c.includeDirs, dirs...)
	}
}

func Defines(defs map[string]string) CcOpt {
	return func(c *compile) {
		c.defines = defs
	}
}

func Link(out string, ofiles, libdirs, libs []string, opts ...LinkOpt) (fab.Target, error) {
	subtarget := &link{
		linker:  "cc",
		out:     out,
		ofiles:  ofiles,
		libdirs: libdirs,
		libs:    libs,
	}

	for _, opt := range opts {
		opt(subtarget)
	}

	return fab.Files(subtarget, in, []string{out}, fab.Autoclean(true))
}

type link struct {
	linker  string
	out     string
	ofiles  []string
	libdirs []string
	libs    []string
}

type LinkOpt func(*link)

func Deps(cfile string, opts ...CcOpt) ([]string, error) {
	c := newCompiler(c, opts...)
	return c.deps()
}

func (c *compile) deps() ([]string, error) {
	args := slices.Map(c.includeDirs, func(dir string) string {
		return "-I" + dir
	})
	for k, v := range c.defines {
		args = append(args, "-D"+k+"="+v)
	}
	args = append(args, "-MM", cfile)

	cmd := exec.Command(c.compiler, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "getting stdout pipe for dependency computation")
	}
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "starting dependency computation")
	}
	defer cmd.Wait()

	deps := set.New[string]()
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		deps.Add(fields[1:]...)
	}
	return deps.Slice(), errors.Wrap(sc.Err(), "scanning dependency computation output")
}

func newCompile(cfile string, opts ...CcOpt) *compile {
	c := &compile{
		cfile: cfile,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
