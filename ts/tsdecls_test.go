package ts

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/davecgh/go-spew/spew"
	"github.com/otiai10/copy"

	"github.com/bobg/fab"
)

func TestDecls(t *testing.T) {
	t.Parallel()

	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	if err = copy.Copy("_testdata/input", tmpdir); err != nil {
		t.Fatal(err)
	}

	con := fab.NewController(tmpdir)
	if err := con.ReadYAMLFile(""); err != nil {
		t.Fatal(err)
	}

	outfile := filepath.Join(tmpdir, "out.ts")

	targ, _ := con.RegistryTarget("Build")
	want := fab.Files(
		&declsType{
			Dir:      filepath.Join(tmpdir, "lib"),
			Typename: "Server",
			Prefix:   "/s",
			Outfile:  outfile,
		},
		[]string{filepath.Join(tmpdir, "lib/tt.go")},
		[]string{outfile},
		fab.Autoclean(true),
	)
	if !reflect.DeepEqual(targ, want) {
		spew.Config.DisableMethods = true
		t.Errorf("got:\n%s\nwant:\n%s", spew.Sdump(targ), spew.Sdump(want))
	}

	ctx := context.Background()
	ctx = fab.WithVerbose(ctx, true)

	if err := con.Run(ctx, targ); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(outfile)
	if err != nil {
		t.Fatal(err)
	}

	snaps := cupaloy.New(cupaloy.SnapshotSubdirectory("_testdata"))
	snaps.SnapshotT(t, string(got))
}
