package fab

import "context"

type verboseKeyType struct{}

func WithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKeyType{}, verbose)
}

func Verbose(ctx context.Context) bool {
	val, _ := ctx.Value(verboseKeyType{}).(bool)
	return val
}
