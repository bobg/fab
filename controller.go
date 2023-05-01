package fab

import (
	"path/filepath"
	"sync"
)

type Controller struct {
	topdir string

	mu sync.Mutex // protects the remaining fields

	depth int

	ran map[uintptr]*outcome

	// Maps output files from Files targets
	// to the targets that create them.
	// Keys are qualified filenames.
	files map[string]*files

	// Keys are qualified names.
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
		files:         make(map[string]*files),
		targetsByName: make(map[string]targetRegistryTuple),
		targetsByAddr: make(map[uintptr]targetRegistryTuple),
	}
}

func (con *Controller) RelPath(path, dir string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return filepath.Rel(con.topdir, filepath.Join(dir, path))
}

func (con *Controller) AbsPath(path, dir string) (string, error) {
	rel, err := con.RelPath(path, dir)
	if err != nil {
		return "", err
	}
	return filepath.Join(con.topdir, rel), nil
}
