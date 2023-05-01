package fab

import "context"

// Target is the interface that Fab targets must implement.
type Target interface {
	// Run invokes the target's logic.
	// It receives a context object and the [Controller] running this target as arguments.
	//
	// Callers should not invoke a target's Run method directly.
	// Instead, pass the target to a Controller's Run method.
	// That will handle concurrency properly
	// and make sure that the target is not rerun
	// when it doesn't need to be.
	Run(context.Context, *Controller) error

	// Desc produces a short descriptive string for this target.
	// It is used by [Describe] when the target is not found in the target registry.
	Desc() string
}
