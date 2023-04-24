package fab

import (
	"context"
	"fmt"

	"github.com/bobg/errors"
	"gopkg.in/yaml.v3"
)

// All produces a target that runs a collection of targets in parallel.
//
// An All target may be specified in YAML using the tag !All,
// which introduces a sequence.
// The elements in the sequence are targets themselves,
// or target names.
func All(targets ...Target) Target {
	return &all{targets: targets}
}

type all struct {
	targets []Target
}

var _ Target = &all{}

// Run implements Target.Run.
func (a *all) Run(ctx context.Context) error {
	return Run(ctx, a.targets...)
}

func (*all) Desc() string {
	return "All"
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
