package fab

import (
	"context"
	"os"
	"testing"
)

func TestDriverless(t *testing.T) {
	t.Parallel()

	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	m := Main{
		Fabdir: tmpdir,
		Topdir: "_testdata/driverless",
		Args:   []string{"Noop"},
	}

	ctx := context.Background()
	ctx = WithVerbose(ctx, true)

	if err := m.Run(ctx); err != nil {
		t.Fatal(err)
	}
}
