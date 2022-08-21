package fab

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

// Runner is an object that knows how to run Targets
// without ever running the same Target twice.
//
// A zero runner is not usable. Use NewRunner to obtain one instead.
type Runner struct {
	depth int32

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
// distinguishing them by their ID() values.
//
// A separate goroutine is created for each Target passed to Run.
// If the Runner has never yet run the Target,
// it does so, and caches the result (error or no error).
// If the Target did already run, the cached error value is used.
// If another goroutine concurrently requests the same Target,
// it blocks until the first one completes,
// then uses the first one's result.
//
// As a special case,
// if the Target is a HashTarget
// and there is a HashDB attached to the context,
// then the HashTarget's hash is computed
// and sought in the HashDB.
// If it's found,
// the target's outputs are already up to date
// and its Run method can be skipped.
// Otherwise, Run is invoked and
// (if it succeeds)
// a new hash is computed for the target
// and added to the HashDB.
//
// This function waits for all goroutines to complete.
// The return value may be an accumulation of multiple errors.
// These can be retrieved with [multierr.Errors].
//
// The runner is added to the context with WithRunner
// and can be retrieved with GetRunner.
// Calls to Run
// (the global function, not the Runner method)
// will use this Runner instead of DefaultRunner
// by finding it in the context.
func (r *Runner) Run(ctx context.Context, targets ...Target) error {
	if len(targets) == 0 {
		return nil
	}

	ctx = WithRunner(ctx, r)

	atomic.AddInt32(&r.depth, 1)
	defer atomic.AddInt32(&r.depth, -1)

	var (
		db   = GetHashDB(ctx)
		errs = make([]error, len(targets))
		wg   sync.WaitGroup
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
				err := r.runTarget(ctx, db, target)
				errs[i] = err
				o.err = err
				o.g.set(true)
			}
		}()
	}

	wg.Wait()

	return multierr.Combine(errs...)
}

func (r *Runner) runTarget(ctx context.Context, db HashDB, target Target) error {
	verbose := GetVerbose(ctx)

	var ht HashTarget
	if db != nil {
		ht, _ = target.(HashTarget)
		if ht != nil {
			h, err := ht.Hash(ctx)
			if err != nil {
				return errors.Wrapf(err, "computing hash for %s", Name(ctx, target))
			}
			has, err := db.Has(ctx, h)
			if err != nil {
				return errors.Wrapf(err, "checking hash db for hash of %s", Name(ctx, target))
			}
			if has {
				if verbose {
					r.Indentf("%s is up to date", Name(ctx, target))
				}
				return nil
			}
		}
	}

	if verbose {
		r.Indentf("Running %s", Name(ctx, target))
	}

	err := target.Run(ctx)
	if err != nil {
		return errors.Wrapf(err, "running %s", Name(ctx, target))
	}

	if ht != nil {
		h, err := ht.Hash(ctx)
		if err != nil {
			return errors.Wrapf(err, "computing updated hash for %s", Name(ctx, target))
		}
		err = db.Add(ctx, h)
		if err != nil {
			return errors.Wrap(err, "updating hash db")
		}
	}

	return nil
}

func (r *Runner) Indentf(format string, args ...any) {
	if depth := atomic.LoadInt32(&r.depth); depth > 0 {
		fmt.Print(strings.Repeat("  ", int(depth)))
	}
	fmt.Printf(format, args...)
	fmt.Println("")
}

// DefaultRunner is a Runner used by default in Run.
var DefaultRunner = NewRunner()

// Run runs the given targets with a Runner.
// If `ctx` contains a Runner
// (e.g., because this is a recursive call
// and the context has been decorated using WithRunner)
// then it uses that Runner,
// otherwise it uses DefaultRunner.
func Run(ctx context.Context, targets ...Target) error {
	runner := GetRunner(ctx)
	if runner == nil {
		runner = DefaultRunner
	}
	return runner.Run(ctx, targets...)
}
