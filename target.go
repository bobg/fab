package fab

import (
	"context"
	"fmt"
	"sync/atomic"
)

// Target is the interface that Fab targets must implement.
type Target interface {
	// Run invokes the target's logic.
	Run(context.Context) error

	// ID is a unique ID for the target.
	ID() string
}

var idcounter uint32

// ID produces an ID string by appending a unique counter value to the given prefix.
func ID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, atomic.AddUint32(&idcounter, 1))
}
