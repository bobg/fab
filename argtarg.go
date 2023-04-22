package fab

import (
	"context"
	"fmt"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/slices"
	"gopkg.in/yaml.v3"
)

// ArgTarget produces a target with associated arguments
// as a list of strings,
// suitable for parsing with the [flag] package.
// When the target runs,
// its arguments are available from the context using [GetArgs].
func ArgTarget(target Target, args ...string) Target {
	return &argTarget{
		Namer:  NewNamer("args-" + target.Name()),
		target: target,
		args:   args,
	}
}

type argTarget struct {
	*Namer
	target Target
	args   []string
}

var _ Target = &argTarget{}

func (a *argTarget) Run(ctx context.Context) error {
	ctx = WithArgs(ctx, a.args...)
	return a.target.Run(ctx)
}

func argTargetDecoder(node *yaml.Node) (Target, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("got node kind %v, want %v", node.Kind, yaml.SequenceNode)
	}
	if len(node.Content) == 0 {
		return nil, fmt.Errorf("no child nodes")
	}
	target, err := YAMLTarget(node.Content[0])
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in target child of AllTarget node")
	}

	args, err := slices.Mapx(node.Content[1:], func(idx int, n *yaml.Node) (string, error) {
		if n.Kind != yaml.ScalarNode {
			return "", fmt.Errorf("got child node kind %v, want %v", n.Kind, yaml.ScalarNode)
		}
		return n.Value, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in AllTarget node")
	}

	return ArgTarget(target, args...), nil
}

func init() {
	RegisterYAMLTarget("ArgTarget", argTargetDecoder)
}
