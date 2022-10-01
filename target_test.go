package fab

import (
	"context"
	"testing"

	"github.com/bobg/go-generics/parallel"
	"github.com/bobg/go-generics/set"
)

func TestUnique(t *testing.T) {
	// Producing any number of unique names, even concurrently, should never produce a duplicate.

	const count = 5000

	names := parallel.Producers(context.Background(), count, func(_ context.Context, _ int, send func(name string) error) error {
		return send(Unique("x"))
	})
	s := set.New[string]()
	for names.Next() {
		s.Add(names.Val())
	}
	if s.Len() != count {
		t.Errorf("got %d distinct values, want %d", s.Len(), count)
	}
}

func TestName(t *testing.T) {
	t1 := F(func(_ context.Context) error { return nil })
	got := t1.Name()
	if got != t1.Name() {
		t.Errorf("got %s, want %s [1]", got, t1.Name())
	}
	t2 := Register("plugh", "", t1)
	got = t2.Name()
	if got != "plugh" {
		t.Errorf("got %s, want plugh", got)
	}
}
