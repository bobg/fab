package fab

import (
	"context"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestYAML(t *testing.T) {
	spew.Config.DisableMethods = true

	f, err := os.Open("_testdata/yaml/fab.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	con := NewController("")

	if err := con.ReadYAML(f, ""); err != nil {
		t.Fatal(err)
	}

	names := con.RegistryNames()
	wantNames := []string{
		"Bar",
		"Baz",
		"Baz2",
		"Foo",
		"W",
		"X",
		"Y",
		"Z",
	}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("got %v, want %v", names, wantNames)
	}

	gotFoo, gotFooDoc := con.RegistryTarget("Foo")
	wantFoo := All(&deferredResolutionTarget{Name: "Bar"}, &deferredResolutionTarget{Name: "Baz"})
	const wantFooDoc = "Foo does Bar and Baz."
	if !reflect.DeepEqual(gotFoo, wantFoo) {
		t.Errorf("mismatch for Foo; got:\n%s\nwant:\n%s", spew.Sdump(gotFoo), spew.Sdump(wantFoo))
	}
	if gotFooDoc != wantFooDoc {
		t.Errorf("got %s for Foo doc, want %s", gotFooDoc, wantFooDoc)
	}

	gotBar, gotBarDoc := con.RegistryTarget("Bar")
	wantBar := &Command{Shell: "echo How do you do", Stdout: os.Stdout, Dir: "."}
	const wantBarDoc = "Bar doesn't do much."
	if !reflect.DeepEqual(gotBar, wantBar) {
		t.Errorf("mismatch for Bar; got:\n%s\nwant:\n%s", spew.Sdump(gotBar), spew.Sdump(wantBar))
	}
	if gotBarDoc != wantBarDoc {
		t.Errorf("got %s for Bar doc, want %s", gotBarDoc, wantBarDoc)
	}

	gotBaz, gotBazDoc := con.RegistryTarget("Baz")
	wantBaz := Deps(
		&deferredResolutionTarget{Name: "X"},
		&deferredResolutionTarget{Name: "Y"},
		&deferredResolutionTarget{Name: "Z"},
	)
	const wantBazDoc = "Baz does X after Y and Z."
	if !reflect.DeepEqual(gotBaz, wantBaz) {
		t.Errorf("mismatch for Baz; got:\n%s\nwant:\n%s", spew.Sdump(gotBaz), spew.Sdump(wantBaz))
	}
	if gotBazDoc != wantBazDoc {
		t.Errorf("got %s for Baz doc, want %s", gotBazDoc, wantBazDoc)
	}

	gotBaz2, _ := con.RegistryTarget("Baz2")
	if !reflect.DeepEqual(gotBaz2, wantBaz) { // sic
		t.Errorf("mismatch for Baz2; got:\n%s\nwant:\n%s", spew.Sdump(gotBaz2), spew.Sdump(wantBaz))
	}

	gotX, gotXDoc := con.RegistryTarget("X")
	wantX := Seq(
		&deferredResolutionTarget{Name: "A"},
		&deferredResolutionTarget{Name: "B"},
		&deferredResolutionTarget{Name: "C"},
	)
	const wantXDoc = "X does A then B then C."
	if !reflect.DeepEqual(gotX, wantX) {
		t.Errorf("mismatch for X; got:\n%s\nwant:\n%s", spew.Sdump(gotX), spew.Sdump(wantX))
	}
	if gotXDoc != wantXDoc {
		t.Errorf("got %s for X doc, want %s", gotXDoc, wantXDoc)
	}

	gotY, gotYDoc := con.RegistryTarget("Y")
	wantY := Clean("file1", "file2")
	const wantYDoc = "Y cleans."
	if !reflect.DeepEqual(gotY, wantY) {
		t.Errorf("mismatch for Y; got:\n%s\nwant:\n%s", spew.Sdump(gotY), spew.Sdump(wantY))
	}
	if gotYDoc != wantYDoc {
		t.Errorf("got %s for Y doc, want %s", gotYDoc, wantYDoc)
	}

	gotZ, gotZDoc := con.RegistryTarget("Z")
	wantZ := Files(
		&Command{Shell: "go build -o output ./...", Dir: "."},
		[]string{"p.go", "q.go", "r.go"},
		[]string{"output"},
	)
	const wantZDoc = "Z builds output if p.go, q.go, or r.go change."
	if !reflect.DeepEqual(gotZ, wantZ) {
		t.Errorf("mismatch for Z; got:\n%s\nwant:\n%s", spew.Sdump(gotZ), spew.Sdump(wantZ))
	}
	if gotZDoc != wantZDoc {
		t.Errorf("got %s for Z doc, want %s", gotZDoc, wantZDoc)
	}

	gotW, gotWDoc := con.RegistryTarget("W")
	wantW := ArgTarget(&deferredResolutionTarget{Name: "X"}, "foo", "bar")
	const wantWDoc = "W tests ArgTarget (passing args foo and bar to X)."
	if !reflect.DeepEqual(gotW, wantW) {
		t.Errorf("mismatch for W; got:\n%s\nwant:\n%s", spew.Sdump(gotW), spew.Sdump(wantW))
	}
	if gotWDoc != wantWDoc {
		t.Errorf("got %s for W doc, want %s", gotWDoc, wantWDoc)
	}
}

func TestDeferredResolutionTarget(t *testing.T) {
	var (
		dtarg = &deferredResolutionTarget{Name: "c"}
		ctarg = &countTarget{}
		con   = NewController("")
	)

	_, err := con.RegisterTarget("c", "", ctarg)
	if err != nil {
		t.Fatal(err)
	}

	if err = con.Run(context.Background(), dtarg); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadUint32(&ctarg.count); got != 1 {
		t.Errorf("got %d, want 1", got)
	}
}
