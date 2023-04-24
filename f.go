package fab

import "context"

// F produces a target whose Run function invokes the given function.
func F(f func(context.Context) error) Target {
	return &ftarget{f: f}
}

type ftarget struct {
	f func(context.Context) error
}

var _ Target = &ftarget{}

// Run implements Target.Run.
func (f *ftarget) Run(ctx context.Context) error {
	return f.f(ctx)
}

func (*ftarget) Desc() string {
	return "F"
}
