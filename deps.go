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
//
// A Deps target may be specified in YAML using the !Deps tag,
// which introduces a sequence.
// The first element of the sequence is the main subtarget,
// or target name.
// Remaining elements are dependency targets or names.
// Example:
//
//	Foo: !Deps
//	  - Post
//	  - Pre1
//	  - Pre2
//
// This creates target Foo,
// which runs target Post after running Pre1 and Pre2.
// xxx reconsider
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
