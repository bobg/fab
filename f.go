package fab

import "context"

// F produces a target whose Execute function invokes the given function.
func F(f func(context.Context) error) Target {
	return &ftarget{f: f}
}

type ftarget struct {
	f func(context.Context) error
}

var _ Target = &ftarget{}

// Execute implements Target.Execute.
func (f *ftarget) Execute(ctx context.Context) error {
	return f.f(ctx)
}

// Desc implements Target.Desc.
func (*ftarget) Desc() string {
	return "F"
}
