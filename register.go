package fab

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/bobg/go-generics/v2/maps"
)

// RegisterTarget places a target in the registry with a given name and doc string.
func RegisterTarget(name, doc string, target Target) (Target, error) {
	addr, err := targetAddr(target)
	if err != nil {
		return nil, err
	}

	tuple := targetRegistryTuple{target: target, name: name, doc: doc}

	targetRegistryMu.Lock()
	targetRegistryByName[name] = tuple
	targetRegistryByAddr[addr] = tuple
	targetRegistryMu.Unlock()

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

var (
	targetRegistryMu     sync.Mutex // protects both maps
	targetRegistryByName = make(map[string]targetRegistryTuple)
	targetRegistryByAddr = make(map[uintptr]targetRegistryTuple)
)

func resetRegistry() {
	targetRegistryMu.Lock()
	targetRegistryByName = make(map[string]targetRegistryTuple)
	targetRegistryByAddr = make(map[uintptr]targetRegistryTuple)
	targetRegistryMu.Unlock()
}

// RegistryNames returns the names in the target registry.
func RegistryNames() []string {
	targetRegistryMu.Lock()
	keys := maps.Keys(targetRegistryByName)
	targetRegistryMu.Unlock()
	sort.Strings(keys)
	return keys
}

// RegistryTarget returns the target in the registry with the given name,
// and its doc string.
func RegistryTarget(name string) (Target, string) {
	targetRegistryMu.Lock()
	tuple := targetRegistryByName[name]
	targetRegistryMu.Unlock()
	return tuple.target, tuple.doc
}

// Describe describes a target.
// The description is the target's name in the registry,
// if it has one
// (i.e., if the target was registered with [RegisterTarget]),
// otherwise it's "unnamed X"
// where X is the result of calling the target's Desc method.
func Describe(target Target) string {
	addr, err := targetAddr(target)
	if err == nil { // sic
		targetRegistryMu.Lock()
		tuple, ok := targetRegistryByAddr[addr]
		targetRegistryMu.Unlock()
		if ok {
			return tuple.name
		}
	}

	return "unnamed " + target.Desc()
}
