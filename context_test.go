package fab

import (
	"context"
	"testing"
)

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

func TestWithRunner(t *testing.T) {
	ctx := context.Background()
	got := GetRunner(ctx)
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
	r := NewRunner()
	ctx = WithRunner(ctx, r)
	got = GetRunner(ctx)
	if got != r {
		t.Errorf("got %v, want %v", got, r)
	}
}

func TestWithNames(t *testing.T) {
	ctx := context.Background()
	got := GetNames(ctx)
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
	m := make(map[uintptr]string)
	ctx = WithNames(ctx, m)
	got = GetNames(ctx)
	if got == nil {
		t.Error("got nil, want non-nil")
	}
}
