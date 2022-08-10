package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/bobg/fab/loader"
)

func main() {
	var pkgdir string
	flag.StringVar(&pkgdir, "d", ".fab", "directory containing fab rules")
	flag.Parse()

	err := loader.Run(context.Background(), pkgdir, os.Args[1:]...)
	if err != nil {
		log.Fatal(err)
	}
}
