package fab

import (
	"os"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestYAML(t *testing.T) {
	resetRegistry()

	spew.Config.DisableMethods = true

	f, err := os.Open("_testdata/yaml/fab.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := ReadYAML(f); err != nil {
		t.Fatal(err)
	}

	names := RegistryNames()
	wantNames := []string{
		"Bar",
		"Baz",
		"Foo",
		"X",
		"Y",
		"Z",
	}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("got %v, want %v", names, wantNames)
	}

	gotFoo, gotFooDoc := RegistryTarget("Foo")
	wantFoo := All(&deferredResolutionTarget{name: "Bar"}, &deferredResolutionTarget{name: "Baz"})
	const wantFooDoc = "Foo does Bar and Baz."
	if !reflect.DeepEqual(gotFoo, wantFoo) {
		t.Errorf("mismatch for Foo; got:\n%s\nwant:\n%s", spew.Sdump(gotFoo), spew.Sdump(wantFoo))
	}
	if gotFooDoc != wantFooDoc {
		t.Errorf("got %s for Foo doc, want %s", gotFooDoc, wantFooDoc)
	}

	gotBar, gotBarDoc := RegistryTarget("Bar")
	wantBar := Command("echo How do you do", CmdStdout(os.Stdout))
	const wantBarDoc = "Bar doesn't do much."
	if !reflect.DeepEqual(gotBar, wantBar) {
		t.Errorf("mismatch for Bar; got:\n%s\nwant:\n%s", spew.Sdump(gotBar), spew.Sdump(wantBar))
	}
	if gotBarDoc != wantBarDoc {
		t.Errorf("got %s for Bar doc, want %s", gotBarDoc, wantBarDoc)
	}

	gotBaz, gotBazDoc := RegistryTarget("Baz")
	wantBaz := Deps(
		&deferredResolutionTarget{name: "X"},
		&deferredResolutionTarget{name: "Y"},
		&deferredResolutionTarget{name: "Z"},
	)
	const wantBazDoc = "Baz does X after Y and Z."
	if !reflect.DeepEqual(gotBaz, wantBaz) {
		t.Errorf("mismatch for Baz; got:\n%s\nwant:\n%s", spew.Sdump(gotBaz), spew.Sdump(wantBaz))
	}
	if gotBazDoc != wantBazDoc {
		t.Errorf("got %s for Baz doc, want %s", gotBazDoc, wantBazDoc)
	}

	gotX, gotXDoc := RegistryTarget("X")
	wantX := Seq(
		&deferredResolutionTarget{name: "A"},
		&deferredResolutionTarget{name: "B"},
		&deferredResolutionTarget{name: "C"},
	)
	const wantXDoc = "X does A then B then C."
	if !reflect.DeepEqual(gotX, wantX) {
		t.Errorf("mismatch for X; got:\n%s\nwant:\n%s", spew.Sdump(gotX), spew.Sdump(wantX))
	}
	if gotXDoc != wantXDoc {
		t.Errorf("got %s for X doc, want %s", gotXDoc, wantXDoc)
	}

	gotY, gotYDoc := RegistryTarget("Y")
	wantY := Clean("file1", "file2")
	const wantYDoc = "Y cleans."
	if !reflect.DeepEqual(gotY, wantY) {
		t.Errorf("mismatch for Y; got:\n%s\nwant:\n%s", spew.Sdump(gotY), spew.Sdump(wantY))
	}
	if gotYDoc != wantYDoc {
		t.Errorf("got %s for Y doc, want %s", gotYDoc, wantYDoc)
	}

	gotZ, gotZDoc := RegistryTarget("Z")
	wantZ := &Files{
		Target: Command("go build -o output ./..."),
		In:     []string{"p.go", "q.go", "r.go"},
		Out:    []string{"output"},
	}
	const wantZDoc = "Z builds output if p.go, q.go, or r.go change."
	if !reflect.DeepEqual(gotZ, wantZ) {
		t.Errorf("mismatch for Z; got:\n%s\nwant:\n%s", spew.Sdump(gotZ), spew.Sdump(wantZ))
	}
	if gotZDoc != wantZDoc {
		t.Errorf("got %s for Z doc, want %s", gotZDoc, wantZDoc)
	}
}
