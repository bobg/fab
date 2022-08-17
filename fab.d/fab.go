package build

import "github.com/bobg/fab"

var Build = fab.Named("Build", &fab.Command{
	Shell: "go build ./...",
	Verbose: true,
})

var Test = fab.Named("Test", &fab.Command{
	Shell:   "go test -race -cover ./...",
	Verbose: true,
})

var Lint = fab.Named("Lint", &fab.Command{
	Shell:   "staticcheck ./...",
	Verbose: true,
})
