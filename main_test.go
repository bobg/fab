package fab

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
)

func TestMain(t *testing.T) {
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
		Pkgdir: filepath.Join(compiledir, "pkg"),
		Fabdir: fabdir,
		Chdir:  tmpdir,
		Args:   []string{"Noop"},

		// The following are here mainly to improve test coverage.
		Verbose: true,
		Force:   true,
	}

	if err = m.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}
