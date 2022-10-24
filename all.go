package fab

import "context"

// All produces a target that runs a collection of targets in parallel.
func All(targets ...Target) Target {
	return &all{Namer: NewNamer("all"), targets: targets}
}

type all struct {
	*Namer
	targets []Target
}

var _ Target = &all{}

// Run implements Target.Run.
func (a *all) Run(ctx context.Context) error {
	return Run(ctx, a.targets...)
}
