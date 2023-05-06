package fab

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"github.com/bobg/go-generics/v2/set"
	"github.com/bradleyjkemp/cupaloy/v2"
)

func TestRunTarget(t *testing.T) {
	var (
		ctx     = context.Background()
		con     = NewController("")
		ct      = &countTarget{}
		target  = Files(ct, nil, []string{"/dev/null"})
		targets []Target
	)

	for i := 0; i < 1000; i++ {
		targets = append(targets, target)
	}

	ctx = WithVerbose(ctx, testing.Verbose())

	err := con.Run(ctx, targets...)
	if err != nil {
		t.Fatal(err)
	}
	if ct.count != 1 {
		t.Errorf("got %d, want 1", ct.count)
	}

	db := memHashDB{s: set.New[string]()}
	ctx = WithHashDB(ctx, &db)

	con = NewController("")
	err = con.Run(ctx, targets...)
	if err != nil {
		t.Fatal(err)
	}
	if ct.count != 2 {
		t.Errorf("got %d, want 2", ct.count)
	}

	con = NewController("")
	err = con.Run(ctx, targets...)
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

func (ct *countTarget) Run(context.Context, *Controller) error {
	atomic.AddUint32(&ct.count, 1)
	return nil
}

func (*countTarget) Desc() string {
	return "count"
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

func TestIndentingCopier(t *testing.T) {
	b, err := os.ReadFile("_testdata/indenting_copier.input")
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)

	var (
		con = NewController("")
		buf = new(bytes.Buffer)
		w   = con.IndentingCopier(buf, "> ")
	)

	fmt.Fprint(w, text)

	con.incDepth()
	w = con.IndentingCopier(buf, "> ")

	fmt.Fprint(w, text)

	con.incDepth()
	w = con.IndentingCopier(buf, "> ")

	fmt.Fprint(w, text)

	con.decDepth()
	w = con.IndentingCopier(buf, "> ")

	fmt.Fprint(w, text)

	snaps := cupaloy.New(cupaloy.SnapshotSubdirectory("_testdata"))
	snaps.SnapshotT(t, buf.String())
}
