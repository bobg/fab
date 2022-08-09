package load

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Run(ctx context.Context, pkgdir string, args ...string) error {
	return Load(ctx, pkgdir, func(dir string) error {
		prog := filepath.Join(dir, "x")
		cmd := exec.CommandContext(ctx, prog, args...)
		cmd.Dir = dir
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		fmt.Printf("xxx about to run %#v\n", cmd)
		return cmd.Run()
	})
}
