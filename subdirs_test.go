package fab

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/otiai10/copy"
)

func TestSubdirs(t *testing.T) {
	t.Parallel()

	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	snaps := cupaloy.New(cupaloy.SnapshotSubdirectory("_testdata/subdirs/output"))

	if err = copy.Copy("_testdata/subdirs/input", tmpdir); err != nil {
		t.Fatal(err)
	}

	con := NewController(tmpdir)
	if err := con.ReadYAMLFile(""); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ctx = WithVerbose(ctx, true)

	t.Run("a", func(t *testing.T) {
		a, _ := con.RegistryTarget("A")
		if a == nil {
			t.Fatal("no target A")
		}
		if err := con.Run(ctx, a); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(tmpdir, "a"))
		if err != nil {
			t.Fatal(err)
		}
		if err = snaps.SnapshotMulti("a", got); err != nil {
			t.Error(err)
		}
	})

	t.Run("b", func(t *testing.T) {
		b, _ := con.RegistryTarget("B")
		if b == nil {
			t.Fatal("no target B")
		}
		if err := con.Run(ctx, b); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(tmpdir, "foo/c")) // sic
		if err != nil {
			t.Fatal(err)
		}
		if err = snaps.SnapshotMulti("b", got); err != nil {
			t.Error(err)
		}
	})

	t.Run("c", func(t *testing.T) {
		c, _ := con.RegistryTarget("foo/C")
		if c == nil {
			t.Fatal("no target C")
		}
		// no need to run C, it already ran as part of B
	})

	t.Run("d", func(t *testing.T) {
		d, _ := con.RegistryTarget("foo/D")
		if d == nil {
			t.Fatal("no target foo/D")
		}
		if err := con.Run(ctx, d); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(tmpdir, "e")) // sic
		if err != nil {
			t.Fatal(err)
		}
		if err = snaps.SnapshotMulti("d", got); err != nil {
			t.Error(err)
		}
	})

	t.Run("e", func(t *testing.T) {
		e, _ := con.RegistryTarget("E")
		if e == nil {
			t.Fatal("no target E")
		}
		// no need to run E, it already ran as part of D
	})
}
