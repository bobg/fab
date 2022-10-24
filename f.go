package fab

import "context"

// F produces a target whose Run function invokes the given function.
func F(f func(context.Context) error) Target {
	return &ftarget{Namer: NewNamer("f"), f: f}
}

type ftarget struct {
	*Namer
	f func(context.Context) error
}

var _ Target = &ftarget{}

// Run implements Target.Run.
func (f *ftarget) Run(ctx context.Context) error {
	return f.f(ctx)
}
