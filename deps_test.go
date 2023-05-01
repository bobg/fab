package fab

import (
	"context"
	"sync"
	"testing"
)

func TestDeps(t *testing.T) {
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

	con := NewController("")

	err := con.Run(context.Background(), Deps(post, pre1, pre2))
	if err != nil {
		t.Fatal(err)
	}

	if !ranpost {
		t.Fatal("somehow post did not run")
	}
}
