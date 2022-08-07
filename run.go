package fab

import (
	"context"
	"os/exec"
	"path/filepath"
)

func Run(ctx context.Context, pkgdir string, args ...string) error {
	return Load(ctx, pkgdir, func(dir string) error {
		prog := filepath.Join(dir, "fab")
		return exec.CommandContext(ctx, prog, args...).Run()
	})
}
