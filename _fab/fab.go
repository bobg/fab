package _fab

import (
	"os"

	"github.com/bobg/fab"
	"github.com/bobg/fab/golang"
)

// Build builds the fab binary.
var Build = golang.MustBinary("cmd/fab", "fab")

// Test runs "go test" with the race detector enabled, plus coverage reporting.
var Test = fab.Shellf("go test -race -cover ./...")

// Lint runs staticcheck.
var Lint = &fab.Command{Shell: "golangci-lint run ./...", Stdout: os.Stdout}

// Vet runs "go vet".
var Vet = &fab.Command{Shell: "go vet ./...", Stdout: os.Stdout}

// Check runs all of Vet, Lint, and Test.
var Check = fab.All(Vet, Lint, Test)

// Cover produces a test-coverage profile and opens it in a browser.
var Cover = fab.Seq(
	fab.Shellf("go test -coverprofile cover.out ./..."),
	fab.Shellf("go tool cover -html cover.out"),
)

var Clean = fab.Clean("fab", "cover.out")
