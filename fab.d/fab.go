package build

import (
	"os"

	"github.com/bobg/fab"
)

var Build = fab.Named("Build", &fab.Command{
	Shell:  "go build ./...",
	Stdout: os.Stdout,
})

var Test = fab.Named("Test", &fab.Command{
	Shell:  "go test -race -cover ./...",
	Stdout: os.Stdout,
})

var Lint = fab.Named("Lint", &fab.Command{
	Shell:  "staticcheck ./...",
	Stdout: os.Stdout,
})
