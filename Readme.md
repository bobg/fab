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

To define build targets in your software project,
write Go code in a `_fab` subdirectory
and/or write a `fab.yaml` file.
(See [Targets](#Targets) below.)

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
This name prevents the code in that directory
from being considered as part of the public API for your project.
([Citation](https://pkg.go.dev/go/build#Context.Import).)

You can use any package name for the code in directory `_fab`;
the official suggestion is `_fab`,
to match the directory name.

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
using the [fab.RegisterTarget](https://pkg.go.dev/github.com/bobg/fab#RegisterTarget) function.

```go
for m := time.January; m <= time.December; m++ {
  fab.RegisterTarget(
    m.String(),
    "Say that it’s "+m.String(),
    fab.Command("echo It is "+m.String(), fab.CmdStdout(os.Stdout)),
  )
}
```

This creates targets named `January`, `February`, `March`, etc.

Internally, static target definition works by calling `fab.RegisterTarget`.

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
  Target: Build

# Unconditionally rebuild prog.
Build: !Command
  - go build -o prog ./cmd/prog

# Test runs all tests.
Test: !Command
  - go test -race -cover ./...
  - stdout
```

This defines `Prog` as a Fab target of type `Files`.
The `In` argument is the list of input files that may change
and the `Out` argument is the list of expected output files that result from running the nested subtarget.
That subtarget is a reference to the `Build` rule,
which is a `Command` that runs `go build`.
`In` is defined as the result of the [deps.Go](https://pkg.go.dev/github.com/bobg/fab/deps#Go) rule,
which produces the list of files on which the Go package in a given directory depends.

This also defines a `Test` target as a `Command` that runs `go test`.

## Defining new target types

You can define new target types in Go code in the `_fab` subdirectory
(or anywhere else, that is then imported into the `_fab` package).

Your type must implement [fab.Target](https://pkg.go.dev/github.com/bobg/fab#Target),
which requires two methods: `Desc` and `Run`.

`Desc` produces a short string describing the target.
It is used by [Describe](https://pkg.go.dev/github.com/bobg/fab#Describe)
to describe targets that don’t have a name
(i.e., ones that were never registered with `RegisterTarget`,
possibly because they are nested inside some other target).

`Run` should unconditionally execute your target type’s logic.
The Fab runtime will take care of making sure your target runs only when it needs to.
More about this appears below.

If part of your `Run` method involves running other targets,
do not invoke their `Run` methods directly.
Instead, invoke the [fab.Run](https://pkg.go.dev/github.com/bobg/fab#Run) _function_,
which will skip that target if it has already run.
This means that a “diamond dependency” —
A depends on B and C,
and B and C each separately depend on X —
won’t cause X to run twice when the user runs `fab A`.

If you would like your new target type to be usable in `fab.yaml`,
you must define a YAML parser for it.
This is done with [RegisterYAMLTarget](https://pkg.go.dev/github.com/bobg/fab#RegisterYAMLTarget),
which associates a `name` with a [YAMLTargetFunc](https://pkg.go.dev/github.com/bobg/fab#YAMLTargetFunc).
When the YAML tag `!name` is encountered in `fab.yaml`
(in a context where a target may be specified),
your function will be invoked to parse the YAML node.

## HashTarget

If your target type implements the interface [HashTarget](https://pkg.go.dev/github.com/bobg/fab#HashTarget),
it is handled specially.
`HashTarget` is the same as `Target` but adds a new method,
`Hash`.
When the Fab runtime wants to run a `HashTarget` it first invokes its `Hash` method,
then checks the result to see if it appears in a _hash database_.

- If it does,
  the target is considered up-to-date
  and succeeds trivially.
  Its `Run` method is skipped.
- If it doesn’t,
  the `Run` method executes
  and then Fab re-invokes the `Hash` method,
  placing the result into the hash database.

To understand `HashTarget`,
consider the [Files](https://pkg.go.dev/github.com/bobg/fab#Files) target type.
It specifies a set of input files,
a set of expected output files,
and a nested subtarget for producing one from the other.
`Files` implements `HashTarget`,
and its `Hash` method produces a hash from the content of all the input files,
all the output files,
_and_ the rules for the nested subtarget.
The first time this rule runs,
its hash won’t be present in the database,
so its `Run` method executes,
and then the hash is recomputed from the now up-to-date files
and placed in the database.

As long as none of the input files,
the output files,
or the build rules change,
`Hash` will produce the same value that was added to the database.
Subsequent runs of the `Files` target
will find that hash in the database and skip calling `Run`.

(This is a key difference between Fab and Make.
Make uses file modification times
to decide when a set of output files needs to be recomputed from their inputs.
Considering the limited resolution of filesystem timestamps,
the possibility of clock skew, etc.,
the content-based test that Fab uses is preferable.
But it would be easy to define a file-modtime-based target type in Fab
if that’s what you wanted.)

The hash database is stored in `$HOME/.cache/fab` by default,
and hash values normally expire after thirty days.

## The Fab runtime

A Fab [Runner](https://pkg.go.dev/github.com/bobg/fab#Runner)
is responsible for invoking targets’ `Run` methods,
keeping track of which ones have already run
so that they don’t get invoked a second time.
A normal Fab session uses a single global default runner.

The runner uses the address of each target as a unique key.
This means that pointer types should be used to implement `Target`.
After a target runs,
the runner records its outcome
(error or no error).
The second and subsequent attempts to run a given target
will use the previously computed outcome.

## Program startup

If you have Go code in a `_fab` subdirectory,
Fab combines it with its own `main` function to produce a _driver,_
which is an executable Go binary that is stored in `$HOME/.cache/fab` by default.

The driver calls [fab.RegisterTarget](https://pkg.go.dev/github.com/bobg/fab#RegisterTarget)
on each of the eligible top-level identifiers in your package;
then it looks for a `fab.yaml` file and registers the target definitions it finds there.
After that,
the driver runs the targets you specified on the `fab` command line
(or lists targets if you specified `-list`, etc).

When you run `fab` and the driver is already built and up to date
(as determined by a _hash_ of the code in the `_fab` dir),
then `fab` simply executes the driver without rebuilding it.
You can force a rebuild of the driver by specifying `-f` to `fab`.

If you do not have a `_fab` subdirectory,
then Fab operates in “driverless” mode,
in which the `fab.yaml` file is loaded
and the targets on the command line executed.

Note that when you have both a `_fab` subdirectory and a `fab.yaml` file,
you may use target types in the YAML file that are defined in your `_fab` package.
When you have only a `fab.yaml` file you are limited to the target types that are predefined in Fab.

## Why not Mage?

Fab was strongly inspired by the excellent [Mage](https://magefile.org/) tool,
which works similarly and has a similar feature set.
But Fab has some features the author needed and did not find in Mage:

- Errors from `Target` rules propagate out instead of causing an exit.
- Targets are values, not functions, and are composable (e.g. with `Seq`, `All`, and `Deps`).
- Rebuilding of up-to-date targets can be skipped based on file contents, not modtimes.
