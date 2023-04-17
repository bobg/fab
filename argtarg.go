package fab

import "context"

// ArgTarget produces a target with associated arguments
// as a list of strings,
// suitable for parsing with the [flag] package.
// When the target runs,
// its arguments are available from the context using [GetArgs].
func ArgTarget(target Target, args []string) Target {
	return &argTarget{
		Namer:  NewNamer("args-" + target.Name()),
		target: target,
		args:   args,
	}
}

type argTarget struct {
	*Namer
	target Target
	args   []string
}

var _ Target = &argTarget{}

func (a *argTarget) Run(ctx context.Context) error {
	ctx = WithArgs(ctx, a.args)
	return a.target.Run(ctx)
}
