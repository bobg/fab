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

	ctx := context.Background()

	m := Main{
		Pkgdir: filepath.Join(compiledir, "pkg"),
		Fabdir: fabdir,
		Args:   []string{"Noop"},

		// The following are here mainly to improve test coverage.
		Verbose: true,
		Force:   true,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(compiledir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	if err = m.Run(ctx); err != nil {
		t.Fatal(err)
	}
}
