package golang

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/bobg/go-generics/v2/slices"
	"github.com/otiai10/copy"

	"github.com/bobg/fab"
)

func TestBinary(t *testing.T) {
	t.Parallel()

	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	var (
		ctx       = context.Background()
		fabdir    = filepath.Join(tmpdir, "fab")
		binarydir = filepath.Join(tmpdir, "binary")
		outfile   = filepath.Join(tmpdir, "out")
	)
	ctx = fab.WithVerbose(ctx, true)

	db, err := fab.OpenHashDB(fabdir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx = fab.WithHashDB(ctx, db)

	if err = copy.Copy("_testdata/binary", binarydir); err != nil {
		t.Fatal(err)
	}

	targ, err := Binary(binarydir, outfile)
	if err != nil {
		t.Fatal(err)
	}

	con := fab.NewController("")

	if err = con.Run(ctx, targ); err != nil {
		t.Fatal(err)
	}
}

var testGoDeps = []string{
	"../all.go",
	"../argtarg.go",
	"../clean.go",
	"../command.go",
	"../compile.go",
	"../context.go",
	"../controller.go",
	"../deps.go",
	"../f.go",
	"../files.go",
	"../gate.go",
	"../hash.go",
	"../main.go",
	"../register.go",
	"../registry.go",
	"../runner.go",
	"../seq.go",
	"../sqlite/db.go",
	"../sqlite/schema.sql",
	"../target.go",
	"../top.go",
	"../yaml.go",
	"go.go",
}

func TestDeps(t *testing.T) {
	t.Parallel()

	got, err := Deps(".", false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Use relative paths to make the result non-system-dependent.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err = slices.Mapx(got, func(_ int, full string) (string, error) {
		return filepath.Rel(cwd, full)
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)

	if !reflect.DeepEqual(got, testGoDeps) {
		t.Errorf("got %v, want %v", got, testGoDeps)
	}
}

func TestGoYAML(t *testing.T) {
	t.Parallel()

	f, err := os.Open("_testdata/go.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	con := fab.NewController("")
	if err = con.ReadYAML(f, "_testdata"); err != nil {
		t.Fatal(err)
	}

	t.Run("binary", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("_testdata/Foo")
		want, err := Binary("_testdata/binary", "_testdata/b")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("deps", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("_testdata/Bar")
		deps, err := slices.Mapx(testGoDeps, func(_ int, s string) (string, error) { return filepath.Abs(s) })
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(deps)
		want := fab.Files(
			&fab.Command{Shell: "echo bar", Dir: "_testdata", StdoutFile: "_testdata/bar"},
			deps,
			[]string{"_testdata/bar"},
		)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}
