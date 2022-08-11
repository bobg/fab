package fab

import (
	"context"
	"sync"

	"go.uber.org/multierr"
)

// Runner is an object that knows how to run Targets
// without ever running the same Target twice.
//
// A zero runner is not usable. Use NewRunner to obtain one instead.
type Runner struct {
	mu  sync.Mutex // protects ran
	ran map[string]*outcome
}

// NewRunner produces a new Runner.
func NewRunner() *Runner {
	return &Runner{ran: make(map[string]*outcome)}
}

type outcome struct {
	g   *gate
	err error
}

// Run runs the given targets, skipping any that have already run.
//
// THEORY OF OPERATION
//
// A Runner remembers which targets it has already run
// (whether in this call or any previous call to Run),
// distinguishing targets by their ID() values.
//
// A separate goroutine is created for each Target passed to Run.
// If the Runner has not already called the Target's Run method,
// it does so, and caches the result (error or no error).
// If the Target did already run, the cached error value is used.
// If another goroutine concurrently requests the same Target,
// it blocks until the first one completes,
// then uses the first one's result.
//
// This function waits for all goroutines to complete.
// The return value may be an accumulation of multiple errors.
// These can be retrieved with [multierr.Errors].
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
