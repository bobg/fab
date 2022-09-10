package fab

import (
	"context"
	"fmt"
	"sync/atomic"
)

// Target is the interface that Fab targets must implement.
type Target interface {
	// Run invokes the target's logic.
	//
	// Callers generally should not invoke a target's Run method.
	// Instead, pass the target to a Runner's Run method,
	// or to the global Run function.
	// That will handle concurrency properly
	// and make sure that the target is not rerun
	// when it doesn't need to be.
	Run(context.Context) error

	// ID is a unique ID for the target.
	// Each instance of each Target must have a persistent, unique ID.
	// The ID function can help with that.
	ID() string
}

var idcounter uint32

// ID produces an ID string by appending a unique counter value to the given prefix.
// For example, ID("Foo") might produce "Foo-17".
func ID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, atomic.AddUint32(&idcounter, 1))
}

// Name returns a name for `target`.
// Normally this is just target.ID().
// But Register can wrap a target with a human-friendlier name
// and in that case Name will return that string instead.
func Name(target Target) string {
	if w, ok := target.(nameWrapper); ok {
		return w.name()
	}
	return target.ID()
}

type nameWrapper interface {
	Target
	name() string
}

type namedTarget struct {
	Target
	n string
}

func (t namedTarget) name() string { return t.n }

type namedHashTarget struct {
	HashTarget
	n string
}

func (t namedHashTarget) name() string { return t.n }
