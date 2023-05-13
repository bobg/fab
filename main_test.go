package fab

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
)

func TestMain(t *testing.T) {
	t.Parallel()

	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	if err = populateFabDir(tmpdir); err != nil {
		t.Fatal(err)
	}

	var (
		fabdir     = filepath.Join(tmpdir, ".fab")
		compiledir = filepath.Join(tmpdir, "compile")
	)
	if err = os.Mkdir(fabdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(compiledir, 0755); err != nil {
		t.Fatal(err)
	}

	if err = copy.Copy("_testdata/compile", compiledir); err != nil {
		t.Fatal(err)
	}

	m := Main{
		Fabdir: fabdir,
		Topdir: compiledir,
		Args:   []string{"Noop"},

		// The following are here mainly to improve test coverage.
		Verbose: true,
		Force:   true,
	}

	if err = m.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	m.Force = false

	// A second Run will exercise more of getDriver.
	if err = m.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	// For some reason a third Run is needed to exercise checkVersion.
	// TODO: figure out why.
	if err = m.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

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

	if err := m.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}
