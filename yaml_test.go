package fab

import (
	"context"
	"io"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestYAML(t *testing.T) {
	t.Parallel()

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
		"DiscardStderr",
		"DiscardStdout",
		"Foo",
		"IndentStderr",
		"IndentStdout",
		"MultiCommand",
		"StderrStderr",
		"StderrStdout",
		"StdoutStderr",
		"StdoutStdout",
		"VerboseStderr",
		"VerboseStdout",
		"W",
		"X",
		"Y",
		"Z",
	}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("got %v, want %v", names, wantNames)
	}

	t.Run("Foo", func(t *testing.T) {
		t.Parallel()

		gotFoo, gotFooDoc := con.RegistryTarget("Foo")
		wantFoo := All(&deferredResolutionTarget{Name: "Bar"}, &deferredResolutionTarget{Name: "Baz"})
		const wantFooDoc = "Foo does Bar and Baz."
		if !reflect.DeepEqual(gotFoo, wantFoo) {
			t.Errorf("mismatch for Foo; got:\n%s\nwant:\n%s", spew.Sdump(gotFoo), spew.Sdump(wantFoo))
		}
		if gotFooDoc != wantFooDoc {
			t.Errorf("got %s for Foo doc, want %s", gotFooDoc, wantFooDoc)
		}
	})

	t.Run("Bar", func(t *testing.T) {
		t.Parallel()

		gotBar, gotBarDoc := con.RegistryTarget("Bar")
		wantBar := &Command{Shell: "echo How do you do"}
		const wantBarDoc = "Bar doesn't do much."
		if !reflect.DeepEqual(gotBar, wantBar) {
			t.Errorf("mismatch for Bar; got:\n%s\nwant:\n%s", spew.Sdump(gotBar), spew.Sdump(wantBar))
		}
		if gotBarDoc != wantBarDoc {
			t.Errorf("got %s for Bar doc, want %s", gotBarDoc, wantBarDoc)
		}
	})

	wantBaz := Deps(
		&deferredResolutionTarget{Name: "X"},
		&deferredResolutionTarget{Name: "Y"},
		&deferredResolutionTarget{Name: "Z"},
	)

	t.Run("Baz", func(t *testing.T) {
		t.Parallel()

		gotBaz, gotBazDoc := con.RegistryTarget("Baz")
		const wantBazDoc = "Baz does X after Y and Z."
		if !reflect.DeepEqual(gotBaz, wantBaz) {
			t.Errorf("mismatch for Baz; got:\n%s\nwant:\n%s", spew.Sdump(gotBaz), spew.Sdump(wantBaz))
		}
		if gotBazDoc != wantBazDoc {
			t.Errorf("got %s for Baz doc, want %s", gotBazDoc, wantBazDoc)
		}
	})

	t.Run("Baz2", func(t *testing.T) {
		t.Parallel()

		gotBaz2, _ := con.RegistryTarget("Baz2")
		if !reflect.DeepEqual(gotBaz2, wantBaz) { // sic
			t.Errorf("mismatch for Baz2; got:\n%s\nwant:\n%s", spew.Sdump(gotBaz2), spew.Sdump(wantBaz))
		}
	})

	t.Run("X", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("Y", func(t *testing.T) {
		t.Parallel()

		gotY, gotYDoc := con.RegistryTarget("Y")
		wantY := &Clean{Files: []string{"file1", "file2"}}
		const wantYDoc = "Y cleans."
		if !reflect.DeepEqual(gotY, wantY) {
			t.Errorf("mismatch for Y; got:\n%s\nwant:\n%s", spew.Sdump(gotY), spew.Sdump(wantY))
		}
		if gotYDoc != wantYDoc {
			t.Errorf("got %s for Y doc, want %s", gotYDoc, wantYDoc)
		}
	})

	t.Run("Z", func(t *testing.T) {
		t.Parallel()

		gotZ, gotZDoc := con.RegistryTarget("Z")
		wantZ := Files(
			&Command{Shell: "go build -o output ./..."},
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
	})

	t.Run("W", func(t *testing.T) {
		t.Parallel()

		gotW, gotWDoc := con.RegistryTarget("W")
		wantW := ArgTarget(&deferredResolutionTarget{Name: "X"}, "foo", "bar")
		const wantWDoc = "W tests ArgTarget (passing args foo and bar to X)."
		if !reflect.DeepEqual(gotW, wantW) {
			t.Errorf("mismatch for W; got:\n%s\nwant:\n%s", spew.Sdump(gotW), spew.Sdump(wantW))
		}
		if gotWDoc != wantWDoc {
			t.Errorf("got %s for W doc, want %s", gotWDoc, wantWDoc)
		}
	})

	t.Run("DiscardStderr", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("DiscardStderr")
		want := &Command{Shell: "echo Hello", Stderr: io.Discard}
		if !commandsEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
	t.Run("DiscardStdout", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("DiscardStdout")
		want := &Command{Shell: "echo Hello", Stdout: io.Discard}
		if !commandsEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
	t.Run("IndentStderr", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("IndentStderr")
		want := &Command{Shell: "echo Hello", StderrFn: deferredIndent(os.Stderr)}
		if !commandsEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
	t.Run("IndentStdout", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("IndentStdout")
		want := &Command{Shell: "echo Hello", StdoutFn: deferredIndent(os.Stdout)}
		if !commandsEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
	t.Run("StderrStderr", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("StderrStderr")
		want := &Command{Shell: "echo Hello", Stderr: os.Stderr}
		if !commandsEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
	t.Run("StderrStdout", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("StderrStdout")
		want := &Command{Shell: "echo Hello", Stdout: os.Stderr}
		if !commandsEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
	t.Run("StdoutStderr", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("StdoutStderr")
		want := &Command{Shell: "echo Hello", Stderr: os.Stdout}
		if !commandsEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
	t.Run("StdoutStdout", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("StdoutStdout")
		want := &Command{Shell: "echo Hello", Stdout: os.Stdout}
		if !commandsEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
	t.Run("VerboseStderr", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("VerboseStderr")
		want := &Command{Shell: "echo Hello", StderrFn: maybeIndent(os.Stderr)}
		if !commandsEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
	t.Run("VerboseStdout", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("VerboseStdout")
		want := &Command{Shell: "echo Hello", StdoutFn: maybeIndent(os.Stdout)}
		if !commandsEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
	t.Run("MultiCommand", func(t *testing.T) {
		t.Parallel()

		got, _ := con.RegistryTarget("MultiCommand")
		want := Seq(
			&Command{Shell: "echo Wang", Dir: "x", StdoutFile: "foo", StderrFile: "bar"},
			&Command{Shell: "echo Chung", Dir: "x", StdoutFile: ">>foo", StderrFile: ">>bar"},
		)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("mismatch, got:\n%s\nwant:\n%s", spew.Sdump(got), spew.Sdump(want))
		}
	})
}

func TestDeferredResolutionTarget(t *testing.T) {
	t.Parallel()

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

func commandsEqual(t1, t2 Target) bool {
	c1, ok := t1.(*Command)
	if !ok {
		return false
	}
	c2, ok := t2.(*Command)
	if !ok {
		return false
	}
	a, b := *c1, *c2
	if a.StdoutFn != nil {
		if b.StdoutFn == nil {
			return false
		}
		a.StdoutFn = nil
		b.StdoutFn = nil
	} else if b.StdoutFn != nil {
		return false
	}
	if a.StderrFn != nil {
		if b.StderrFn == nil {
			return false
		}
		a.StderrFn = nil
		b.StderrFn = nil
	} else if b.StderrFn != nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}
