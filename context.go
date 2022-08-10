package fab

import "context"

type (
	verboseKeyType struct{}
	dirKeyType     struct{}
)

func WithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKeyType{}, verbose)
}

func Verbose(ctx context.Context) bool {
	val, _ := ctx.Value(verboseKeyType{}).(bool)
	return val
}

func WithDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, dirKeyType{}, dir)
}

func Dir(ctx context.Context) string {
	val, _ := ctx.Value(dirKeyType{}).(string)
	return val
}
