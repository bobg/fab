package fab

import (
	"context"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/slices"
	"gopkg.in/yaml.v3"
)

// Seq produces a target that runs a collection of targets in sequence.
// Its Run method exits early when a target in the sequence fails.
//
// It is JSON-encodable
// (and therefore usable as the subtarget in [Files])
// if all of the targets in its collection are.
//
// A Seq target may be specified in YAML using the tag !Seq,
// which introduces a sequence.
// The elements in the sequence are targets themselves,
// or target names.
func Seq(targets ...Target) Target {
	return &seq{targets: targets}
}

type seq struct {
	targets []Target
}

var _ Target = &seq{}

// Run implements Target.Run.
func (s *seq) Run(ctx context.Context, con *Controller) error {
	for _, t := range s.targets {
		if err := con.Run(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func (*seq) Desc() string {
	return "Seq"
}

func seqDecoder(con *Controller, node *yaml.Node, dir string) (Target, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, BadYAMLNodeKindError{Got: node.Kind, Want: yaml.SequenceNode}
	}
	targets, err := slices.Mapx(node.Content, func(idx int, n *yaml.Node) (Target, error) {
		target, err := con.YAMLTarget(n, dir)
		if err != nil {
			return nil, errors.Wrapf(err, "child %d", idx)
		}
		return target, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Seq node")
	}
	return Seq(targets...), nil
}

func init() {
	RegisterYAMLTarget("Seq", seqDecoder)
}
