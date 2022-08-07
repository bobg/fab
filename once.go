package fab

import "context"

func Once(ctx context.Context, t Target) error {
	var err error
	t.Once().Do(func() {
		err = t.Run(ctx)
	})
	return err
}
