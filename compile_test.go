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
	tbCompile(t, func(tmpdir, pkgdir string) {
		var (
			fabdir  = filepath.Join(tmpdir, "fab")
			rulesgo = filepath.Join(pkgdir, "rules.go")
		)

		m := Main{
			Pkgdir:  pkgdir,
			Fabdir:  fabdir,
			Verbose: testing.Verbose(),
		}

		ctx := context.Background()

		driver, err := m.getDriver(ctx)
		if err != nil {
			t.Fatal(err)
		}

		info, err := os.Stat(driver)
		if err != nil {
			t.Fatal(err)
		}
		modtime := info.ModTime()

		cmd := exec.CommandContext(ctx, driver, "Noop")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err = cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// If the driver is (wrongly) recompiled here,
		// this sleep forces the modtime to be different
		// (on systems where file-modtime granularity
		// is no better than one second).
		time.Sleep(time.Second)

		driver, err = m.getDriver(ctx)
		if err != nil {
			t.Fatal(err)
		}
		info, err = os.Stat(driver)
		if err != nil {
			t.Fatal(err)
		}
		if !modtime.Equal(info.ModTime()) {
			t.Error("driver got rebuilt unexpectedly")
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

		driver, err = m.getDriver(ctx)
		if err != nil {
			t.Fatal(err)
		}
		info, err = os.Stat(driver)
		if err != nil {
			t.Fatal(err)
		}
		if modtime.Equal(info.ModTime()) {
			t.Error("driver did not get rebuilt but should have")
		}

		cmd = exec.CommandContext(ctx, driver, "Noop")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err = cmd.Run(); err != nil {
			t.Fatal(err)
		}
	})
}

func BenchmarkCompile(b *testing.B) {
	tbCompile(b, func(_, pkgdir string) {
		tmpfile, err := os.CreateTemp("", "fab")
		if err != nil {
			b.Fatal(err)
		}
		tmpname := tmpfile.Name()
		defer os.Remove(tmpname)
		if err = tmpfile.Close(); err != nil {
			b.Fatal(err)
		}

		ctx := context.Background()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if err = Compile(ctx, pkgdir, tmpname); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Test or benchmark the compiler.
func tbCompile(tb testing.TB, f func(tmpdir, pkgdir string)) {
	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		tb.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	if err = populateFabDir(tmpdir); err != nil {
		tb.Fatal(err)
	}

	compiledir := filepath.Join(tmpdir, "compile")

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

	f(tmpdir, pkgdir)
}
