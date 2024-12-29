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
	t.Parallel()

	con, err := NewController("")
	if err != nil {
		t.Fatal(err)
	}
	con.Verbose = true

	ctx := context.Background()

	var (
		ct      = &countTarget{}
		target  = Files(ct, nil, []string{"/dev/null"})
		targets []Target
	)

	for i := 0; i < 1000; i++ {
		targets = append(targets, target)
	}

	if err = con.Run(ctx, targets...); err != nil {
		t.Fatal(err)
	}
	if ct.count != 1 {
		t.Errorf("got %d, want 1", ct.count)
	}

	con, err = NewController("")
	if err != nil {
		t.Fatal(err)
	}
	db := &memHashDB{s: set.New[string]()}
	con.DB = db
	con.Verbose = true

	err = con.Run(ctx, targets...)
	if err != nil {
		t.Fatal(err)
	}
	if ct.count != 2 {
		t.Errorf("got %d, want 2", ct.count)
	}

	con, err = NewController("")
	if err != nil {
		t.Fatal(err)
	}
	con.DB = db
	con.Verbose = true

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
	t.Parallel()

	b, err := os.ReadFile("_testdata/indenting_copier.input")
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)

	con, err := NewController("")
	if err != nil {
		t.Fatal(err)
	}

	var (
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

func TestIndentf(t *testing.T) {
	t.Parallel()

	con, err := NewController("")
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	con.indentf(buf, "foo")
	if got := buf.String(); got != "foo\n" {
		t.Errorf("got %s, want foo\\n", buf.String())
	}
	buf.Reset()

	con.incDepth()
	con.indentf(buf, "bar")
	if got := buf.String(); got != "  bar\n" {
		t.Errorf("got %s, want \"  bar\\n\"", buf.String())
	}
}
