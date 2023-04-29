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
		pkgdir  string
		fabdir  string
		chdir   string
		verbose bool
		list    bool
		force   bool
	)
	flag.StringVar(&pkgdir, "pkg", "_fab", "directory containing Go package of build rules")
	flag.StringVar(&fabdir, "fab", filepath.Join(cacheDir, "fab"), "directory containing fab DB and compiled drivers")
	flag.StringVar(&chdir, "C", "", "chdir to this directory on startup")
	flag.BoolVar(&verbose, "v", false, "run verbosely")
	flag.BoolVar(&list, "list", false, "list available targets")
	flag.BoolVar(&force, "f", false, "force compilation of -bin executable")
	flag.Parse()

	m := fab.Main{
		Pkgdir:  pkgdir,
		Fabdir:  fabdir,
		Chdir:   chdir,
		Verbose: verbose,
		List:    list,
		Force:   force,
		Args:    flag.Args(),
	}
	if err := m.Run(context.Background()); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}
