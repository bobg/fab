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
	ID() string
}

var idcounter uint32

// ID produces an ID string by appending a unique counter value to the given prefix.
func ID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, atomic.AddUint32(&idcounter, 1))
}
