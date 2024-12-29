package fab

import (
	"context"
	"reflect"
	"testing"
)

func TestArgTarget(t *testing.T) {
	t.Parallel()

	args := []string{"one", "two", "three"}

	f := F(func(ctx context.Context, _ *Controller) error {
		got := GetArgs(ctx)
		if !reflect.DeepEqual(got, args) {
			t.Errorf("got %v, want %v", got, args)
		}
		return nil
	})

	con := NewController("", nil)
	con.Verbose = true

	var (
		a   = ArgTarget(f, args...)
		ctx = context.Background()
	)

	if err := con.Run(ctx, a); err != nil {
		t.Fatal(err)
	}
}
