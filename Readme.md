# Fab - software fabricator

[![Go Reference](https://pkg.go.dev/badge/github.com/bobg/fab.svg)](https://pkg.go.dev/github.com/bobg/fab)
[![Go Report Card](https://goreportcard.com/badge/github.com/bobg/fab)](https://goreportcard.com/report/github.com/bobg/fab)
[![Tests](https://github.com/bobg/fab/actions/workflows/go.yml/badge.svg)](https://github.com/bobg/fab/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/bobg/fab/badge.svg?branch=main)](https://coveralls.io/github/bobg/fab?branch=main)

This is fab,
a system for orchestrating software builds.
It’s like [Make](https://en.wikipedia.org/wiki/Make_(software)),
except that you express build rules and dependencies as [Go](https://go.dev/) code.

(But that doesn’t mean it’s for building Go programs only,
any more than writing shell commands in a Makefile means Make builds only shell programs.)

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

then you can run `fab Build`, `fab Check`, etc. in the shell.

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

By default, your build rules are found in the `fab.d` subdir.
Running `fab` combines your rules with its own `main` function to produce a _driver_,
which lives in `$HOME/.fab` by default.
(These defaults can be overridden.)

When you run `fab` and the driver is already present and up to date
(as determined by a _hash_ of the code in the `fab.d` dir),
then `fab` simply executes the driver without rebuilding it.

The directory `$HOME/.fab` also contains a _hash database_
to tell when certain targets -
those satisfying the `HashTarget` interface -
are up to date and do not need rebuilding.
When a `HashTarget` runs,
it first computes a hash representing the complete state of the target -
all inputs, outputs, and build rules.
If that hash is in the database,
the target is considered up to date and `fab` skips the build rule.
(Otherwise, the build rule runs,
and the hash is recomputed and added to the database.)
This approach is preferable to using file modification times
(like Make does, for example)
to know when a target is up to date.
Those aren’t always sufficient for this purpose,
nor are they entirely reliable,
considering the limited resolution of filesystem timestamps,
the possibility of clock skew, etc.

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

Fab was strongly inspired by the excellent [Mage](https://magefile.org/) tool,
which works similarly and has a similar feature set.
But Fab has some features the author needed and did not find in Mage:

- Errors from `Target` rules propagate out instead of causing an exit.
- Targets are values, not functions, and are composable (e.g. with `Seq`, `All`, and `Deps`).
- Rebuilding of up-to-date targets can be skipped based on file contents, not modtimes.
