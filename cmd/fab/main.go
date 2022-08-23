package main

import (
	"context"
	"flag"
	"log"

	"github.com/bobg/fab"
)

func main() {
	var (
		pkgdir  string
		binfile string
		dbfile  string
		verbose bool
		force   bool
	)
	flag.StringVar(&pkgdir, "d", "fab.d", "directory containing fab rules")
	flag.StringVar(&binfile, "bin", "fab.bin", "path of executable file to create/run")
	flag.StringVar(&dbfile, "db", "", "path to Sqlite3 hash database file")
	flag.BoolVar(&verbose, "v", false, "run verbosely")
	flag.BoolVar(&force, "f", false, "force compilation of -bin executable")
	flag.Parse()

	m := fab.Main{
		Pkgdir:  pkgdir,
		Binfile: binfile,
		DBFile:  dbfile,
		Verbose: verbose,
		Force:   force,
		Args:    flag.Args(),
	}
	if err := m.Run(context.Background()); err != nil {
		log.Fatalf("Error: %s", err)
	}
}
