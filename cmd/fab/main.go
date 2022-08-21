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
		verbose bool
		dbfile  string
	)
	flag.StringVar(&pkgdir, "d", "fab.d", "directory containing fab rules")
	flag.BoolVar(&verbose, "v", false, "run verbosely")
	flag.StringVar(&dbfile, "db", "", "path to Sqlite3 hash database file")
	flag.Parse()

	ctx := context.Background()
	ctx = fab.WithVerbose(ctx, verbose)

	var args []string
	if dbfile != "" {
		args = append(args, "-db", dbfile)
	}
	args = append(args, flag.Args()...)

	err := fab.CompileAndRun(ctx, pkgdir, args...)
	if err != nil {
		log.Fatal(err)
	}
}
