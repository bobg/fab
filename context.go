package fab

import "context"

type (
	verboseKeyType struct{}
	dirKeyType     struct{}
)

// WithVerbose decorates a context with the value of a "verbose" boolean.
// Retrieve it with Verbose.
func WithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKeyType{}, verbose)
}

// Verbose returns the value of the verbose boolean added to `ctx` with WithVerbose.
// The default, if WithVerbose was not used, is false.
func Verbose(ctx context.Context) bool {
	val, _ := ctx.Value(verboseKeyType{}).(bool)
	return val
}

// WithDir decorates a context with the path to a directory in which rules should run.
// Retrieve it with Dir.
func WithDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, dirKeyType{}, dir)
}

// Dir returns the value of the directory added to `ctx` with WithDir.
// The default, if WithDir was not used, is the empty string.
func Dir(ctx context.Context) string {
	val, _ := ctx.Value(dirKeyType{}).(string)
	return val
}
