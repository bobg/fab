package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/bobg/fab"
)

func main() {
	var (
		pkgdir  string
		binfile string
		dbfile  string
		verbose bool
		list    bool
		force   bool
	)
	flag.StringVar(&pkgdir, "d", "fab.d", "directory containing fab rules")
	flag.StringVar(&binfile, "bin", "fab.bin", "path of executable file to create/run")
	flag.StringVar(&dbfile, "db", "", "path to Sqlite3 hash database file")
	flag.BoolVar(&verbose, "v", false, "run verbosely")
	flag.BoolVar(&list, "list", false, "list available targets")
	flag.BoolVar(&force, "f", false, "force compilation of -bin executable")
	flag.Parse()

	m := fab.Main{
		Pkgdir:  pkgdir,
		Binfile: binfile,
		DBFile:  dbfile,
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
