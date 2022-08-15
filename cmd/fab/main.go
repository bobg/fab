package main

import (
	"context"
	"flag"
	"log"

	"github.com/bobg/fab"
	"github.com/bobg/fab/internal"
)

func main() {
	var (
		pkgdir  string
		verbose bool
	)
	flag.StringVar(&pkgdir, "d", ".fab", "directory containing fab rules")
	flag.BoolVar(&verbose, "v", false, "run verbosely")
	flag.Parse()

	ctx := context.Background()
	ctx = fab.WithVerbose(ctx, verbose)

	err := internal.Run(ctx, pkgdir, flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}
}
