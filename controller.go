package fab

import (
	"path/filepath"
	"sync"
)

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
// The top directory is where a _fab subdirectory and/or a fab.yaml file is expected.
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
