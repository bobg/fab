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

// Clean is a Target that deletes the files named in Files when it runs.
// Files that already don't exist are silently ignored.
//
// If Autoclean is true,
// files listed in the "autoclean registry" are also removed.
// See [Autoclean] for more about this feature.
//
// A Clean target may be specified in YAML using the tag !Clean.
// It may introduce a sequence,
// in which case the elements are files to delete,
// or a mapping with fields `Files`,
// the files to delete,
// and `Autoclean`,
// a boolean for enabling the autoclean feature.
//
// When [GetDryRun] is true,
// Clean will not remove any files.
type Clean struct {
	Files     []string
	Autoclean bool
}

// Run implements Target.Run.
func (c *Clean) Run(ctx context.Context, con *Controller) error {
	files := c.Files
	if c.Autoclean {
		autocleanMu.Lock()
		autocleanFiles := autocleanRegistry.Slice()
		files = append(files, autocleanFiles...)
		autocleanMu.Unlock()
	}
	sort.Strings(files)

	if len(files) == 0 {
		return nil
	}

	if GetDryRun(ctx) {
		if GetVerbose(ctx) {
			con.Indentf("  would remove %v", files)
		}
		return nil
	}
	if GetVerbose(ctx) {
		con.Indentf("  removing %v", files)
	}
	for _, f := range files {
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
func (*Clean) Desc() string {
	return "Clean"
}

var (
	autocleanMu       sync.Mutex
	autocleanRegistry = set.New[string]()
)

func cleanDecoder(con *Controller, node *yaml.Node, dir string) (Target, error) {
	var (
		files     []string
		autoclean bool
		err       error
	)

	switch node.Kind {
	case yaml.MappingNode:
		var yclean struct {
			Files     yaml.Node `yaml:"Files"`
			Autoclean bool      `yaml:"Autoclean"`
		}
		if err = node.Decode(&yclean); err != nil {
			return nil, errors.Wrap(err, "YAML error in Clean node")
		}
		files, err = con.YAMLFileList(&yclean.Files, dir)
		if err != nil {
			return nil, errors.Wrap(err, "YAML error in Clean.Files node")
		}
		autoclean = yclean.Autoclean

	case yaml.SequenceNode:
		files, err = con.YAMLFileListFromNodes(node.Content, dir)
		if err != nil {
			return nil, errors.Wrap(err, "YAML error in Clean node children")
		}

	default:
		return nil, BadYAMLNodeKindError{Got: node.Kind, Want: yaml.MappingNode | yaml.SequenceNode}
	}

	return &Clean{Files: files, Autoclean: autoclean}, nil
}

func init() {
	RegisterYAMLTarget("Clean", cleanDecoder)
}
