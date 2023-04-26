package _fab

import (
	"github.com/bobg/fab"
)

// Build runs "go build".
var Build = fab.Shellf("go build ./...")

// Test runs "go test" with the race detector enabled, plus coverage reporting.
var Test = fab.Shellf("go test -race -cover ./...")

// Lint runs staticcheck.
var Lint = fab.Shellf("golangci-lint run ./...")

// Vet runs "go vet".
var Vet = fab.Shellf("go vet ./...")

// Check runs all of Vet, Lint, and Test.
var Check = fab.All(Vet, Lint, Test)

// Cover produces a test-coverage profile and opens it in a browser.
var Cover = fab.Seq(
	fab.Shellf("go test -coverprofile cover.out ./..."),
	fab.Shellf("go tool cover -html cover.out"),
)
