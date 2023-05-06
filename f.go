package fab

import "context"

// F produces a target whose Run function invokes the given function.
// It is not JSON-encodable,
// so it should not be used as the subtarget in a [Files] rule.
//
// The behavior of F does not change according to [GetDryRun].
// It's up to the function you pass to F to detect dry-run mode
// and avoid adding, removing, or updating files,
// or making other state-altering changes.
func F(f func(context.Context, *Controller) error) Target {
	return &ftarget{f: f}
}

type ftarget struct {
	f func(context.Context, *Controller) error
}

var _ Target = &ftarget{}

// Run implements Target.Run.
func (f *ftarget) Run(ctx context.Context, con *Controller) error {
	return f.f(ctx, con)
}

// Desc implements Target.Desc.
func (*ftarget) Desc() string {
	return "F"
}
