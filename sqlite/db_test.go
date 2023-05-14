package sqlite_test

import (
	"context"
	"os"
	"testing"
	"testing/quick"
	"time"

	"github.com/benbjohnson/clock"

	"github.com/bobg/fab"
	. "github.com/bobg/fab/sqlite"
)

func TestDB(t *testing.T) {
	t.Parallel()

	tmpfile, err := os.CreateTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	ctx := context.Background()
	ctx = fab.WithVerbose(ctx, true)

	db, err := Open(tmpfile.Name())
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

func TestDBKeep(t *testing.T) {
	t.Parallel()

	tmpfile, err := os.CreateTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	var (
		clk = clock.NewMock()
		ctx = context.Background()
	)
	ctx = fab.WithVerbose(ctx, true)

	db, err := Open(tmpfile.Name(), Keep(time.Hour), WithClock(clk), UpdateOnAccess(false))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Add(ctx, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	has, err := db.Has(ctx, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("entry [1] missing")
	}

	clk.Add(45 * time.Minute) // not enough to expire [1]

	err = db.Add(ctx, []byte{2})
	if err != nil {
		t.Fatal(err)
	}
	has, err = db.Has(ctx, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("entry [1] missing after 45 minutes")
	}
	has, err = db.Has(ctx, []byte{2})
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("entry [2] missing")
	}

	clk.Add(30 * time.Minute) // expire [1] but not [2]

	err = db.Add(ctx, []byte{3})
	if err != nil {
		t.Fatal(err)
	}
	has, err = db.Has(ctx, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("entry [1] present after 75 minutes")
	}
	has, err = db.Has(ctx, []byte{2})
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("entry [2] missing after 30 minutes")
	}
	has, err = db.Has(ctx, []byte{3})
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("entry [3] missing")
	}
}
