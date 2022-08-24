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

	ctx := context.Background()

	if err = copy.Copy("_testdata/compile", tmpdir); err != nil {
		tb.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		tb.Fatal(err)
	}
	if err = os.Chdir(tmpdir); err != nil {
		tb.Fatal(err)
	}
	defer os.Chdir(cwd)

	tmpfile, err := os.CreateTemp("", "fab")
	if err != nil {
		tb.Fatal(err)
	}
	tmpname := tmpfile.Name()
	defer os.Remove(tmpname)
	if err = tmpfile.Close(); err != nil {
		tb.Fatal(err)
	}

	pkgdir := filepath.Join(tmpdir, "pkg")

	if b, ok := tb.(*testing.B); ok {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if err = Compile(ctx, pkgdir, tmpname); err != nil {
				b.Fatal(err)
			}
		}

		return
	}

	if t, ok := tb.(*testing.T); ok {
		rulesgo := filepath.Join(pkgdir, "rules.go")

		if err = Compile(ctx, pkgdir, tmpname); err != nil {
			t.Fatal(err)
		}

		info, err := os.Stat(tmpname)
		if err != nil {
			t.Fatal(err)
		}
		modtime := info.ModTime()

		time.Sleep(time.Second)

		cmd := exec.CommandContext(ctx, tmpname, "noop")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err = cmd.Run(); err != nil {
			t.Fatal(err)
		}

		info, err = os.Stat(tmpname)
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

		time.Sleep(time.Second)

		cmd = exec.CommandContext(ctx, tmpname, "noop")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err = cmd.Run(); err != nil {
			t.Fatal(err)
		}

		info, err = os.Stat(tmpname)
		if err != nil {
			t.Fatal(err)
		}
		if modtime.Equal(info.ModTime()) {
			t.Error("driver did not get rebuilt but should have")
		}

		return
	}
}
