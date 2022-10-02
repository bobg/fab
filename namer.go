package fab

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Namer aids in giving unique names to objects.
// It implements the Name and SetName methods needed by Target.
// Most Target implementations will want to embed a *Namer
// and initialize it with a call to NewNamer.
type Namer struct {
	mu         sync.Mutex
	base, name string
}

// NewNamer creates a Namer with the given "base" name.
func NewNamer(base string) *Namer {
	return &Namer{base: base}
}

// Name returns the name in the Namer.
// This is the string from the latest call to SetName.
// If SetName has never been called,
// then the Namer's name is first set to its "base" name
// (which was set with NewNamer)
// passed through a call to Unique.
// E.g. NewNamer("foo").Name() might produce "foo-17".
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

// SetName sets the name in a Namer.
func (n *Namer) SetName(name string) {
	n.mu.Lock()
	n.name = name
	n.mu.Unlock()
}

var uniquecounter uint32

// Unique produces a unique string by appending a unique counter value to the given prefix.
// For example, Unique("foo") might produce "foo-17".
func Unique(s string) string {
	return fmt.Sprintf("%s-%d", s, atomic.AddUint32(&uniquecounter, 1))
}
