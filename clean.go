package fab

import (
	"context"
	"io/fs"
	"os"

	"github.com/bobg/errors"
	"gopkg.in/yaml.v3"
)

// Clean is a Target that deletes the files named in Files when it runs.
// Files that already don't exist are silently ignored.
//
// A Clean target may be specified in YAML using the tag !Clean,
// which introduces a sequence.
// The elements of the sequence are interpreted by [YAMLStringListFromNodes]
// to produce the list of files for the target.
//
// When [GetDryRun] is true,
// Clean will not remove any files.
func Clean(files ...string) Target {
	return &clean{
		Files: files,
	}
}

type clean struct {
	Files []string
}

// Run implements Target.Run.
func (c *clean) Run(ctx context.Context, con *Controller) error {
	if GetDryRun(ctx) {
		if GetVerbose(ctx) {
			con.Indentf("  would remove %v", c.Files)
		}
		return nil
	}
	if GetVerbose(ctx) {
		con.Indentf("  removing %v", c.Files)
	}
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

// Desc implements Target.Desc.
func (*clean) Desc() string {
	return "Clean"
}

func cleanDecoder(con *Controller, node *yaml.Node, dir string) (Target, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, BadYAMLNodeKindError{Got: node.Kind, Want: yaml.SequenceNode}
	}
	files, err := con.YAMLFileListFromNodes(node.Content, dir)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Clean node")
	}
	return Clean(files...), nil
}

func init() {
	RegisterYAMLTarget("Clean", cleanDecoder)
}
