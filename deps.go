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
// A Deps target may be specified in YAML using the !Deps tag.
// This may introduce a sequence or a mapping.
//
// If a sequence,
// then the first element is the main subtarget (or target name),
// and the remaining elements are dependency targets (or names).
// Example:
//
//	Foo: !Deps
//	  - Main
//	  - Pre1
//	  - Pre2
//
// This creates target Foo,
// which runs target Main after running Pre1 and Pre2.
//
// If a mapping,
// then the `Pre` field specifies a sequence of dependency targets,
// and the `Post` field specifies the main subtarget.
// Example:
//
//	Foo: !Deps
//	  Pre:
//	    - Pre1
//	    - Pre2
//	  Post: Main
//
// This is equivalent to the first example above.
func Deps(target Target, depTargets ...Target) Target {
	return Seq(All(depTargets...), target)
}

func depsDecoder(node *yaml.Node) (Target, error) {
	switch node.Kind {
	case yaml.SequenceNode:
		if len(node.Content) == 0 {
			return nil, fmt.Errorf("no child nodes")
		}
		target, err := YAMLTarget(node.Content[0])
		if err != nil {
			return nil, errors.Wrap(err, "YAML error in Deps sequence")
		}
		depTargets, err := slices.Mapx(node.Content[1:], func(idx int, n *yaml.Node) (Target, error) {
			target, err := YAMLTarget(n)
			return target, errors.Wrapf(err, "deptarget %d", idx)
		})
		if err != nil {
			return nil, errors.Wrap(err, "YAML error in child of Deps node")
		}
		return Deps(target, depTargets...), nil

	case yaml.MappingNode:
		var d struct {
			Pre  []yaml.Node `yaml:"Pre"`
			Post yaml.Node   `yaml:"Post"`
		}
		if err := node.Decode(&d); err != nil {
			return nil, errors.Wrap(err, "YAML error in Deps mapping")
		}
		target, err := YAMLTarget(&d.Post)
		if err != nil {
			return nil, errors.Wrap(err, "YAML error in Deps Post target")
		}
		depTargets, err := slices.Mapx(d.Pre, func(idx int, n yaml.Node) (Target, error) {
			target, err := YAMLTarget(&n)
			return target, errors.Wrapf(err, "deptarget %d", idx)
		})
		if err != nil {
			return nil, errors.Wrap(err, "YAML error in a Deps Pre target")
		}
		return Deps(target, depTargets...), nil

	default:
		return nil, fmt.Errorf("got node kind %v, want %v or %v", node.Kind, yaml.SequenceNode, yaml.MappingNode)
	}
}

func init() {
	RegisterYAMLTarget("Deps", depsDecoder)
}
