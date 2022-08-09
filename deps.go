package fab

import (
	"context"
	"sync"

	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
)

func Deps(ctx context.Context, targets ...Target) error {
	if len(targets) == 0 {
		return nil
	}

	var (
		g errgroup.Group

		mu   sync.Mutex // protects errs
		errs error
	)
	for _, target := range targets {
		target := target // Go loop-var pitfall
		g.Go(func() error {
			err := Once(ctx, target)

			mu.Lock()
			errs = multierr.Combine(errs, err)
			mu.Unlock()

			return err
		})
	}
	g.Wait()
	return errs
}
