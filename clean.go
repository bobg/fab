package fab

import (
	"context"
	"io/fs"
	"os"

	"github.com/pkg/errors"
)

// Clean is a Target that deletes the files named in Files when it runs.
// Files that don't exist are silently ignored.
func Clean(files ...string) Target {
	return &clean{
		Namer: NewNamer("clean"),
		Files: files,
	}
}

type clean struct {
	*Namer
	Files []string
}

// Run implements Target.Run.
func (c *clean) Run(_ context.Context) error {
	for _, f := range c.Files {
		err := os.Remove(f)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return errors.Wrapf(err, "removing %s", f)
		}
	}
	return nil
}
