package fab

import (
	"context"
	"reflect"
	"testing"
)

func TestArgTarget(t *testing.T) {
	args := []string{"one", "two", "three"}

	f := F(func(ctx context.Context, _ *Controller) error {
		got := GetArgs(ctx)
		if !reflect.DeepEqual(got, args) {
			t.Errorf("got %v, want %v", got, args)
		}
		return nil
	})
	a := ArgTarget(f, args...)
	con := NewController("")
	if err := con.Run(context.Background(), a); err != nil {
		t.Fatal(err)
	}
}
