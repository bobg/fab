package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/bobg/fab"
	"github.com/bobg/fab/sqlite"
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
	if dbfile != "" {
		db, err := sqlite.Open(ctx, dbfile, sqlite.Keep(30*24*time.Hour)) // keep db entries for 30 days
		if err != nil {
			log.Fatalf("Error opening %s: %s", dbfile, err)
		}
		defer db.Close()
		ctx = fab.WithHashDB(ctx, db)
	}

	err := fab.CompileAndRun(ctx, pkgdir, flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}
}
