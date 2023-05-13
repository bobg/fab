package fab

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/bobg/errors"
)

func TestClean(t *testing.T) {
	t.Parallel()

	tmpfile, err := os.CreateTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	tmpname := tmpfile.Name()
	defer os.Remove(tmpname)

	fmt.Fprintln(tmpfile, "Hello, world!")
	if err = tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	con := NewController("")
	clean := &Clean{
		Files: []string{
			tmpname,
			"/tmp/i-hope-i-am-a-file-that-does-not-exist",
		},
	}
	if err = con.Run(context.Background(), clean); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(tmpname)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// ok!
	case err != nil:
		t.Fatal(err)
	default:
		t.Errorf("failed to remove %s", tmpname)
	}
}

func TestAutoclean(t *testing.T) {
	t.Parallel()

	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	path := filepath.Join(tmpdir, "outfile")

	mkfile := F(func(context.Context, *Controller) error {
		return os.WriteFile(path, []byte("Professor Little Old Man!"), 0644)
	})
	files := Files(mkfile, nil, []string{path}, Autoclean(true))

	var (
		con = NewController("")
		ctx = context.Background()
	)

	ctx = WithVerbose(ctx, testing.Verbose())

	if err = con.Run(ctx, files); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if err = con.Run(ctx, &Clean{Autoclean: true}); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(path)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got %v, want %v", err, fs.ErrNotExist)
	}
}
