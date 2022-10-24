package fab

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestSeq(t *testing.T) {
	var (
		mu         sync.Mutex // protects ran1 and ran2 during Run
		ran1, ran2 bool

		t1secondTimeErr = errors.New("t1 running a second time")
		ctx             = context.Background()
	)

	t1 := F(func(_ context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if ran1 {
			return t1secondTimeErr
		}
		if ran2 {
			return fmt.Errorf("t2 ran before t1")
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
		if !ran1 {
			return fmt.Errorf("t1 has not run yet")
		}
		ran2 = true
		return nil
	})
	s := Seq(t1, t2)
	err := Run(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	if !ran1 {
		t.Error("t1 did not run")
	}
	if !ran2 {
		t.Error("t2 did not run")
	}

	// Reset ran2 and run s with a new Runner.
	// t1 should error and prevent t2 from running.
	ran2 = false

	r := NewRunner()
	err = r.Run(ctx, s)
	if !errors.Is(err, t1secondTimeErr) {
		t.Errorf("want %s, got %v", t1secondTimeErr, err)
	}
	if ran2 {
		t.Error("t2 ran but should not have")
	}
}
