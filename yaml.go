package fab

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type YAMLFunc = func(*yaml.Node) (Target, error)

var (
	yamlRegistryMu sync.Mutex
	yamlRegistry   = make(map[string]YAMLFunc)
)

func RegisterYAML(name string, fn YAMLFunc) {
	yamlRegistryMu.Lock()
	yamlRegistry[name] = fn
	yamlRegistryMu.Unlock()
}

func YAMLTarget(node *yaml.Node) (Target, error) {
	if node.Kind == yaml.ScalarNode {
		return &deferredResolutionTarget{name: node.Value}, nil
	}
	tag := node.Tag
	if tag == "" {
		return nil, fmt.Errorf("untyped YAML target node")
	}
	if strings.HasPrefix(tag, "!!") {
		return nil, fmt.Errorf("invalid YAML target type %s", tag)
	}
	typ := tag[1:]

	yamlRegistryMu.Lock()
	fn, ok := yamlRegistry[typ]
	yamlRegistryMu.Unlock()

	if !ok {
		return nil, fmt.Errorf("unknown YAML target type %s", typ)
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
	dt.name = name
	if dt.target != nil {
		dt.target.SetName(name)
	}
}
