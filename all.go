package fab

import (
	"context"
	"fmt"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/slices"
	"gopkg.in/yaml.v3"
)

// All produces a target that runs a collection of targets in parallel.
//
// It is JSON-encodable
// (and therefore usable as the subtarget in [Files])
// if all of the targets in its collection are.
//
// An All target may be specified in YAML using the tag !All,
// which introduces a sequence.
// The elements in the sequence are targets themselves,
// or target names.
func All(targets ...Target) Target {
	return &all{Targets: targets}
}

type all struct {
	Targets []Target
}

var _ Target = &all{}

// Run implements Target.Execute.
func (a *all) Execute(ctx context.Context) error {
	return Run(ctx, a.Targets...)
}

// Desc implements Target.Desc.
func (*all) Desc() string {
	return "All"
}

func allDecoder(node *yaml.Node) (Target, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("got node kind %v, want %v", node.Kind, yaml.SequenceNode)
	}
	targets, err := slices.Mapx(node.Content, func(idx int, n *yaml.Node) (Target, error) {
		target, err := YAMLTarget(n)
		return target, errors.Wrapf(err, "child %d", idx)
	})
	if err != nil {
		return nil, errors.Wrap(err, "YAML error decoding All")
	}
	return All(targets...), nil
}

func init() {
	RegisterYAMLTarget("All", allDecoder)
}
