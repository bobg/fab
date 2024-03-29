package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bobg/fab"
	_ "github.com/bobg/fab/golang"
)

func main() {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home dir: %s\n", err)
			os.Exit(1)
		}
		cacheDir = filepath.Join(home, ".cache")
	}

	var (
		fabdir  string
		verbose bool
		list    bool
		force   bool
		dryrun  bool
	)
	flag.StringVar(&fabdir, "fab", filepath.Join(cacheDir, "fab"), "directory containing fab DB and compiled drivers")
	flag.BoolVar(&verbose, "v", false, "run verbosely")
	flag.BoolVar(&list, "list", false, "list available targets")
	flag.BoolVar(&force, "f", false, "force compilation of -bin executable")
	flag.BoolVar(&dryrun, "n", false, "dry run mode")
	flag.Parse()

	m := fab.Main{
		Fabdir:  fabdir,
		Verbose: verbose,
		List:    list,
		Force:   force,
		DryRun:  dryrun,
		Args:    flag.Args(),
	}
	if err := m.Run(context.Background()); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}
