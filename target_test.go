package fab

import (
	"context"
	"testing"

	"github.com/bobg/go-generics/parallel"
	"github.com/bobg/go-generics/set"
)

func TestID(t *testing.T) {
	// Producing any number of id's, even concurrently, should never produce a duplicate.

	const count = 10000

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
