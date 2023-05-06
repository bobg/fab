package fab

import "context"

type (
	dryrunKeyType  struct{}
	forceKeyType   struct{}
	hashDBKeyType  struct{}
	verboseKeyType struct{}
	argsKeyType    struct{}
)

// WithDryRun decorates a context with the value of a "dryrun" boolean.
// Retrieve it with [GetDryRun].
func WithDryRun(ctx context.Context, dryrun bool) context.Context {
	return context.WithValue(ctx, dryrunKeyType{}, dryrun)
}

// GetDryRun returns the value of the dryrun boolean added to `ctx` with [WithDryRun].
// The default, if WithDryRun was not used, is false.
func GetDryRun(ctx context.Context) bool {
	val, _ := ctx.Value(dryrunKeyType{}).(bool)
	return val
}

// WithForce decorates a context with the value of a "force" boolean.
// Retrieve it with [GetForce].
func WithForce(ctx context.Context, force bool) context.Context {
	return context.WithValue(ctx, forceKeyType{}, force)
}

// GetForce returns the value of the force boolean added to `ctx` with [WithForce].
// The default, if WithForce was not used, is false.
func GetForce(ctx context.Context) bool {
	val, _ := ctx.Value(forceKeyType{}).(bool)
	return val
}

// WithHashDB decorates a context with a [HashDB].
// Retrieve it with [GetHashDB].
func WithHashDB(ctx context.Context, db HashDB) context.Context {
	return context.WithValue(ctx, hashDBKeyType{}, db)
}

// GetHashDB returns the value of the HashDB added to `ctx` with [WithHashDB].
// The default, if WithHashDB was not used, is nil.
func GetHashDB(ctx context.Context) HashDB {
	db, _ := ctx.Value(hashDBKeyType{}).(HashDB)
	return db
}

// WithVerbose decorates a context with the value of a "verbose" boolean.
// Retrieve it with [GetVerbose].
func WithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKeyType{}, verbose)
}

// GetVerbose returns the value of the verbose boolean added to `ctx` with [WithVerbose].
// The default, if WithVerbose was not used, is false.
func GetVerbose(ctx context.Context) bool {
	val, _ := ctx.Value(verboseKeyType{}).(bool)
	return val
}

// WithArgs decorates a context with a list of arguments as a slice of strings.
// Retrieve it with [GetArgs].
func WithArgs(ctx context.Context, args ...string) context.Context {
	return context.WithValue(ctx, argsKeyType{}, args)
}

// GetArgs returns the list of arguments added to `ctx` with [WithArgs].
// The default, if WithArgs was not used, is nil.
func GetArgs(ctx context.Context) []string {
	val, _ := ctx.Value(argsKeyType{}).([]string)
	return val
}
