package fab

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/bobg/go-generics/set"
	"github.com/pkg/errors"
)

func TestHashTarget(t *testing.T) {
	dir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	inPath := filepath.Join(dir, "in")
	outPath := filepath.Join(dir, "out")

	teststr := "foo"

	err = os.WriteFile(inPath, []byte(teststr), 0644)
	if err != nil {
		t.Fatal(err)
	}

	fc := &FilesCommand{
		Command: &Command{
			Shell: fmt.Sprintf("sh -c 'cat %s >> %s'", inPath, outPath),
		},
		In:  []string{inPath},
		Out: []string{outPath},
	}

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
			got, err := os.ReadFile(outPath)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("Reading %s: %s", outPath, err)
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

	t.Run("first run, no db", try(true))

	db := memdb(set.New[string]())
	ctx = WithHashDB(ctx, db)

	t.Run("second run, with empty db", try(true))
	t.Run("third run, with non-empty db", try(false))

	// Rewrite inPath but don't change its contents.
	err = os.WriteFile(inPath, []byte(teststr), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("fourth run, with unaltered inPath but new modtime", try(false))

	// Rewrite inPath and do change its contents.
	teststr = "bar"
	err = os.WriteFile(inPath, []byte(teststr), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("fifth run, with altered inPath", try(true))

	// Remove outPath.
	err = os.Remove(outPath)
	if err != nil {
		t.Fatal(err)
	}

	expect = ""

	t.Run("sixth run, with removed outPath", try(true))

	// Alter outPath.
	expect = teststr + "x"
	err = os.WriteFile(outPath, []byte(expect), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("seventh run, with altered outPath", try(true))

	// Set inPath and outPath to an earlier state that's in the db.
	expect = "foofoo"
	teststr = "foo"
	err = os.WriteFile(inPath, []byte(teststr), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(outPath, []byte(expect), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("eighth run, with an earlier state restored", try(false))
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
