package fab

import (
	"context"
	"fmt"

	"github.com/bobg/errors"
	"gopkg.in/yaml.v3"
)

// All produces a target that runs a collection of targets in parallel.
func All(targets ...Target) Target {
	return &all{Namer: NewNamer("all"), targets: targets}
}

type all struct {
	*Namer
	targets []Target
}

var _ Target = &all{}

// Run implements Target.Run.
func (a *all) Run(ctx context.Context) error {
	return Run(ctx, a.targets...)
}

func allDecoder(node *yaml.Node) (Target, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("got node kind %v, want %v", node.Kind, yaml.SequenceNode)
	}
	var targets []Target
	for i, child := range node.Content {
		target, err := YAMLTarget(child)
		if err != nil {
			return nil, errors.Wrapf(err, "YAML error in All node, child %d", i)
		}
		targets = append(targets, target)
	}
	return All(targets...), nil
}

func init() {
	RegisterYAMLTarget("All", allDecoder)
}
