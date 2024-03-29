package proto

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"

	"github.com/bobg/fab"
)

func TestProto(t *testing.T) {
	t.Parallel()

	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	snaps := cupaloy.New(cupaloy.SnapshotSubdirectory("testdata"))

	var (
		ctx        = context.Background()
		con        = fab.NewController("")
		outfilecpp = filepath.Join(tmpdir, "foo2.pb.cc")
		outfileh   = filepath.Join(tmpdir, "foo2.pb.h")
	)
	ctx = fab.WithVerbose(ctx, true)

	p, err := Proto([]string{"testdata/foo2.proto"}, []string{outfilecpp, outfileh}, []string{"testdata"}, []string{"--cpp_out=" + tmpdir})
	if err != nil {
		t.Fatal(err)
	}
	if err = con.Run(ctx, p); err != nil {
		t.Fatal(err)
	}

	cpp, err := os.ReadFile(outfilecpp)
	if err != nil {
		t.Fatal(err)
	}
	if err = snaps.SnapshotMulti("cpp", cpp); err != nil {
		t.Error(err)
	}

	h, err := os.ReadFile(outfileh)
	if err != nil {
		t.Fatal(err)
	}
	if err = snaps.SnapshotMulti("h", h); err != nil {
		t.Error(err)
	}
}

func TestDeps(t *testing.T) {
	t.Parallel()

	want := []string{
		"testdata/foo.proto",
		"testdata/x/bar.proto",
		"testdata/x/plugh.proto",
	}

	got, err := Deps("testdata/foo.proto", []string{"testdata"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestProtoYAML(t *testing.T) {
	t.Parallel()

	f, err := os.Open("testdata/proto.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	con := fab.NewController("")
	if err = con.ReadYAML(f, "testdata"); err != nil {
		t.Fatal(err)
	}
	got, _ := con.RegistryTarget("testdata/Foo")
	want, err := Proto(
		[]string{"testdata/foo.proto"},
		[]string{"testdata/out1", "testdata/out2"},
		[]string{"testdata/x"},
		[]string{"opt1", "opt2"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
