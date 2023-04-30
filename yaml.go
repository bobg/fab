package fab

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/slices"
	"gopkg.in/yaml.v3"
)

type (
	// YAMLTargetFunc is the type of a function in the YAML target registry.
	YAMLTargetFunc = func(fs.FS, *yaml.Node, string) (Target, error)

	// YAMLStringListFunc is the type of a function in the YAML string-list registry.
	YAMLStringListFunc = func(*yaml.Node) ([]string, error)
)

var (
	yamlTargetRegistryMu sync.Mutex
	yamlTargetRegistry   = make(map[string]YAMLTargetFunc)

	yamlStringListRegistryMu sync.Mutex
	yamlStringListRegistry   = make(map[string]YAMLStringListFunc)
)

// RegisterYAMLTarget places a function in the YAML target registry with the given name.
// Use a YAML `!name` tag to introduce a node that should be parsed using this function.
func RegisterYAMLTarget(name string, fn YAMLTargetFunc) {
	yamlTargetRegistryMu.Lock()
	yamlTargetRegistry[name] = fn
	yamlTargetRegistryMu.Unlock()
}

// YAMLTarget parses a [Target] from a YAML node.
// If the node has a tag `!foo`,
// then the [YAMLTargetFunc] in the YAML target registry named `foo` is used to parse the node.
// Otherwise,
// if the node is a bare string `foo`,
// then it is presumed to refer to a target in the (non-YAML) target registry named `foo`.
func YAMLTarget(fsys fs.FS, node *yaml.Node, dir string) (Target, error) {
	tag := normalizeTag(node.Tag)

	if tag == "" && node.Kind == yaml.ScalarNode {
		qualifiedName := filepath.Join(dir, node.Value)
		if tdir := filepath.Dir(qualifiedName); tdir != "." {
			found, _ := RegistryTarget(qualifiedName)
			if found != nil {
				return found, nil
			}

			if err := ReadYAMLFileFS(fsys, tdir); err != nil {
				return nil, errors.Wrapf(err, "resolving target %s", qualifiedName)
			}

			found, _ = RegistryTarget(qualifiedName)
			if found != nil {
				return found, nil
			}

			return nil, fmt.Errorf("cannot resolve target %s", qualifiedName)
		}

		return &deferredResolutionTarget{Name: qualifiedName}, nil
	}

	if tag == "" {
		return nil, fmt.Errorf("untyped YAML target node")
	}

	yamlTargetRegistryMu.Lock()
	fn, ok := yamlTargetRegistry[tag]
	yamlTargetRegistryMu.Unlock()

	if !ok {
		return nil, fmt.Errorf("unknown YAML target type %s", tag)
	}
	return fn(fsys, node, dir)
}

type deferredResolutionTarget struct {
	mu     sync.Mutex
	Name   string
	Target Target
}

var _ Target = &deferredResolutionTarget{}

func (dt *deferredResolutionTarget) resolve() (Target, error) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if dt.Target == nil {
		target, _ := RegistryTarget(dt.Name)
		if target == nil {
			return nil, fmt.Errorf("cannot resolve target %s", dt.Name)
		}
		dt.Target = target
	}

	return dt.Target, nil
}

func (dt *deferredResolutionTarget) Execute(ctx context.Context) error {
	target, err := dt.resolve()
	if err != nil {
		return err
	}
	return Run(ctx, target)
}

func (dt *deferredResolutionTarget) Desc() string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.Name
}

// ReadYAML reads a YAML document from the given source,
// registering Targets that it finds.
//
// The top level of the YAML document should be a mapping from names to targets.
// Each target is either a target-typed node,
// selected by a !tag,
// or the name of some other target.
//
// For example,
// the following creates a target named `Check`,
// which is an `All`-typed target
// referring to two other targets: `Vet` and `Test`.
// Each of those is a `Command`-typed target
// executing specific shell commands.
//
//	Check: !All
//	  - Vet
//	  - Test
//
//	Vet: !Command
//	  - go vet ./...
//
//	Test: !Command
//	  - go test ./...
func ReadYAML(fsys fs.FS, r io.Reader, dir string) error { // xxx propagate dir
	var (
		dec = yaml.NewDecoder(r)
		doc yaml.Node
	)

	if err := dec.Decode(&doc); err != nil {
		return errors.Wrap(err, "decoding YAML")
	}

	if doc.Kind != yaml.DocumentNode {
		return fmt.Errorf("got top-level node kind %v, want %v", doc.Kind, yaml.DocumentNode)
	}
	if len(doc.Content) != 1 {
		return fmt.Errorf("got %d children of top-level node, want 1", len(doc.Content))
	}

	m := doc.Content[0]
	if m.Kind != yaml.MappingNode {
		return fmt.Errorf("got second-level node kind %v, want %v", m.Kind, yaml.MappingNode)
	}

	if len(m.Content)%2 != 0 {
		return fmt.Errorf("got %d children for second-level node, want an even number", len(m.Content))
	}

	for i := 0; i < len(m.Content); i += 2 {
		nameNode := m.Content[i]
		if nameNode.Kind != yaml.ScalarNode {
			return fmt.Errorf("got name-node kind %v for entry %d, want %v", nameNode.Kind, i, yaml.ScalarNode)
		}

		var (
			name = nameNode.Value
			doc  = nameNode.HeadComment
		)
		if doc == "" {
			doc = nameNode.LineComment
		}
		doc = strings.TrimLeft(doc, "# ")

		if name == "_dir" {
			// xxx check value against actual dir
			continue
		}

		if strings.Contains(name, "/") {
			return fmt.Errorf("no slashes in target names")
		}

		targetNode := m.Content[i+1]
		target, err := YAMLTarget(fsys, targetNode, dir)
		if err != nil {
			return errors.Wrapf(err, "in YAML node for %s", name)
		}

		// The following was previously inside a "if target is not a deferredResolutionTarget" block,
		// but I think that was wrong.
		// Or maybe I'm wrong now...
		qualifiedName := filepath.Join(dir, name)
		_, err = RegisterTarget(qualifiedName, doc, target)
		if err != nil {
			return errors.Wrapf(err, "registering target %s", qualifiedName)
		}
	}

	return nil
}

