package fab

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/bobg/errors"
)

func TestClean(t *testing.T) {
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

	r := NewRunner()
	if err = r.Run(context.Background(), Clean(tmpname, "/tmp/i-hope-i-am-a-file-that-does-not-exist")); err != nil {
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
