package build

import (
	"os"

	"github.com/bobg/fab"
)

// Build runs "go build".
var Build = &fab.Command{
	Shell:  "go build ./...",
	Stdout: os.Stdout,
}

// Test runs "go test" with the race detector enabled, plus coverage reporting.
var Test = &fab.Command{
	Shell:  "go test -race -cover ./...",
	Stdout: os.Stdout,
}

// Lint runs staticcheck.
var Lint = &fab.Command{
	Shell:  "staticcheck ./...",
	Stdout: os.Stdout,
}

// Vet runs "go vet".
var Vet = &fab.Command{
	Shell:  "go vet ./...",
	Stdout: os.Stdout,
}

// Check runs all of Vet, Lint, and Test.
var Check = fab.All(Vet, Lint, Test)
