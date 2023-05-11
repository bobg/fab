package fab

import (
	"context"
	"io/fs"
	"os"
	"sort"
	"sync"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/set"
	"gopkg.in/yaml.v3"
)

var (
	autocleanRegistryMu sync.Mutex
	autocleanRegistry   = set.New[string]()
)

// Clean is a Target that deletes the files named in Files when it runs.
// Files that don't exist are silently ignored.
//
// A Clean target may be specified in YAML using the tag !Clean,
// which introduces a sequence.
// The elements of the sequence are interpreted by [YAMLStringListFromNodes]
// to produce the list of files for the target.
// Any of those elements may be !Autoclean,
// which is replaced with the list of files selected for autocleaning by [Files].
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
			con.Indentf("would remove %v", c.Files)
		}
		return nil
	}
	if GetVerbose(ctx) {
		con.Indentf("removing %v", c.Files)
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

// AutocleanFiles returns the list of files in the autoclean registry.
// See [Files].
func AutocleanFiles() []string {
	autocleanRegistryMu.Lock()
	result := autocleanRegistry.Slice()
	autocleanRegistryMu.Unlock()

	sort.Strings(result)

	return result
}

// AutocleanAdd adds files to the autoclean registry.
// Duplicates are eliminated.
// See [Files].
func AutocleanAdd(files []string) {
	autocleanRegistryMu.Lock()
	autocleanRegistry.Add(files...)
	autocleanRegistryMu.Unlock()
}

func init() {
	RegisterYAMLTarget("Clean", cleanDecoder)
	RegisterYAMLStringList("Autoclean", func(*yaml.Node) ([]string, error) {
		return AutocleanFiles(), nil
	})
}
