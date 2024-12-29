package fab

import (
	"context"
	"sync"
	"testing"
)

func TestDeps(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex // protects ran1, ran2, and ranpost during Run
	var ran1, ran2, ranpost bool

	pre1 := F(func(context.Context, *Controller) error {
		mu.Lock()
		ran1 = true
		mu.Unlock()
		return nil
	})
	pre2 := F(func(context.Context, *Controller) error {
		mu.Lock()
		ran2 = true
		mu.Unlock()
		return nil
	})

	post := F(func(context.Context, *Controller) error {
		mu.Lock()
		defer mu.Unlock()
		if !ran1 || !ran2 {
			t.Errorf("ran1 is %v, ran2 is %v (want true, true)", ran1, ran2)
		}
		ranpost = true
		return nil
	})

	con, err := NewController("")
	if err != nil {
		t.Fatal(err)
	}
	con.Verbose = true

	ctx := context.Background()

	if err = con.Run(ctx, Deps(post, pre1, pre2)); err != nil {
		t.Fatal(err)
	}

	if !ranpost {
		t.Fatal("somehow post did not run")
	}
}
