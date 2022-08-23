package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bobg/fab"
)

func main() {
	var (
		pkgdir  string
		verbose bool
		dbfile  string
		binfile string
		ctx     = context.Background()
	)
	flag.StringVar(&pkgdir, "d", "fab.d", "directory containing fab rules")
	flag.BoolVar(&verbose, "v", false, "run verbosely")
	flag.StringVar(&dbfile, "db", "", "path to Sqlite3 hash database file")
	flag.StringVar(&binfile, "bin", "fab.bin", "path of executable file to create/run")
	flag.Parse()

	var args []string
	if verbose {
		args = append(args, "-v")
	}
	if dbfile != "" {
		args = append(args, "-db", dbfile)
	}

	info, err := os.Stat(binfile)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if verbose {
			fmt.Printf("Compiling %s\n", binfile)
		}
		if err := fab.Compile(ctx, pkgdir, binfile); err != nil {
			log.Fatalf("Compiling %s: %s", binfile, err)
		}
		args = append(args, "-nocheck")

	case err != nil:
		log.Fatalf("Statting %s: %s", binfile, err)

	case info.IsDir():
		log.Fatalf("Will not clobber %s, which is a directory", binfile)

	case info.Mode().Perm()&1 == 0:
		log.Fatalf("File %s exists but is not world-executable", binfile)

	default:
		if verbose {
			fmt.Printf("Using existing %s\n", binfile)
		}
	}

	args = append(args, flag.Args()...)

	abs, err := filepath.Abs(binfile)
	if err != nil {
		log.Fatalf("Computing absolute pathname for %s", binfile)
	}
	cmd := exec.CommandContext(ctx, abs, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err = cmd.Run(); err != nil {
		log.Fatalf("Running %s %s: %s", abs, strings.Join(args, " "), err)
	}
}
