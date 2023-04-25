package _fab

import (
	"os"

	"github.com/bobg/fab"
)

// Build runs "go build".
var Build = fab.Command("go build ./...", fab.CmdStdout(os.Stdout))

// Test runs "go test" with the race detector enabled, plus coverage reporting.
var Test = fab.Command("go test -race -cover ./...", fab.CmdStdout(os.Stdout))

// Lint runs staticcheck.
var Lint = fab.Command("golangci-lint run ./...", fab.CmdStdout(os.Stdout))

// Vet runs "go vet".
var Vet = fab.Command("go vet ./...", fab.CmdStdout(os.Stdout))

// Check runs all of Vet, Lint, and Test.
var Check = fab.All(Vet, Lint, Test)

// Cover produces a test-coverage profile and opens it in a browser.
var Cover = fab.Seq(
	fab.Command("go test -coverprofile cover.out ./...", fab.CmdStdout(os.Stdout)),
	fab.Command("go tool cover -html cover.out"),
)
