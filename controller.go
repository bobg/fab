package fab

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
)

// Controller is in charge of registering and running targets.
// It keeps track of targets that are presently running or have previously run.
// It will not run the same target more than once.
// The second and subsequent request to run a given target
// will used the cached outcome
// (error or no error)
// of the first run.
type Controller struct {
	topdir string // absolute, or relative to the current directory

	mu sync.Mutex // protects the remaining fields

	depth int

	// Records targets that have run or are running.
	ran map[uintptr]*outcome

	// Keys are names related to topdir.
	targetsByName map[string]targetRegistryTuple

	targetsByAddr map[uintptr]targetRegistryTuple
}

// NewController creates a new [Controller]
// for the project with the given top-level directory.
//
// The top directory is where a _fab subdirectory and/or a top-level fab.yaml file is expected.
func NewController(topdir string) *Controller {
	return &Controller{
		topdir:        topdir,
		ran:           make(map[uintptr]*outcome),
		targetsByName: make(map[string]targetRegistryTuple),
		targetsByAddr: make(map[uintptr]targetRegistryTuple),
	}
}

// JoinPath is like [filepath.Join] with some additional behavior.
// Any absolute path segment discards everything to the left of it.
// If all path segments are relative,
// then con's top directory is implicitly joined at the beginning.
//
// Examples:
//
//   - JoinPath("a/b", "c/d") -> TOP/a/b/c/d
//   - JoinPath("a/b", "/c/d") -> /c/d
func (con *Controller) JoinPath(elts ...string) string {
	for i := len(elts) - 1; i >= 0; i-- {
		if filepath.IsAbs(elts[i]) {
			return filepath.Join(elts[i:]...)
		}
	}
	args := make([]string, 1, 1+len(elts))
	args[0] = con.topdir
	args = append(args, elts...)
	return filepath.Join(args...)
}

// RelPath returns the relative path to `path` from con's top directory.
func (con *Controller) RelPath(path string) (string, error) {
	return filepath.Rel(con.topdir, path)
}

// ParseArgs parses the remaining arguments on a fab command line,
// after option flags.
// They are either a list of target names in the registry,
// in which case those targets are returned;
// or a single registry target followed by option flags for that,
// in which case the target is wrapped up in an [ArgTarget] with its options.
// The two cases are distinguished by whether there is a second argument
// and whether it begins with a hyphen.
// (That's the ArgTarget case.)
func (con *Controller) ParseArgs(args []string) ([]Target, error) {
	var (
		targets []Target
		unknown []string
	)

	if len(args) > 1 && args[1][0] == '-' {
		// Just one target, and remaining args are arguments for that target.
		if target, _ := con.RegistryTarget(args[0]); target != nil {
			targets = append(targets, ArgTarget(target, args[1:]...))
		} else {
			unknown = append(unknown, args[0])
		}
	} else {
		for _, arg := range args {
			if target, _ := con.RegistryTarget(arg); target != nil {
				targets = append(targets, target)
			} else {
				unknown = append(unknown, arg)
			}
		}
	}

	switch len(unknown) {
	case 0:
		return targets, nil
	case 1:
		return nil, fmt.Errorf("unknown target %s", unknown[0])
	default:
		return nil, fmt.Errorf("unknown targets: %s", strings.Join(unknown, " "))
	}
}

// ListTargets outputs a formatted list of the targets in the registry and their docstrings.
func (con *Controller) ListTargets(w io.Writer) {
	names := con.RegistryNames()
	for _, name := range names {
		fmt.Fprintln(w, name)
		if _, d := con.RegistryTarget(name); d != "" {
			d = bolRegex.ReplaceAllString(d, "    ")
			fmt.Fprintln(w, d)
		}
	}
}
