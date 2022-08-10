package loader

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

func Run(ctx context.Context, pkgdir string, args ...string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "getting working directory")
	}
	return Load(ctx, pkgdir, func(dir string) error {
		prog := filepath.Join(dir, "x")
		args = append([]string{"-d", cwd}, args...)
		cmd := exec.CommandContext(ctx, prog, args...)
		cmd.Dir = dir
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		fmt.Printf("xxx about to run %#v\n", cmd)
		return cmd.Run()
	})
}
