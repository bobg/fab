package fab

import (
	"context"
	"encoding/binary"
	"sync/atomic"
	"testing"

	"github.com/bobg/go-generics/v2/set"
)

func TestRunTarget(t *testing.T) {
	var (
		ctx     = context.Background()
		r       = NewRunner()
		ct      = &countTarget{}
		target  = &Files{Target: ct, In: nil, Out: []string{"/dev/null"}}
		targets []Target
	)

	for i := 0; i < 1000; i++ {
		targets = append(targets, target)
	}

	ctx = WithVerbose(ctx, testing.Verbose())

	err := r.Run(ctx, targets...)
	if err != nil {
		t.Fatal(err)
	}
	if ct.count != 1 {
		t.Errorf("got %d, want 1", ct.count)
	}

	db := memHashDB{s: set.New[string]()}
	ctx = WithHashDB(ctx, &db)

	r = NewRunner()
	err = r.Run(ctx, targets...)
	if err != nil {
		t.Fatal(err)
	}
	if ct.count != 2 {
		t.Errorf("got %d, want 2", ct.count)
	}

	r = NewRunner()
	err = r.Run(ctx, targets...)
	if err != nil {
		t.Fatal(err)
	}
	if ct.count != 2 {
		t.Errorf("got %d, want 2", ct.count)
	}
}

type countTarget struct {
	count uint32
}

func (ct *countTarget) Run(_ context.Context) error {
	atomic.AddUint32(&ct.count, 1)
	return nil
}

func (*countTarget) Desc() string {
	return "count"
}

func (ct *countTarget) Hash(_ context.Context) ([]byte, error) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], atomic.LoadUint32(&ct.count))
	return b[:], nil
}

type memHashDB struct {
	s set.Of[string]
}

func (m *memHashDB) Has(_ context.Context, h []byte) (bool, error) {
	return m.s.Has(string(h)), nil
}

func (m *memHashDB) Add(_ context.Context, h []byte) error {
	m.s.Add(string(h))
	return nil
}
