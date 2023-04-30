package fab

import "context"

// F produces a target whose Execute function invokes the given function.
// It is not JSON-encodable,
// so it should not be used as the subtarget in a [Files] rule.
func F(f func(context.Context, *Controller) error) Target {
	return &ftarget{f: f}
}

type ftarget struct {
	f func(context.Context, *Controller) error
}

var _ Target = &ftarget{}

// Execute implements Target.Execute.
func (f *ftarget) Execute(ctx context.Context, con *Controller) error {
	return f.f(ctx, con)
}

// Desc implements Target.Desc.
func (*ftarget) Desc() string {
	return "F"
}
