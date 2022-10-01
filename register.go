package fab

import (
	"sort"
	"sync"

	"github.com/bobg/go-generics/maps"
)

// Register places a target in the registry with a given name.
func Register(name, doc string, target Target) Target {
	target.SetName(name)
	registryMu.Lock()
	registry[name] = targetDocPair{target: target, doc: doc}
	registryMu.Unlock()
	return target
}

type targetDocPair struct {
	target Target
	doc    string
}

var (
	registryMu sync.Mutex // protects registry
	registry   = make(map[string]targetDocPair)
)

// RegistryNames returns the names in the registry.
func RegistryNames() []string {
	registryMu.Lock()
	keys := maps.Keys(registry)
	registryMu.Unlock()
	sort.Strings(keys)
	return keys
}

// RegistryTarget returns the target in the registry with the given name.
func RegistryTarget(name string) (Target, string) {
	registryMu.Lock()
	pair := registry[name]
	registryMu.Unlock()
	return pair.target, pair.doc
}
