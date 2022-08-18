# Fab - software fabricator

[![Go Reference](https://pkg.go.dev/badge/github.com/bobg/fab.svg)](https://pkg.go.dev/github.com/bobg/fab)
[![Go Report Card](https://goreportcard.com/badge/github.com/bobg/fab)](https://goreportcard.com/report/github.com/bobg/fab)
[![Tests](https://github.com/bobg/fab/actions/workflows/go.yml/badge.svg)](https://github.com/bobg/fab/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/bobg/fab/badge.svg?branch=master)](https://coveralls.io/github/bobg/fab?branch=master)

This is fab, a system for orchestrating software builds in Go.

Like Make,
fab executes recipes to turn inputs into outputs,
making sure to build prerequisites before the targets that depend on them,
and avoiding recompilation of targets that don’t need it.

Unlike Make, recipes are written in Go.
(Which is not to say that what you’re building has to be in Go; it doesn’t.)

## Usage

You create a package of Go code in your project.
By default the `fab` program looks for the package in the `fab.d` subdir.
Every exported symbol in that package
whose type satisfies the `fab.Target` interface
is a target that fab can run.

For example, if you write this in `fab.d/build.go`:

```go
package any_name_you_like

import (
	"os"

	"github.com/bobg/fab"
)

var (
	Build = &fab.Command{Shell: "go build ./..."}
	Vet   = &fab.Command{Shell: "go vet ./...", Stdout: os.Stdout}
	Test  = &fab.Command{Shell: "go test -race -cover ./...", Stdout: os.Stdout}
	Check = fab.All(Vet, Test)
)
```

you can then run `fab build`, `fab check`, etc. in the shell.

## Under the hood

When you run `fab`,
it compiles a new Go binary using your package of build rules and a custom `main` function.

## Why not Mage?

Fab was strongly inspired by [Mage](https://magefile.org/),
which has a similar feature set.
However, the author found Mage a little cumbersome for a few particular uses:

- Adding persistent hashes of targets
  to determine when running one can be skipped,
  because its outputs are already up to date
  with respect to its inputs.
- Propagating errors outward from within target implementations.
- Defining targets as the result of suitably typed expressions assigned to top-level vars,
  instead of having to be Go `func`s.
