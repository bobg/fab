package fab

import (
	"context"
	"testing"
)

func TestWithForce(t *testing.T) {
	ctx := context.Background()
	got := GetForce(ctx)
	if got {
		t.Error("got true, want false [1]")
	}
	ctx = WithForce(ctx, false)
	got = GetForce(ctx)
	if got {
		t.Error("got true, want false [2]")
	}
	ctx = WithForce(ctx, true)
	got = GetForce(ctx)
	if !got {
		t.Error("got false, want true")
	}
}

func TestWithVerbose(t *testing.T) {
	ctx := context.Background()
	got := GetVerbose(ctx)
	if got {
		t.Error("got true, want false [1]")
	}
	ctx = WithVerbose(ctx, false)
	got = GetVerbose(ctx)
	if got {
		t.Error("got true, want false [2]")
	}
	ctx = WithVerbose(ctx, true)
	got = GetVerbose(ctx)
	if !got {
		t.Error("got false, want true")
	}
}
