package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/bobg/fab"
)

func main() {
	var pkgdir string
	flag.StringVar(&pkgdir, "d", ".fab", "directory containing fab rules")
	flag.Parse()

	err := fab.Run(context.Background(), pkgdir, os.Args[1:]...)
	if err != nil {
		log.Fatal(err)
	}
}
