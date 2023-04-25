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

You will need an installation of Go version 1.20 or later.
Download Go here: [go.dev](https://go.dev/dl/).

Once Go is installed you can install Fab like this:

```sh
go install github.com/bobg/fab/cmd/fab@latest
```

To build targets in your software project,
run

```sh
fab TARGET1 TARGET2 ...
```

To see the progress of your build you can add the `-v` flag (for “verbose”):

```sh
fab -v TARGET1 TARGET2 ...
```

If you have a target that takes command-line parameters,
you can invoke it like this:

```sh
fab TARGET ARG1 ARG2 ...
```

In this form,
`ARG1` must start with a `-`,
and no other targets may be specified.

To see the available build targets in your project,
run

```sh
fab -list
```

## Targets

Each fab target has a _type_
that dictates the parameters required by the target (if any),
and the actions that the target should take when it runs.

Fab predefines several target types.
Here is a partial list:

- `Command` invokes a shell command when it runs.
- `F` invokes an arbitrary Go function.
- `Files` specifies a set of input files and a set of output files,
  and a nested subtarget that runs
  only when the outputs are out-of-date with respect to the inputs.
- `All` invokes a set of subtargets in parallel.
- `Seq` invokes subtargets in sequence.
- `Deps` invokes a subtarget only after its dependencies
  (other subtargets)
  have run.
- `Clean` deletes a set of files.

You define targets by instantiating one of these types,
supplying it with any necessary arguments,
and giving it a name.
There are three ways to do this:
statically in Go code;
dynamically in Go code;
and declaratively in a YAML file.
These options are discussed below.

You can also define new target types
by implementing the [fab.Target](https://pkg.go.dev/github.com/bobg/fab#Target) interface.

## Static target definition in Go

You can write Go code to define targets that Fab can run.
To do this,
create a subdirectory named `_fab` at the root of your project,
and create `.go` files in that directory.
You can use any package name;
the official suggestion is `_fab`
(to match the directory name).

Any exported identifiers at the top level of this package
whose type implements the [fab.Target](https://pkg.go.dev/github.com/bobg/fab#Target) interface
are usable Fab targets.
For example:

```go
package _fab

import (
    "os"

    "github.com/bobg/fab"
)

// Test runs tests.
var Test = fab.Command("go test -cover ./...", fab.CmdStdout(os.Stdout))
```

This creates a `Command`-typed target named `Test`,
which you can invoke with `fab Test`.
When it runs,
it executes the given shell command
and copies its output to fab’s standard output.

The comment above the variable declaration gets attached to the created target.
Running `fab -list` will show that comment as a docstring:

```sh
$ fab -list
Test
    Test runs tests.
```

## Dynamic target definition in Go

Not all targets are suitable for creation via top-level variable declarations.
Those that require more complex processing can be defined dynamically
using the [fab.Register](https://pkg.go.dev/github.com/bobg/fab#Register) function.

```go
for m := time.January; m <= time.December; m++ {
  fab.Register(
    m.String(),
    "Say that it’s "+m.String(),
    fab.Command("echo It is "+m.String(), fab.CmdStdout(os.Stdout)),
  )
}
```

This creates targets named `January`, `February`, `March`, etc.

Internally, static target definition works by calling `fab.Register`.

## Declarative target definition in YAML

In addition to Go code in the `_fab` subdirectory,
or instead of it,
you can define targets in a `fab.yaml` file
at the top level of your project.

The top-level structure of the YAML file is a mapping from names to targets.
Targets are specified using YAML type tags.
Most Fab target types define a tag and a syntax for extracting necessary arguments from YAML.

Targets may also be referred to by name.

Here is an example `fab.yaml` file:

```yaml
# Prog rebuilds prog if any of the files it depends on changes.
Prog: !Files
  In: !deps.Go
    Dir: cmd/prog
  Out:
    - prog
  Target: !Command
    - go build -o prog ./cmd/prog

# Test runs all tests.
Test: !Command
  - go test -race -cover ./...
  - stdout
```

This defines `Prog` as a Fab target of type `Files`.
The `In` argument is the list of input files that may change
and the `Out` argument is the list of expected output files that result from running the nested subtarget.
That subtarget is a `Command` that runs `go build`.
`In` is defined as the result of the [deps.Go](https://pkg.go.dev/github.com/bobg/fab/deps#Go) rule,
which produces the list of files on which the Go package in a given directory depends.

This also defines a `Test` target as a `Command` that runs `go test`.

## Defining new target types

You can define new target types in Go code in the `_fab` subdirectory
(or anywhere else, that is then imported into the `_fab` package).

Your type must implement [fab.Target](https://pkg.go.dev/github.com/bobg/fab#Target),
which requires three methods: `Run`, `Name`, and `SetName`.

To implement `Name` and `SetName`,
your type _should_ embed a [*fab.Namer](https://pkg.go.dev/github.com/bobg/fab#Namer).
More about this appears below.

This just leaves `Run`,
which should unconditionally execute your target type’s logic.
The Fab runtime will take care of making sure your target runs only when it needs to.
More about this appears below, too.

If part of your `Run` method involves running other targets,
do not invoke their `Run` methods directly.
Instead, invoke the [fab.Run](https://pkg.go.dev/github.com/bobg/fab#Run) _function_,
which will skip that target if it has already run.
This means that a “diamond dependency” —
A depends on B and C,
and B and C each separately depend on X —
won’t cause X to run twice when the user runs `fab A`.

## Namer

Every target defined during a run of Fab must have a unique name.
Fab uses that name to record whether the target has or hasn’t yet run.











There are multiple ways to specify build targets:
statically in Go code;
dynamically in Go code;
and declaratively in a YAML file.







You create a package of Go code in your project.
By default `fab` looks for it in the `_fab` subdir of your top-level directory
(so named because the leading underscore prevents it being considered part of your module’s public API).
Every exported symbol in that package
whose type satisfies the `fab.Target` interface
is a target that fab can run.

For example, if you write this in `_fab/build.go`:

```go
package any_name_you_like

import (
  "os"

  "github.com/bobg/fab"
)

// Build builds all available Go targets.
var Build = fab.Command("go build ./...")

// Vet runs “go vet” on all available Go targets.
var Vet = fab.Command("go vet ./...", fab.CmdStdout(os.Stdout))

// Test runs “go test” on all available Go targets.
var Test = fab.Command("go test -race -cover ./...", fab.CmdStdout(os.Stdout))

// Check runs the Vet and Test checks.
var Check = fab.All(Vet, Test)
```

then you can run `fab Build`, `fab Check`, etc. in the shell.

This is “static target registration.”
The name of the variable is used as the name of the target,
and the variable’s doc comment is used as the target’s documentation string.
You can make additional targets available by registering them “dynamically”
using calls to `fab.Register` during program initialization
(e.g. in `init()` functions).
Internally, calling `fab.Register` is how static registration works too.

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
  *fab.Namer
  dependencies []fab.Target
}

func (tt *myTargetType) Run(ctx context.Context) error {
  if err := fab.Run(ctx, tt.dependencies...); err != nil {
    return err
  }
  // ...other myTargetType build logic...
}
```

(Here, `*fab.Namer` is an embedded field
that supplies the needed behavior
for the Target interface’s `Name` and `SetName` methods.)

Fab ensures that no target runs more than once during a build,
no matter how many times that target shows up in other targets’ dependencies
or calls to `Run`, etc.

The fab command line can list multiple targets, e.g. `fab Vet Test Build`,
or a single target plus arguments, e.g. `fab Build -verbose`,
depending on whether the first string after the first target starts with `-`.
In argument-passing mode,
the named target is wrapped with `ArgTarget`
and the arguments are available at runtime using `GetArgs`.

## Details

By default, your build rules are found in the `_fab` subdir.
Running `fab` combines your rules with its own `main` function to produce a _driver_,
which lives in `$HOME/.cache/fab` by default.
(These defaults can be overridden.)

When you run `fab` and the driver is already present and up to date
(as determined by a _hash_ of the code in the `_fab` dir),
then `fab` simply executes the driver without rebuilding it.

The directory `$HOME/.cache/fab` also contains a _hash database_
to tell when certain targets -
those satisfying the `HashTarget` interface -
are up to date and do not need rebuilding.
When a `HashTarget` runs,
it first computes a hash representing the complete state of the target -
all inputs, outputs, and build rules.
If that hash is in the database,
the target is considered up to date and `fab` skips the build rule.
Otherwise, the build rule runs,
and the hash is recomputed and added to the database.
This approach is preferable to using file modification times
(like Make does, for example)
to know when a target is up to date.
Those aren’t always sufficient for this purpose,
nor are they entirely reliable,
considering the limited resolution of filesystem timestamps,
the possibility of clock skew, etc.

## Installation

Fab requires Go 1.20 or later.
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
