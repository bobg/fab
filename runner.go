package fab

import (
	"context"
	"sync"

	"go.uber.org/multierr"
)

type Runner struct {
	mu  sync.Mutex // protects ran
	ran map[string]*outcome
}

func NewRunner() *Runner {
	return &Runner{ran: make(map[string]*outcome)}
}

type outcome struct {
	g   *gate
	err error
}

func (r *Runner) Run(ctx context.Context, targets ...Target) error {
	if len(targets) == 0 {
		return nil
	}

	var (
		wg   sync.WaitGroup
		errs = make([]error, len(targets))
	)
	for i, target := range targets {
		i, target := i, target // Go loop-var pitfall
		wg.Add(1)
		go func() {
			defer wg.Done()

			id := target.ID()

			r.mu.Lock()
			o, ok := r.ran[id]
			if !ok {
				o = &outcome{g: newGate(false)}
				r.ran[id] = o
			}
			r.mu.Unlock()

			if ok {
				o.g.wait()
				errs[i] = o.err
			} else {
				errs[i] = target.Run(ctx)
				o.err = errs[i]
				o.g.set(true)
			}
		}()
	}

	wg.Wait()

	return multierr.Combine(errs...)
}
