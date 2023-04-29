package fab

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCommand(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	ctx := context.Background()

	hw, err := os.ReadFile("_testdata/hw")
	if err != nil {
		t.Fatal(err)
	}

	f1 := filepath.Join(tmpdir, "f1")

	t.Run("stdoutfile", func(t *testing.T) {
		c1 := &Command{Cmd: "cat", Args: []string{"_testdata/hw"}, StdoutFile: f1}
		if err = Run(ctx, c1); err != nil {
			t.Fatal(err)
		}
		got1, err := os.ReadFile(f1)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got1, hw) {
			t.Errorf("got %s, want %s", string(got1), string(hw))
		}
	})

	t.Run("stdoutfile_append", func(t *testing.T) {
		hwhw := append([]byte{}, hw...)
		hwhw = append(hwhw, hw...)

		c2 := &Command{Cmd: "cat", Args: []string{"_testdata/hw"}, StdoutFile: ">>" + f1}
		if err = Run(ctx, c2); err != nil {
			t.Fatal(err)
		}
		got2, err := os.ReadFile(f1)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got2, hwhw) {
			t.Errorf("got %s, want %s", string(got2), string(hwhw))
		}
	})

	const dne = "_a_file_that_does_not_exist_"
	var (
		f3     = filepath.Join(tmpdir, "f3")
		f3size int64
	)

	t.Run("stderrfile", func(t *testing.T) {
		c3 := &Command{Cmd: "cat", Args: []string{dne}, StderrFile: f3}
		err = Run(ctx, c3)
		if err == nil { // sic
			t.Fatal("got no error but expected one")
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatal(err)
		}
		// Make sure f3 exists and has non-zero size.
		info, err := os.Stat(f3)
		if err != nil {
			t.Fatal(err)
		}
		f3size = info.Size()
		if f3size == 0 {
			t.Error("f3 exists but has size zero")
		}
	})

	t.Run("stderrfile_append", func(t *testing.T) {
		c4 := &Command{Cmd: "cat", Args: []string{dne}, StderrFile: ">>" + f3}
		err = Run(ctx, c4)
		if err == nil { // sic
			t.Fatal("got no error but expected one")
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatal(err)
		}
		// Make sure f3 has a greater size than before.
		info, err := os.Stat(f3)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() <= f3size {
			t.Errorf("got f3 size %d, want some number greater than %d", info.Size(), f3size)
		}
	})

	t.Run("stdoutfn", func(t *testing.T) {
		buf := new(bytes.Buffer)
		fn := func(context.Context) io.Writer { return buf }
		c5 := &Command{Cmd: "cat", Args: []string{"_testdata/hw"}, StdoutFn: fn}
		if err = Run(ctx, c5); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buf.Bytes(), hw) {
			t.Errorf("got %s, want %s", buf, string(hw))
		}
	})

	t.Run("stderrfn", func(t *testing.T) {
		buf := new(bytes.Buffer)
		fn := func(context.Context) io.Writer { return buf }
		c6 := &Command{Cmd: "cat", Args: []string{dne}, StderrFn: fn}
		err = Run(ctx, c6)
		if err == nil { // sic
			t.Fatal("got no error but expected one")
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatal(err)
		}
		if buf.Len() == 0 {
			t.Error("no stderr bytes captured")
		}
	})
}
