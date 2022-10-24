package fab

import "context"

// Seq produces a target that runs a collection of targets in sequence.
// Its Run method exits early when a target in the sequence fails.
func Seq(targets ...Target) Target {
	return &seq{Namer: NewNamer("seq"), targets: targets}
}

type seq struct {
	*Namer
	targets []Target
}

var _ Target = &seq{}

// Run implements Target.Run.
func (s *seq) Run(ctx context.Context) error {
	for _, t := range s.targets {
		if err := Run(ctx, t); err != nil {
			return err
		}
	}
	return nil
}
