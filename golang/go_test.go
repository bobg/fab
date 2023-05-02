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

	db, err := fab.OpenHashDB(ctx, fabdir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx = fab.WithHashDB(ctx, db)
	ctx = fab.WithVerbose(ctx, testing.Verbose())

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

func TestDeps(t *testing.T) {
	want := []string{
		"../all.go",
		"../all_test.go",
		"../argtarg.go",
		"../argtarg_test.go",
		"../clean.go",
		"../clean_test.go",
		"../command.go",
		"../command_test.go",
		"../compile.go",
		"../compile_test.go",
		"../context.go",
		"../context_test.go",
		"../controller.go",
		"../controller_test.go",
		"../deps.go",
		"../deps_test.go",
		"../dirhash.go",
		"../driver.go.tmpl",
		"../embeds.go",
		"../f.go",
		"../files.go",
		"../files_test.go",
		"../gate.go",
		"../gate_test.go",
		"../go.mod",
		"../go.sum",
		"../hash.go",
		"../hash_test.go",
		"../main.go",
		"../main_test.go",
		"../proto/proto.go",
		"../proto/proto_test.go",
		"../register.go",
		"../register_test.go",
		"../runner.go",
		"../runner_test.go",
		"../seq.go",
		"../seq_test.go",
		"../sqlite/db.go",
		"../sqlite/db_test.go",
		"../sqlite/migrations/20220829093104_initial.sql",
		"../subdirs_test.go",
		"../target.go",
		"../top.go",
		"../top_test.go",
		"../ts/tsdecls.go",
		"../types.go",
		"../types_test.go",
		"../yaml.go",
		"../yaml_test.go",
		"go.go",
		"go_test.go",
	}

	got, err := Deps(".", false)
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

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
