package fab

import (
	"context"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/set"
)

func TestHashTarget(t *testing.T) {
	dir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var (
		inpath  = filepath.Join(dir, "in")
		outpath = filepath.Join(dir, "out")
		teststr = "foo"
	)

	err = os.WriteFile(inpath, []byte(teststr), 0644)
	if err != nil {
		t.Fatal(err)
	}

	fc := Files(
		Shellf("sh -c 'cat %s >> %s'", inpath, outpath),
		[]string{inpath},
		[]string{outpath},
	)

	ctx := context.Background()
	ctx = WithVerbose(ctx, testing.Verbose())

	expect := ""
	try := func(want bool) func(t *testing.T) {
		t.Helper()

		return func(t *testing.T) {
			r := NewRunner()
			err = r.Run(ctx, fc)
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(outpath)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("Reading %s: %s", outpath, err)
			}
			if err != nil {
				got = nil
			}
			if want {
				if string(got) != expect+teststr {
					t.Errorf("got %s, want %s%s", string(got), expect, teststr)
				} else {
					expect += teststr
				}
			} else {
				if string(got) != expect {
					t.Errorf("got %s, want %s", string(got), expect)
				}
			}
		}
	}

	t.Run("1 no db", try(true))

	db := memdb(set.New[string]())
	ctx = WithHashDB(ctx, db)
	ctx = WithVerbose(ctx, testing.Verbose())

	t.Run("2 empty db", try(true))
	t.Run("3 non-empty db", try(false))

	// Rewrite inpath but don't change its contents.
	err = os.WriteFile(inpath, []byte(teststr), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("4 unaltered inpath but new modtime", try(false))

	// Rewrite inpath and do change its contents.
	teststr = "bar"
	err = os.WriteFile(inpath, []byte(teststr), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("5 altered inpath", try(true))

	// Remove outpath.
	err = os.Remove(outpath)
	if err != nil {
		t.Fatal(err)
	}

	expect = ""

	t.Run("6 removed outpath", try(true))

	// Alter outpath.
	expect = teststr + "x"
	err = os.WriteFile(outpath, []byte(expect), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("7 altered outpath", try(true))

	// Set inpath and outpath to an earlier state that's in the db.
	expect = "foofoo"
	teststr = "foo"
	err = os.WriteFile(inpath, []byte(teststr), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(outpath, []byte(expect), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("8 earlier state restored", try(false))
}

type memdb set.Of[string]

var _ HashDB = memdb{}

func (m memdb) Has(_ context.Context, h []byte) (bool, error) {
	return (set.Of[string])(m).Has(hex.EncodeToString(h)), nil
}

func (m memdb) Add(_ context.Context, h []byte) error {
	(set.Of[string])(m).Add(hex.EncodeToString(h))
	return nil
}
