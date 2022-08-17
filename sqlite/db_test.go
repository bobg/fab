package sqlite

import (
	"context"
	"os"
	"testing"
	"testing/quick"
)

func TestDB(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	ctx := context.Background()

	db, err := Open(ctx, tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var count, had int

	err = quick.Check(func(s string) bool {
		if len(s) == 0 {
			return true
		}
		b := []byte(s)
		want := (b[0]&1 == 1)
		if want {
			err = db.Add(ctx, b)
			if err != nil {
				t.Fatal(err)
			}
		}
		got, err := db.Has(ctx, b)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			return false
		}

		count++
		if want {
			had++
		}

		return true
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("count %d had %d", count, had)
}
