package fab

import (
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func Yaml(r io.Reader, f func(name, comment, tag string, conf map[string]any) error) error {
	var (
		dec  = yaml.NewDecoder(r)
		node yaml.Node
	)
	if err := dec.Decode(&node); err != nil {
		return errors.Wrap(err, "reading input")
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("parse error: got yaml node kind %d, want %d", node.Kind, yaml.MappingNode)
	}
	if len(node.Content)%2 != 0 {
		return fmt.Errorf("parse error: top node has %d children, want an even number", len(node.Content))
	}
	for i := 0; i < len(node.Content); i += 2 {
		tag := node.Content[i+1].ShortTag()
		if strings.HasPrefix(tag, "!!") {
			// Skip nodes without user-defined type tags.
			continue
		}
		tag = strings.TrimPrefix(tag, "!")

		comment = node.Content[i].HeadComment

		if node.Content[i].Kind != yaml.ScalarNode {
			return fmt.Errorf("parse error: yaml subnode %d has kind %d, want %d", node.Content[i].Kind, yaml.ScalarNode)
		}
		name = node.Content[i].Value

		var m map[conf]Tagged
		if err = node.Content[i+1].Decode(&m); err != nil {
			return errors.Wrapf(err, "decoding subnode %d", i+1)
		}

		simplified := simplify(m)

		if err = f(name, comment, tag, simplified); err != nil {
			return errors.Wrapf(err, "handling callback for %s, yaml subnode %d", name, i+1)
		}
	}

	return nil
}

// xxx topological sort (if a depends on b, register b before a)
// (solution: a wrapper type implementing Target that defers resolution of a given name)
func YamlRegister(r io.Reader) error {
	return Yaml(r, func(name, comment, tag string, conf map[string]any) error {
		f, ok := GetFactory(tag)
		if !ok {
			return fmt.Errorf("unknown type tag %s", tag)
		}
		target, err := f(name, conf)
		if err != nil {
			return errors.Wrapf(err, "in %s factory", tag)
		}
		Register(name, comment, target)
		return nil
	})
}

type Tagged struct {
	Tag string
	Val any
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (t *Tagged) UnmarshalYAML(node *yaml.Node) error {
	t.Tag = node.ShortTag()

	switch node.Kind {
	case yaml.SequenceNode:
		var slice []Tagged
		err := node.Decode(&slice)
		t.Val = slice
		return err

	case yaml.MappingNode:
		var m map[string]Tagged
		err := node.Decode(&m)
		t.Val = m
		return err
	}

	return node.Decode(&t.Val)
}

func simplifyMap(in map[string]Tagged) map[string]any {
	result := make(map[string]any)
	for k, v := range in {
		result[k] = simplify(v)
	}
	return result
}

func simplify(in any) any {
	switch in := in.(type) {
	case Tagged:
		if strings.HasPrefix(in.Tag, "!!") {
			return simplify(in.Val)
		}

	case []Tagged:
		result := make([]any, 0, len(in))
		for _, item := range in {
			result = append(result, simplify(item))
		}
		return result

	case map[string]Tagged:
		return simplifyMap(in)
	}

	return in
}
