package fab

import (
	"context"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v4/slices"
	"gopkg.in/yaml.v3"
)

// Parallel produces a target that runs a collection of targets in parallel.
//
// It is JSON-encodable
// (and therefore usable as the subtarget in [Files])
// if all of the targets in its collection are.
//
// A Parallel target may be specified in YAML using the tag !Parallel,
// which introduces a sequence of subtargets.
func Parallel(targets ...Target) Target {
	return &all{Targets: targets}
}

type all struct {
	Targets []Target
}

var _ Target = &all{}

// Run implements Target.Run.
func (a *all) Run(ctx context.Context, con *Controller) error {
	return con.Run(ctx, a.Targets...)
}

// Desc implements Target.Desc.
func (*all) Desc() string {
	return "Parallel"
}

func parallelDecoder(con *Controller, node *yaml.Node, dir string) (Target, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, BadYAMLNodeKindError{Got: node.Kind, Want: yaml.SequenceNode}
	}
	targets, err := slices.Mapx(node.Content, func(idx int, n *yaml.Node) (Target, error) {
		target, err := con.YAMLTarget(n, dir)
		return target, errors.Wrapf(err, "child %d", idx)
	})
	if err != nil {
		return nil, errors.Wrap(err, "YAML error decoding Parallel")
	}
	return Parallel(targets...), nil
}