// ReadYAMLFile calls ReadYAML
// on the file `fab.yaml` in the given directory
// or, if that doesn't exist,
// `fab.yml`.
//
// The specified directory should be relative to the project's top level.
func ReadYAMLFile(dir string) error {
	// https://pkg.go.dev/os#DirFS assures us that the result of os.DirFS implements StatFS.
	return ReadYAMLFileFS(os.DirFS("/"), dir)
}

func ReadYAMLFileFS(fsys fs.FS, dir string) error {
	rc, err := openFabYAMLFS(fsys, dir)
	if err != nil {
		return err
	}
	defer rc.Close()

	return ReadYAML(fsys, rc, dir)
}

func openFabYAMLFS(fsys fs.FS, dir string) (io.ReadCloser, error) {
	filename := filepath.Join(dir, "fab.yaml")
	f, err := fsys.Open(filename)
	if errors.Is(err, fs.ErrNotExist) {
		filename = filepath.Join(dir, "fab.yml")
		f, err = fsys.Open(filename)
	}
	return f, err
}

// RegisterYAMLStringList places a function in the YAML string-list registry with the given name.
// Use a YAML `!name` tag to introduce a node that should be parsed using this function.
func RegisterYAMLStringList(name string, fn YAMLStringListFunc) {
	yamlStringListRegistryMu.Lock()
	yamlStringListRegistry[name] = fn
	yamlStringListRegistryMu.Unlock()
}

// YAMLStringList parses a []string from a YAML node.
// If the node has a tag `!foo`,
// then the [YAMLStringListFunc] in the YAML string-list registry named `foo` is used to parse the node.
// Otherwise,
// the node is expected to be a sequence,
// and [YAMLStringListFromNodes] is called on its children.
func YAMLStringList(node *yaml.Node) ([]string, error) {
	if node.Kind == 0 {
		return nil, nil
	}

	tag := normalizeTag(node.Tag)

	if tag != "" {
		yamlStringListRegistryMu.Lock()
		fn, ok := yamlStringListRegistry[tag]
		yamlStringListRegistryMu.Unlock()

		if !ok {
			return nil, fmt.Errorf("unknown YAML string-list type %s", tag)
		}

		return fn(node)
	}

	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("got node kind %v, want %v", node.Kind, yaml.SequenceNode)
	}

	return YAMLStringListFromNodes(node.Content)
}

// YAMLStringListFromNodes constructs a slice of strings from a slice of YAML nodes.
// Each node may be a plain scalar,
// in which case it is added to the result slice;
// or a tagged node,
// in which case it is parsed with the corresponding YAML string-list registry function
// and the output appended to the result slice.
func YAMLStringListFromNodes(nodes []*yaml.Node) ([]string, error) {
	var result []string

	for _, node := range nodes {
		tag := normalizeTag(node.Tag)

		if tag == "" && node.Kind == yaml.ScalarNode {
			result = append(result, node.Value)
			continue
		}

		if tag == "" {
			return nil, fmt.Errorf("got node kind %v, want %v", node.Kind, yaml.ScalarNode)
		}

		yamlStringListRegistryMu.Lock()
		fn, ok := yamlStringListRegistry[tag]
		yamlStringListRegistryMu.Unlock()

		if !ok {
			return nil, fmt.Errorf("unknown YAML string-list type %s", tag)
		}

		strs, err := fn(node)
		if err != nil {
			return nil, err
		}
		result = append(result, strs...)
	}

	return result, nil
}

func YAMLFileList(node *yaml.Node, dir string) ([]string, error) {
	strs, err := YAMLStringList(node)
	if err != nil {
		return nil, err
	}
	return slices.Map(strs, func(s string) string { return filepath.Join(dir, s) }), nil // xxx unless absolute
}

func YAMLFileListFromNodes(nodes []*yaml.Node, dir string) ([]string, error) {
	strs, err := YAMLStringListFromNodes(nodes)
	if err != nil {
		return nil, err
	}
	return slices.Map(strs, func(s string) string { return filepath.Join(dir, s) }), nil // xxx unless absolute
}

func normalizeTag(tag string) string {
	if strings.HasPrefix(tag, "!!") {
		return ""
	}
	return strings.TrimPrefix(tag, "!")
}
