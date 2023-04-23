package fab

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/bobg/errors"
	"gopkg.in/yaml.v3"
)

type (
	YAMLTargetFunc     = func(*yaml.Node) (Target, error)
	YAMLStringListFunc = func(*yaml.Node) ([]string, error)
)

var (
	yamlTargetRegistryMu sync.Mutex
	yamlTargetRegistry   = make(map[string]YAMLTargetFunc)

	yamlStringListRegistryMu sync.Mutex
	yamlStringListRegistry   = make(map[string]YAMLStringListFunc)
)

func RegisterYAMLTarget(name string, fn YAMLTargetFunc) {
	yamlTargetRegistryMu.Lock()
	yamlTargetRegistry[name] = fn
	yamlTargetRegistryMu.Unlock()
}

func YAMLTarget(node *yaml.Node) (Target, error) {
	tag := node.Tag
	if strings.HasPrefix(tag, "!!") {
		tag = ""
	}
	if strings.HasPrefix(tag, "!") {
		tag = tag[1:]
	}

	if tag == "" && node.Kind == yaml.ScalarNode {
		return &deferredResolutionTarget{name: node.Value}, nil
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
	return fn(node)
}

type deferredResolutionTarget struct {
	mu     sync.Mutex
	name   string
	target Target
}

var _ Target = &deferredResolutionTarget{}

func (dt *deferredResolutionTarget) resolve() (Target, error) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if dt.target == nil {
		target, _ := RegistryTarget(dt.name)
		if target == nil {
			return nil, fmt.Errorf("cannot resolve target %s", dt.name)
		}
		dt.target = target
	}

	return dt.target, nil
}

func (dt *deferredResolutionTarget) Run(ctx context.Context) error {
	target, err := dt.resolve()
	if err != nil {
		return err
	}
	return target.Run(ctx)
}

func (dt *deferredResolutionTarget) Name() string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.name
}

func (dt *deferredResolutionTarget) SetName(name string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.target != nil {
		dt.target.SetName(name)
	}
}

func ReadYAML(r io.Reader) error {
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

		targetNode := m.Content[i+1]
		target, err := YAMLTarget(targetNode)
		if err != nil {
			return errors.Wrapf(err, "in YAML node for %s", name)
		}

		if _, ok := target.(*deferredResolutionTarget); !ok {
			Register(name, doc, target)
		}
	}

	return nil
}

func RegisterYAMLStringList(name string, fn YAMLStringListFunc) {
	yamlStringListRegistryMu.Lock()
	yamlStringListRegistry[name] = fn
	yamlStringListRegistryMu.Unlock()
}

func YAMLStringList(node *yaml.Node) ([]string, error) {
	tag := node.Tag
	if strings.HasPrefix(tag, "!!") {
		tag = ""
	}
	if strings.HasPrefix(tag, "!") {
		tag = tag[1:]
	}

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

	var result []string

	for _, child := range node.Content {
		strs, err := YAMLStringList(child)
		if err != nil {
			return nil, err
		}
		result = append(result, strs...)
	}

	return result, nil
}
