package fab

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
// If the Runner has never yet run the target,
// it does so, and caches the result (error or no error).
// If the target did already run, the cached error value is used.
// If another goroutine concurrently requests the same target,
// it blocks until the first one completes,
// then uses the first one's result.
//
// This function waits for all goroutines to complete.
// The return value may be an accumulation of multiple errors
// produced with [errors.Join].
//
// The runner is added to the context with [WithRunner]
// and can be retrieved with [GetRunner].
// Calls to [Run]
// will use it instead of [DefaultRunner]
// by finding it in the context.
func (r *Runner) Run(ctx context.Context, targets ...Target) error {
	if len(targets) == 0 {
		return nil
	}

	ctx = WithRunner(ctx, r)

	atomic.AddInt32(&r.depth, 1)
	defer atomic.AddInt32(&r.depth, -1)

	var (
		verbose = GetVerbose(ctx)
		errs    = make([]error, len(targets))
		wg      sync.WaitGroup
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
				if verbose {
					r.Indentf("Running %s", Describe(target))
				}
				err := target.Execute(ctx)
				if err != nil {
					err = errors.Wrapf(err, "running %s", Describe(target))
				}
				errs[i] = err
				o.err = err
				o.g.set(true)
			}
		}()
	}

	wg.Wait()

	return errors.Join(errs...)
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
// (e.g., because this call is nested inside a pending call to [Runner.Run]
// and the context has been decorated using [WithRunner])
// then it uses that Runner,
// otherwise it uses [DefaultRunner].
//
// A given Runner will not run the same target more than once.
// See [Runner.Run].
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

// IndentingCopier creates an [io.Writer] that copies its data to an underlying writer,
// indenting each line according to the indentation depth of the [Runner] in the given context.
// After indentation,
// each line additionally gets any prefix specified in `prefix`.
//
// The wrapper converts \r\n to \n, and bare \r to \n.
func IndentingCopier(ctx context.Context, w io.Writer, prefix string) io.Writer {
	runner := GetRunner(ctx)
	if runner == nil {
		runner = DefaultRunner
	}
	depth := atomic.LoadInt32(&runner.depth)

	return &indentingCopier{
		w:      bufio.NewWriter(w),
		indent: strings.Repeat("  ", int(depth)) + prefix,
		bol:    true,
	}
}

type indentingCopier struct {
	w          *bufio.Writer
	indent     string
	bol, sawcr bool
}

func (c *indentingCopier) Write(buf []byte) (int, error) {
	var n int

	for _, b := range buf {
		switch b {
		case '\n':
			if err := c.newline(); err != nil {
				return n, err
			}

		case '\r':
			if c.sawcr {
				if err := c.newline(); err != nil {
					return n, err
				}
			}
			c.sawcr = true

		default:
			if c.sawcr {
				if err := c.newline(); err != nil {
					return n, err
				}
			}
			if c.bol {
				_, err := c.w.WriteString(c.indent)
				if err != nil {
					return n, err
				}
			}
			c.bol = false
			if err := c.w.WriteByte(b); err != nil {
				return n, err
			}
		}
		n++
	}

	err := c.w.Flush()
	return n, err
}

func (c *indentingCopier) newline() error {
	if err := c.w.WriteByte('\n'); err != nil {
		return err
	}
	c.bol = true
	c.sawcr = false
	return nil
}
