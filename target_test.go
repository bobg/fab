package fab

import (
	"context"
	"reflect"
	"testing"

	"github.com/bobg/go-generics/parallel"
	"github.com/bobg/go-generics/set"
)

func TestID(t *testing.T) {
	// Producing any number of id's, even concurrently, should never produce a duplicate.

	const count = 5000

	ids := parallel.Producers(context.Background(), count, func(_ context.Context, _ int, send func(id string) error) error {
		return send(ID("x"))
	})
	s := set.New[string]()
	for ids.Next() {
		s.Add(ids.Val())
	}
	if s.Len() != count {
		t.Errorf("got %d distinct values, want %d", s.Len(), count)
	}
}

func TestName(t *testing.T) {
	t1 := F(func(_ context.Context) error { return nil })
	ctx := context.Background()
	got := Name(ctx, t1)
	if got != t1.ID() {
		t.Errorf("got %s, want %s [1]", got, t1.ID())
	}
	names := map[uintptr]string{0: "foo"}
	ctx = WithNames(ctx, names)
	got = Name(ctx, t1)
	if got != t1.ID() {
		t.Errorf("got %s, want %s [2]", got, t1.ID())
	}
	v := reflect.ValueOf(t1)
	names[v.Pointer()] = "plugh"
	got = Name(ctx, t1)
	if got != "plugh" {
		t.Errorf("got %s, want plugh", got)
	}
}
