package fab

import (
	"context"
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

	err = os.WriteFile(inPath, []byte("foo"), 0644)
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
				if string(got) != expect+"foo" {
					t.Errorf("got %s, want %sfoo", string(got), expect)
				} else {
					expect += "foo"
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
}

type memdb set.Of[string]

var _ HashDB = memdb{}

func (m memdb) Has(_ context.Context, h []byte) (bool, error) {
	return (set.Of[string])(m).Has(string(h)), nil
}

func (m memdb) Add(_ context.Context, h []byte) error {
	(set.Of[string])(m).Add(string(h))
	return nil
}
