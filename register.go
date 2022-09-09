package fab

import (
	"sort"
	"sync"

	"github.com/bobg/go-generics/maps"
)

// Register places a target in the registry with a given name.
func Register(name string, target Target) {
	registryMu.Lock()
	registry[name] = target
	registryMu.Unlock()
}

var (
	registryMu sync.Mutex // protects registry
	registry   = map[string]Target{}
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
func RegistryTarget(name string) Target {
	registryMu.Lock()
	defer registryMu.Unlock()
	return registry[name]
}
