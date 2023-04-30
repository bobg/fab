package fab

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/bobg/go-generics/v2/maps"
)

// RegisterTarget places a target in the registry with a given name and doc string.
func (con *Controller) RegisterTarget(name, doc string, target Target) (Target, error) {
	addr, err := targetAddr(target)
	if err != nil {
		return nil, err
	}

	tuple := targetRegistryTuple{target: target, name: name, doc: doc}

	con.mu.Lock()
	con.targetsByName[name] = tuple
	con.targetsByAddr[addr] = tuple
	con.mu.Unlock()

	return target, nil
}

func targetAddr(target Target) (uintptr, error) {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer {
		return 0, fmt.Errorf("got type %T for target, want a pointer type", target)
	}
	return uintptr(v.UnsafePointer()), nil
}

type targetRegistryTuple struct {
	target    Target
	name, doc string
}

// RegistryNames returns the names in the target registry.
func (con *Controller) RegistryNames() []string {
	con.mu.Lock()
	keys := maps.Keys(con.targetsByName)
	con.mu.Unlock()
	sort.Strings(keys)
	return keys
}

// RegistryTarget returns the target in the registry with the given name,
// and its doc string.
func (con *Controller) RegistryTarget(name string) (Target, string) {
	con.mu.Lock()
	tuple := con.targetsByName[name]
	con.mu.Unlock()
	return tuple.target, tuple.doc
}

// Describe describes a target.
// The description is the target's name in the registry,
// if it has one
// (i.e., if the target was registered with [RegisterTarget]),
// otherwise it's "unnamed X"
// where X is the result of calling the target's Desc method.
func (con *Controller) Describe(target Target) string {
	addr, err := targetAddr(target)
	if err == nil { // sic
		con.mu.Lock()
		tuple, ok := con.targetsByAddr[addr]
		con.mu.Unlock()
		if ok {
			return tuple.name
		}
	}

	return "unnamed " + target.Desc()
}
