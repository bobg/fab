package fab

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bobg/errors"
)

// Runner is an object that knows how to run Targets
// without ever running the same Target twice.
//
// A zero runner is not usable. Use NewRunner to obtain one instead.
type Runner struct {
	depth int32

	mu  sync.Mutex // protects ran and Indentf
	ran map[uintptr]*outcome
}

// NewRunner produces a new Runner.
func NewRunner() *Runner {
	return &Runner{ran: make(map[uintptr]*outcome)}
}

type outcome struct {
	g   *gate
	err error
}

// Run runs the given targets, skipping any that have already run.
//
// A Runner remembers which targets it has already run
// (whether in this call or any previous call to Run).
//
// The targets are executed concurrently.
// A separate goroutine is created for each one passed to Run.
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
// and looked up in the HashDB.
// If it's found,
// the target's outputs are already up to date
// and its Run method can be skipped.
// Otherwise, Run is invoked and
// (if it succeeds)
// a new hash is computed for the target
// and added to the HashDB.
//
// This function waits for all goroutines to complete.
// The return value may be an accumulation of multiple errors
// produced with [errors.Join].
//
// The runner is added to the context with WithRunner
// and can be retrieved with GetRunner.
// Calls to Run
// (the global function, not the Runner.Run method)
// will use it instead of DefaultRunner
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
		addr, err := targetAddr(target)
		if err != nil {
			errs[i] = err
			continue
		}

		i, target := i, target // Go loop-var pitfall

		wg.Add(1)
		go func() {
			defer wg.Done()

			r.mu.Lock()
			o, ok := r.ran[addr]
			if !ok {
				o = &outcome{g: newGate(false)}
				r.ran[addr] = o
			}
			r.mu.Unlock()

			if ok {
				// This target was launched in a different goroutine.
				// Wait for it to produce a result.
				o.g.wait()
				errs[i] = o.err
			} else {
				// This target was not previously launched,
				// so run it and then open its "outcome gate."
				err := r.runTarget(ctx, db, target)
				errs[i] = err
				o.err = err
				o.g.set(true)
			}
		}()
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (r *Runner) runTarget(ctx context.Context, db HashDB, target Target) error {
	var (
		verbose = GetVerbose(ctx)
		force   = GetForce(ctx)
	)

	var ht HashTarget
	if db != nil {
		ht, _ = target.(HashTarget)
		if ht != nil && !force {
			h, err := ht.Hash(ctx)
			if err != nil {
				return errors.Wrapf(err, "computing hash for %s", Describe(target))
			}
			has, err := db.Has(ctx, h)
			if err != nil {
				return errors.Wrapf(err, "checking hash db for hash of %s", Describe(target))
			}
			if has {
				if verbose {
					r.Indentf("%s is up to date", Describe(target))
				}
				return nil
			}
		}
	}

	if verbose {
		r.Indentf("Running %s", Describe(target))
	}

	err := target.Run(ctx)
	if err != nil {
		return errors.Wrapf(err, "running %s", Describe(target))
	}

	if ht != nil {
		h, err := ht.Hash(ctx)
		if err != nil {
			return errors.Wrapf(err, "computing new updatedhash for %s", Describe(target))
		}
		err = db.Add(ctx, h)
		if err != nil {
			return errors.Wrap(err, "updating hash db")
		}
	}

	return nil
}

// Indentf formats and prints its arguments
// with leading indentation based on the nesting depth of the Runner.
// The nesting depth increases with each call to Runner.Run
// and decreases at the end of the call.
//
// A newline is added to the end of the string if one is not already there.
func (r *Runner) Indentf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	if depth := atomic.LoadInt32(&r.depth); depth > 0 {
		fmt.Print(strings.Repeat("  ", int(depth)))
	}
	fmt.Printf(format, args...)
}

// DefaultRunner is a Runner used by default in Run.
var DefaultRunner = NewRunner()

// Run runs the given targets with a Runner.
// If `ctx` contains a Runner
// (e.g., because this call is nested inside a pending call to Runner.Run
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

// Indentf calls Runner.Indent with the given format and args
// if a Runner can be found in the given context.
// If one cannot, then the formatted string is simply printed
// (with a trailing newline added if needed).
func Indentf(ctx context.Context, format string, args ...any) {
	if runner := GetRunner(ctx); runner != nil {
		runner.Indentf(format, args...)
	} else {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		fmt.Printf(format, args...)
	}
}
