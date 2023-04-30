package fab

import (
	"io/fs"
	"os"
	"sync"
)

type Controller struct {
	fsys   fs.FS
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

func NewController(topdir string) *Controller {
	return NewControllerFS(os.DirFS("/"), topdir)
}

func NewControllerFS(fsys fs.FS, topdir string) *Controller {
	return &Controller{
		fsys:          fsys,
		topdir:        topdir,
		ran:           make(map[uintptr]*outcome),
		files:         make(map[string]*files),
		targetsByName: make(map[string]targetRegistryTuple),
		targetsByAddr: make(map[uintptr]targetRegistryTuple),
	}
}
