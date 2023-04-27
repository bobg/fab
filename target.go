package fab

import "context"

// Target is the interface that Fab targets must implement.
type Target interface {
	// Execute invokes the target's logic.
	//
	// Callers generally should not invoke a target's Execute method.
	// Instead, pass the target to a [Runner]'s Run method,
	// or to the global [Run] function.
	// That will handle concurrency properly
	// and make sure that the target is not rerun
	// when it doesn't need to be.
	Execute(context.Context) error

	// Desc produces a short descriptive string for this target.
	// It is used by [Describe] when the target is not found in the target registry.
	Desc() string
}
