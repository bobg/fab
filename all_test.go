package fab

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestAll(t *testing.T) {
	var mu sync.Mutex // protects ran1 and ran2 during Run
	var ran1, ran2 bool

	t1 := F(func(_ context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if ran1 {
			return fmt.Errorf("t1 running a second time")
		}
		ran1 = true
		return nil
	})
	t2 := F(func(_ context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if ran2 {
			return fmt.Errorf("t2 running a second time")
		}
		ran2 = true
		return nil
	})
	a := All(t1, t2)
	err := Run(context.Background(), a)
	if err != nil {
		t.Fatal(err)
	}
	if !ran1 {
		t.Error("t1 did not run")
	}
	if !ran2 {
		t.Error("t2 did not run")
	}
}
