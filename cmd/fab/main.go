package main

import (
	"context"
	"flag"
	"log"

	"github.com/bobg/fab"
	"github.com/bobg/fab/internal"
	"github.com/bobg/fab/sqlite"
)

func main() {
	var (
		pkgdir  string
		verbose bool
		dbfile  string
	)
	flag.StringVar(&pkgdir, "d", ".fab", "directory containing fab rules")
	flag.BoolVar(&verbose, "v", false, "run verbosely")
	flag.StringVar(&dbfile, "db", "", "path to Sqlite3 hash database file")
	flag.Parse()

	ctx := context.Background()
	ctx = fab.WithVerbose(ctx, verbose)
	if dbfile != "" {
		db, err := sqlite.Open(ctx, dbfile)
		if err != nil {
			log.Fatalf("Error opening %s: %s", dbfile, err)
		}
		defer db.Close()
		ctx = fab.WithHashDB(ctx, db)
	}

	err := internal.Run(ctx, pkgdir, flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}
}
