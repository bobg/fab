package fab

import "context"

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

	// Name is a unique name for the target.
	// Each instance of each Target must have a persistent, unique name.
	// You can embed a *Namer in your concrete type to achieve this.
	Name() string

	// SetName sets the name of this target.
	// The name must be unique across all targets.
	// You can embed a *Namer in your concrete type to help with this.
	SetName(string)
}
