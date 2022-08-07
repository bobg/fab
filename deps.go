package fab

import (
	"context"

	"golang.org/x/sync/errgroup"
)

func Deps(ctx context.Context, targets ...Target) error {
	if len(targets) == 0 {
		return nil
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, target := range targets {
		target := target // Go loop-var pitfall
		g.Go(func() error { return Once(ctx, target) })
	}
	return g.Wait()
}
