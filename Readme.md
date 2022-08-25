# Fab - software fabricator

[![Go Reference](https://pkg.go.dev/badge/github.com/bobg/fab.svg)](https://pkg.go.dev/github.com/bobg/fab)
[![Go Report Card](https://goreportcard.com/badge/github.com/bobg/fab)](https://goreportcard.com/report/github.com/bobg/fab)
[![Tests](https://github.com/bobg/fab/actions/workflows/go.yml/badge.svg)](https://github.com/bobg/fab/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/bobg/fab/badge.svg?branch=main)](https://coveralls.io/github/bobg/fab?branch=main)

This is fab,
a system for orchestrating software builds.
It’s like [Make](https://en.wikipedia.org/wiki/Make_(software)),
except that you express build rules and dependencies as [Go](https://go.dev/) code.

(But that doesn’t mean it’s for building Go programs only.
After all, writing shell commands in a Makefile doesn’t mean Make is for shell programs only.)

Running `fab` on one or more targets ensures that the targets’ prerequisites,
and the targets themselves,
are up to date according to your build rules,
while avoiding unnecessarily rebuilding any target that is already up to date.

## Usage

You create a package of Go code in your project.
By default `fab` looks for it in the `fab.d` subdir of your top-level directory.
Every exported symbol in that package
whose type satisfies the `fab.Target` interface
is a target that fab can run.

Target names that appear in CamelCase in Go code
are converted to snake_case for use by the fab command.

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

then you can run `fab build`, `fab check`, etc. in the shell.

To express a dependency between targets, use the `Deps` construct:

```go
// MyTarget ensures that pre1, pre2, etc. are built before post
// (each of which is some form of Target).
var MyTarget = fab.Deps(post, pre1, pre2, ...)
```

Alternatively,
you can define your own type satisfying the `Target` interface,
and express dependencies by calling the `Run` function in your type’s `Run` method:

```go
type myTargetType struct {
  dependencies []fab.Target
  id           string
}

func (tt *myTargetType) Run(ctx, context.Context) error {
  if err := fab.Run(ctx, tt.dependencies...); err != nil {
    return err
  }
  // ...other myTargetType build logic...
}

// Each instance of any Target type must have a persistent, distinct ID.
// The fab.ID function can help with this.
func (tt *myTargetType) ID() string {
  if tt.id == "" {
    tt.id = fab.ID("MyTargetType”)
  }
  return tt.id
}
```

Fab ensures that no target runs more than once during a build,
no matter how many times that target shows up in other targets’ dependencies
or calls to `Run`, etc.

## Details

Running `fab` combines your build rules with its own `main` function to produce a _driver_,
which is an executable binary that performs the actual execution of your rules.
By default, your rules are found in the `fab.d` subdir and the driver is named `fab.bin`
(but these can be overridden).

When you run `fab` and `fab.bin` is already present and up to date
(see next paragraph)
then `fab` simply executes the driver without rebuilding it.

The `fab.bin` driver embeds a _hash_ of the code in the `fab.d` dir.
At startup it can check to see whether that hash has changed.
If it has, then the driver is out of date with respect to your build rules.
In that case `fab.bin` will automatically recompile, replace, and rerun itself.

You can also specify a _hash database_ to fab.
When certain targets run -
those satisfying the `HashTarget` interface -
a hash representing the complete state of the target
(all inputs, outputs, and build rules)
is stored in the database.
The next time you want to build that target,
if it’s already up to date -
because the current state hashes to a value already in the database -
fab can skip recompilation.

## Installation

Fab requires Go 1.19 or later.
Download Go [here](https://go.dev/dl/).

Once a suitable version of Go is available
(you can check by running `go version`),
install Fab with:

```sh
go install github.com/bobg/fab/cmd/fab@latest
```

## Why not Mage?

Fab was strongly inspired by [Mage](https://magefile.org/),
which works similarly and has a similar feature set.
However, the author found Mage a little cumbersome for a few particular uses:

- Adding persistent hashes of targets
  to determine when running one can be skipped,
  because its outputs are already up to date
  with respect to its inputs.
- Propagating errors outward from within target implementations.
- Defining targets as the result of suitably typed expressions assigned to top-level vars,
  instead of having to be Go `func`s.
