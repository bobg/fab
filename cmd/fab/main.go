package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bobg/fab"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home dir: %s\n", err)
		os.Exit(1)
	}

	var (
		pkgdir  string
		fabdir  string
		verbose bool
		list    bool
		force   bool
	)
	flag.StringVar(&pkgdir, "pkg", "fab.d", "directory containing Go package of build rules")
	flag.StringVar(&fabdir, "fab", filepath.Join(home, ".fab"), "directory containing fab DB and compiled drivers")
	flag.BoolVar(&verbose, "v", false, "run verbosely")
	flag.BoolVar(&list, "list", false, "list available targets")
	flag.BoolVar(&force, "f", false, "force compilation of -bin executable")
	flag.Parse()

	m := fab.Main{
		Pkgdir:  pkgdir,
		Fabdir:  fabdir,
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
