package fab

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/otiai10/copy"
)

func TestCompile(t *testing.T) {
	tbCompile(t)
}

func BenchmarkCompile(b *testing.B) {
	tbCompile(b)
}

// Test or benchmark the compiler.
func tbCompile(tb testing.TB) {
	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		tb.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	compiledir := filepath.Join(tmpdir, "compile")

	ctx := context.Background()

	if err = copy.Copy("_testdata/compile", compiledir); err != nil {
		tb.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		tb.Fatal(err)
	}
	if err = os.Chdir(compiledir); err != nil {
		tb.Fatal(err)
	}
	defer os.Chdir(cwd)

	pkgdir := filepath.Join(compiledir, "pkg")

	if b, ok := tb.(*testing.B); ok {
		tmpfile, err := os.CreateTemp("", "fab")
		if err != nil {
			b.Fatal(err)
		}
		tmpname := tmpfile.Name()
		defer os.Remove(tmpname)
		if err = tmpfile.Close(); err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if err = Compile(ctx, pkgdir, tmpname); err != nil {
				b.Fatal(err)
			}
		}

		return
	}

	if t, ok := tb.(*testing.T); ok {
		var (
			fabdir  = filepath.Join(tmpdir, "fab")
			rulesgo = filepath.Join(pkgdir, "rules.go")
		)

		m := Main{
			Pkgdir:  pkgdir,
			Fabdir:  fabdir,
			Verbose: testing.Verbose(),
		}

		driver1, err := m.getDriver(ctx)
		if err != nil {
			t.Fatal(err)
		}

		info, err := os.Stat(driver1)
		if err != nil {
			t.Fatal(err)
		}
		modtime := info.ModTime()

		cmd := exec.CommandContext(ctx, driver1, "noop")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err = cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// If the driver is (wrongly) recompiled here,
		// this sleep forces driver2's modtime to be different from driver1's
		// (on systems where file-modtime granularity is no better than one second).
		time.Sleep(time.Second)

		driver2, err := m.getDriver(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if driver1 != driver2 {
			t.Errorf("unexpected new driver %s (vs %s)", driver2, driver1)
		} else {
			info, err = os.Stat(driver2)
			if err != nil {
				t.Fatal(err)
			}
			if !modtime.Equal(info.ModTime()) {
				t.Errorf("driver %s got rebuilt unexpectedly", driver2)
			}
		}

		cmd = exec.CommandContext(ctx, driver2, "noop")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err = cmd.Run(); err != nil {
			t.Fatal(err)
		}

		f, err := os.OpenFile(rulesgo, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		fmt.Fprintln(f, "// comment")
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}

		driver3, err := m.getDriver(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if driver1 == driver3 {
			t.Error("driver did not get rebuilt but should have")
		}

		cmd = exec.CommandContext(ctx, driver3, "noop")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err = cmd.Run(); err != nil {
			t.Fatal(err)
		}

		return
	}
}
