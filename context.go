package fab

import "context"

type (
	forceKeyType   struct{}
	hashDBKeyType  struct{}
	namesKeyType   struct{}
	runnerKeyType  struct{}
	verboseKeyType struct{}
)

// WithForce decorates a context with the value of a "force" boolean.
// Retrieve it with GetForce.
func WithForce(ctx context.Context, force bool) context.Context {
	return context.WithValue(ctx, forceKeyType{}, force)
}

// GetForce returns the value of the force boolean added to `ctx` with WithForce.
// The default, if WithForce was not used, is false.
func GetForce(ctx context.Context) bool {
	val, _ := ctx.Value(forceKeyType{}).(bool)
	return val
}

// WithHashDB decorates a context with a HashDB.
// Retrieve it with GetHashDB.
func WithHashDB(ctx context.Context, db HashDB) context.Context {
	return context.WithValue(ctx, hashDBKeyType{}, db)
}

// GetHashDB returns the value of the HashDB added to `ctx` with WithHashDB.
// The default, if WithHashDB was not used, is nil.
func GetHashDB(ctx context.Context) HashDB {
	db, _ := ctx.Value(hashDBKeyType{}).(HashDB)
	return db
}

// WithNames decorates a context with a map[uintptr]string.
// The keys are the addresses of Target objects
// and the values are their "pretty names."
// When available, these are shown to the user at runtime
// in preference to the value of Target.ID.
func WithNames(ctx context.Context, names map[uintptr]string) context.Context {
	return context.WithValue(ctx, namesKeyType{}, names)
}

// GetNames returns the map[uintptr]string added to `ctx` with WithNames.
// The default, if WithNames was not used, is nil.
func GetNames(ctx context.Context) map[uintptr]string {
	names, _ := ctx.Value(namesKeyType{}).(map[uintptr]string)
	return names
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
