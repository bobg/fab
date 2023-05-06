package main

import (
	"embed"
	"io"
	"os"
)

//go:embed data
var fs embed.FS

func main() {
	f, err := fs.Open("data/file")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	io.Copy(os.Stdout, f)
}
