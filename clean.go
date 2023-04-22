package fab

import (
	"context"
	"io/fs"
	"os"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/slices"
	"gopkg.in/yaml.v3"
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

func cleanDecoder(node *yaml.Node) (Target, error) {
	if node.Kind != yaml.SequenceNode {
		// xxx error
	}
	files, err := slices.Mapx(node.Content, func(idx int, n *yaml.Node) (string, error) {
		if n.Kind != yaml.ScalarNode {
			// xxx error
		}
		return n.Value, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Clean node")
	}
	return Clean(files...), nil
}

func init() {
	RegisterYAML("Clean", cleanDecoder)
}
