package fab

import (
	"context"
	"fmt"

	"github.com/bobg/errors"
	"gopkg.in/yaml.v3"
)

// ArgTarget produces a target with associated arguments
// as a list of strings,
// suitable for parsing with the [flag] package.
// When the target runs,
// its arguments are available from the context using [GetArgs].
//
// An ArgTarget target may be specified in YAML using the tag !ArgTarget,
// which introduces a sequence.
// The first element of the sequence is a target or target name.
// The remaining elements of the sequence are interpreted byu [YAMLStringListFromNodes]
// to produce the arguments for the target.
func ArgTarget(target Target, args ...string) Target {
	return &argTarget{
		target: target,
		args:   args,
	}
}

type argTarget struct {
	target Target
	args   []string
}

var _ Target = &argTarget{}

func (a *argTarget) Run(ctx context.Context) error {
	ctx = WithArgs(ctx, a.args...)
	return Run(ctx, target)
}

func (*argTarget) Desc() string {
	return "ArgTarget"
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

	args, err := YAMLStringListFromNodes(node.Content[1:])
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in ArgTarget args")
	}

	return ArgTarget(target, args...), nil
}

func init() {
	RegisterYAMLTarget("ArgTarget", argTargetDecoder)
}
