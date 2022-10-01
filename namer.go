package fab

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type Namer struct {
	mu         sync.Mutex
	base, name string
}

func NewNamer(base string) *Namer {
	return &Namer{base: base}
}

func (n *Namer) Name() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.name == "" {
		if n.base == "" {
			n.name = Unique(n.base)
		} else {
			n.name = Unique("target")
		}
	}
	return n.name
}

func (n *Namer) SetName(name string) {
	n.mu.Lock()
	n.name = name
	n.mu.Unlock()
}

var uniquecounter uint32

// Unique produces a unique string by appending a unique counter value to the given prefix.
// For example, Unique("Foo") might produce "Foo-17".
func Unique(s string) string {
	return fmt.Sprintf("%s-%d", s, atomic.AddUint32(&uniquecounter, 1))
}
