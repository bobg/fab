package fab

import "context"

type (
	verboseKeyType struct{}
	dirKeyType     struct{}
	runnerKeyType  struct{}
)

// WithVerbose decorates a context with the value of a "verbose" boolean.
// Retrieve it with GetVerbose.
func WithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKeyType{}, verbose)
}

// GetVerbose returns the value of the verbose boolean added to `ctx` with WithVerbose.
// The default, if WithVerbose was not used, is false.
func GetVerbose(ctx context.Context) bool {
	val, _ := ctx.Value(verboseKeyType{}).(bool)
	return val
}

// WithDir decorates a context with the path to a directory in which rules should run.
// Retrieve it with GetDir.
func WithDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, dirKeyType{}, dir)
}

// GetDir returns the value of the directory added to `ctx` with WithDir.
// The default, if WithDir was not used, is the empty string.
func GetDir(ctx context.Context) string {
	val, _ := ctx.Value(dirKeyType{}).(string)
	return val
}

// WithRunner decorates a context with a Runner.
// Retrieve it with GetRunner.
func WithRunner(ctx context.Context, r *Runner) context.Context {
	return context.WithValue(ctx, runnerKeyType{}, r)
}

// GetRunner returns the value of the Runner added to `ctx` with WithRunner.
// The default, if WithRunner was not used, is nil.
func GetRunner(ctx context.Context) *Runner {
	r, _ := ctx.Value(runnerKeyType{}).(*Runner)
	return r
}
