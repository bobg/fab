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
	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	ctx := context.Background()

	if err = copy.Copy("_testdata/compile", tmpdir); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(tmpdir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	tmpfile, err := os.CreateTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	tmpname := tmpfile.Name()
	defer os.Remove(tmpname)
	if err = tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	var (
		pkgdir  = filepath.Join(tmpdir, "pkg")
		rulesgo = filepath.Join(pkgdir, "rules.go")
	)

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
}

func TestModuleRelPath(t *testing.T) {
	cases := []struct {
		dir, want string
	}{{
		dir: "foo", want: "./foo",
	}, {
		dir: "foo/bar", want: "./foo/bar",
	}}

	for _, tc := range cases {
		t.Run(tc.dir, func(t *testing.T) {
			got, err := moduleRelPath(filepath.Join("_testdata/module_rel_path", tc.dir))
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
