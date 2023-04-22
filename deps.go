package fab

import (
	"fmt"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/slices"
	"gopkg.in/yaml.v3"
)

// Deps wraps a target with a set of dependencies,
// making sure those run first.
//
// It is equivalent to Seq(All(depTargets...), target).
func Deps(target Target, depTargets ...Target) Target {
	return Seq(All(depTargets...), target)
}

func depsDecoder(node *yaml.Node) (Target, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("got node kind %v, want %v", node.Kind, yaml.SequenceNode)
	}
	if len(node.Content) == 0 {
		return nil, fmt.Errorf("no child nodes")
	}
	target, err := YAMLTarget(node.Content[0])
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in target child of Deps node")
	}

	depTargets, err := slices.Mapx(node.Content[1:], func(idx int, n *yaml.Node) (Target, error) {
		target, err := YAMLTarget(n)
		return target, errors.Wrapf(err, "deptarget %d", idx)
	})
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in child of Deps node")
	}

	return Deps(target, depTargets...), nil
}

func init() {
	RegisterYAMLTarget("Deps", depsDecoder)
}
